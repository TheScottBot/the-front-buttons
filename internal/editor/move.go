package editor

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The move form's two buttons, which share the field name "action".
const (
	moveKeyAction = "move"
	copyKeyAction = "copy"
)

// destinationParts is a destination's page, row and column.
const destinationParts = 3

// The reasons a move or copy is refused. Each is a whole sentence shown to the
// author after "Error: ", so none is ever silently swapped or overwritten.
var (
	errMoveNoSource = errors.New(
		"there is no key here to move or copy; the page may have changed since it was opened")
	errMoveSamePlace = errors.New("that destination is where the key already is; choose another tile")
	errMoveOffGrid   = errors.New(
		"that destination is not on the deck; choose a free tile from the list")
	errMoveReserved = errors.New("that destination is reserved for paging back or forward; choose another tile")
	errMoveOccupied = errors.New(
		"that destination already has a key; choose a free tile, or remove that key first")
)

// keyAddress is where a key sits: its page and its position on that page.
type keyAddress struct {
	page     int
	position layout.Position
}

// moveTile is one free tile in the destination list: its value is the page,
// row and column the file uses, joined by dots, and its label counts from one,
// as a person does.
type moveTile struct {
	Value string
	Label string
}

// moveGroup is one page's free tiles, under the page's number and name.
type moveGroup struct {
	Label string
	Tiles []moveTile
}

// moveView is the move or copy form: one list of every free tile on every
// page. The author found separate page, row and column lists offering the
// paging tiles, which the server then refused (evaluation, deck testing); a
// tile that cannot be used is now never offered. The server still checks.
type moveView struct {
	Groups       []moveGroup
	Focus        bool
	ReservedNote string
}

// moveChoices lists, page by page, every tile on the grid that paging does not
// hold, no key uses, and is not the key's own place.
func (pages *pageHandlers) moveChoices(context keyContext) moveView {
	grid := pages.options.Grid
	back, forward := grid.BackPosition(), grid.ForwardPosition()
	view := moveView{
		Groups: nil,
		Focus:  context.moveFailed && context.hasKey,
		ReservedNote: fmt.Sprintf("Only free tiles are listed: row %d, column %d and row %d, column %d are kept for paging.",
			back.Row+1, back.Column+1, forward.Row+1, forward.Column+1),
	}

	for pageIndex, page := range context.destinationPages {
		group := moveGroup{Label: fmt.Sprintf("Page %d: %s", pageIndex+1, page.Name), Tiles: nil}

		for row := range grid.Rows() {
			for column := range grid.Columns() {
				position := layout.Position{Row: row, Column: column}
				if grid.IsReserved(position) || (pageIndex == context.pageIndex && position == context.position) {
					continue
				}

				if _, taken := keyAt(page, position); taken {
					continue
				}

				group.Tiles = append(group.Tiles, moveTile{
					Value: fmt.Sprintf("%d.%d.%d", pageIndex, row, column),
					Label: fmt.Sprintf("Row %d, column %d", row+1, column+1),
				})
			}
		}

		if len(group.Tiles) > 0 {
			view.Groups = append(view.Groups, group)
		}
	}

	return view
}

// destinationFrom reads the destination list's value, page, row and column
// joined by dots. Anything else is a page that does not exist, which the checks
// refuse as not on the deck.
func destinationFrom(value string) keyAddress {
	parts := strings.Split(value, ".")
	if len(parts) != destinationParts {
		return keyAddress{page: -1, position: layout.Position{Row: -1, Column: -1}}
	}

	numbers := make([]int, 0, destinationParts)

	for _, part := range parts {
		number, parseError := strconv.Atoi(part)
		if parseError != nil {
			return keyAddress{page: -1, position: layout.Position{Row: -1, Column: -1}}
		}

		numbers = append(numbers, number)
	}

	return keyAddress{page: numbers[0], position: layout.Position{Row: numbers[1], Column: numbers[2]}}
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
	destination := destinationFrom(form.Get("to"))

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
