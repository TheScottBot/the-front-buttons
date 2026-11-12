package editor

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The move form's two buttons, which share the field name "action".
const (
	moveKeyAction = "move"
	copyKeyAction = "copy"
)

// The reasons a move or copy is refused. Each is a whole sentence shown to the
// author after "Error: ", so none is ever silently swapped or overwritten.
var (
	errMoveNoSource = errors.New(
		"there is no key here to move or copy; the page may have changed since it was opened")
	errMoveSamePlace = errors.New("that destination is where the key already is; choose another tile")
	errMoveOffGrid   = errors.New(
		"that destination is not on the deck; choose a page, row and column from the lists")
	errMoveReserved = errors.New("that destination is reserved for paging back or forward; choose another tile")
	errMoveOccupied = errors.New(
		"that destination already has a key; choose a free tile, or remove that key first")
)

// keyAddress is where a key sits: its page and its position on that page.
type keyAddress struct {
	page     int
	position layout.Position
}

// numberOption is one choice in a row or column list: the value counts from
// zero, as the file does, and the label from one, as a person does.
type numberOption struct {
	Value    int
	Label    int
	Selected bool
}

// movePage is one page in the destination list, numbered from one.
type movePage struct {
	Index   int
	Number  int
	Name    string
	Current bool
}

// moveView is the move or copy form: every page, row and column a key could go
// to, with the key's own place chosen to start with.
type moveView struct {
	Pages        []movePage
	Rows         []numberOption
	Columns      []numberOption
	Focus        bool
	ReservedNote string
}

// moveChoices builds the move form's lists from the configuration's pages and
// the grid's size, so a deck of another size gets its own rows and columns.
func (pages *pageHandlers) moveChoices(context keyContext) moveView {
	grid := pages.options.Grid
	back, forward := grid.BackPosition(), grid.ForwardPosition()
	view := moveView{
		Pages:   make([]movePage, 0, len(context.pageNames)),
		Rows:    numberOptions(grid.Rows(), context.position.Row),
		Columns: numberOptions(grid.Columns(), context.position.Column),
		Focus:   context.moveFailed && context.hasKey,
		ReservedNote: fmt.Sprintf("Row %d, column %d and row %d, column %d are kept for paging.",
			back.Row+1, back.Column+1, forward.Row+1, forward.Column+1),
	}

	for index, name := range context.pageNames {
		view.Pages = append(view.Pages,
			movePage{Index: index, Number: index + 1, Name: name, Current: index == context.pageIndex})
	}

	return view
}

func numberOptions(count int, selected int) []numberOption {
	options := make([]numberOption, 0, count)
	for value := range count {
		options = append(options, numberOption{Value: value, Label: value + 1, Selected: value == selected})
	}

	return options
}

// moveKey moves or copies a stored key to a free tile, through the same save
// path as every other change, so the store validates and writes it atomically
// and editing still waits while the file on disk is invalid.
func (pages *pageHandlers) moveKey(writer http.ResponseWriter, request *http.Request) {
	form := request.PostForm

	action := form.Get("action")
	if action != moveKeyAction && action != copyKeyAction {
		http.Error(writer, "unknown action", http.StatusBadRequest)

		return
	}

	source := keyAddress{
		page:     integerParameter(form, "page"),
		position: layout.Position{Row: integerParameter(form, "row"), Column: integerParameter(form, "column")},
	}
	destination := keyAddress{
		page:     integerParameter(form, "to_page"),
		position: layout.Position{Row: integerParameter(form, "to_row"), Column: integerParameter(form, "to_column")},
	}

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		return relocateKey(configuration, pages.options.Grid, source, destination, action == copyKeyAction)
	})
	if saveError != nil {
		pages.refuseMove(writer, request, source, saveError)

		return
	}

	redirectToSavedPage(writer, request, destination.page)
}

// refuseMove shows the key's page again with why the move was refused, and
// takes focus to the destination, the part the author has to change.
func (pages *pageHandlers) refuseMove(
	writer http.ResponseWriter,
	request *http.Request,
	source keyAddress,
	saveError error,
) {
	messages := saveProblems(saveError)
	if isMoveRefusal(saveError) {
		messages = []string{errorPrefix + saveError.Error()}
	}

	configuration, problems := pages.configuration()

	context, typeError := pages.storedKeyContext(configuration, source.page, source.position, "")
	if typeError != nil {
		http.Error(writer, "unknown key type", http.StatusUnprocessableEntity)

		return
	}

	context.errors, context.problems, context.moveFailed = messages, problems, true
	pages.renderKey(writer, request, http.StatusUnprocessableEntity, context)
}

// isMoveRefusal reports a refusal explained in its own words rather than as a
// failed save.
func isMoveRefusal(saveError error) bool {
	return errors.Is(saveError, errMoveNoSource) || errors.Is(saveError, errMoveSamePlace) ||
		errors.Is(saveError, errMoveOffGrid) || errors.Is(saveError, errMoveReserved) ||
		errors.Is(saveError, errMoveOccupied)
}

// relocateKey puts the key at a checked destination, removing it from the
// source for a move. The key is deep copied, so a copy and its original share
// no parameters.
func relocateKey(
	configuration *config.Configuration,
	grid layout.Grid,
	source keyAddress,
	destination keyAddress,
	copying bool,
) error {
	key, checkError := checkRelocation(*configuration, grid, source, destination)
	if checkError != nil {
		return checkError
	}

	if !copying {
		sourcePage := &configuration.Pages[source.page]
		sourcePage.Keys = slices.DeleteFunc(sourcePage.Keys, func(existing config.Key) bool {
			return existing.Position == source.position
		})
	}

	placed := key.Clone()
	placed.Position = destination.position

	destinationPage := &configuration.Pages[destination.page]
	destinationPage.Keys = append(destinationPage.Keys, placed)

	return nil
}

// checkRelocation returns the key at the source once the destination is known
// to be a different, free tile on the deck that paging does not hold.
func checkRelocation(
	configuration config.Configuration,
	grid layout.Grid,
	source keyAddress,
	destination keyAddress,
) (config.Key, error) {
	if source.page < 0 || source.page >= len(configuration.Pages) {
		return config.Key{Position: source.position, Kind: "", Parameters: nil, Preset: nil}, errMoveNoSource
	}

	key, found := keyAt(configuration.Pages[source.page], source.position)

	switch {
	case !found:
		return key, errMoveNoSource
	case destination == source:
		return key, errMoveSamePlace
	case destination.page < 0 || destination.page >= len(configuration.Pages) || !grid.Contains(destination.position):
		return key, errMoveOffGrid
	case grid.IsReserved(destination.position):
		return key, errMoveReserved
	}

	if _, taken := keyAt(configuration.Pages[destination.page], destination.position); taken {
		return key, errMoveOccupied
	}

	return key, nil
}
