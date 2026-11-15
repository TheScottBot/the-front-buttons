package editor

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// Copying ticked keys: in select mode each key on the grid is a check box,
// and the ticked keys are copied onto every page ticked, app screens among
// them, each on the tile it came from. Either every key goes onto every page,
// or nothing is copied and every tile in the way is named, so nothing is ever
// overwritten. Moving is the same copy, then the keys are taken off the page
// they came from; removing takes them off, and Undo puts them back. The form
// posts the ticked tiles as "row.column", so a later way of ticking, such as
// Shift and a click, only has to post the same.

// The fields of the copy, and the query that turns select mode on.
const (
	copyTileField   = "tile"
	copyTargetField = "to"
	selectQuery     = "select"
	copiedQuery     = "copied"
	copiedMessage   = "Copied. The deck shows the change within a second."
	movedQuery      = "moved"
	movedMessage    = "Moved. The deck shows the change within a second."
	removedQuery    = "removed"
	tileParts       = 2
)

// The reasons a copy of ticked keys is refused, each a whole sentence after
// "Error: ".
var (
	errCopyNoKeys    = errors.New("no keys were ticked; tick the keys to copy")
	errCopyNoTargets = errors.New("no page was ticked to copy onto; tick at least one")
	errCopyNoKeyAt   = errors.New("there is no key")
	errCopyBlocked   = errors.New("nothing was copied")
	errMoveBlocked   = errors.New("nothing was moved")
)

// copySelection is what a copy asks for: the page copied from, its ticked
// tiles, the pages ticked to copy onto, and how far each key shifts, in rows
// and columns, which is none from the form and as far as the key dragged when
// the keys are dragged together.
type copySelection struct {
	source  int
	tiles   []layout.Position
	targets []int
	shift   layout.Position
}

// The fields of a shift, and what one that is not a whole number reads as:
// off the deck.
const (
	rowShiftField    = "row_offset"
	columnShiftField = "column_offset"
	offDeckShift     = 1 << 16
)

// shiftFrom reads one part of a shift, none if it is left out.
func shiftFrom(value string) int {
	if value == "" {
		return 0
	}

	shift, parseError := strconv.Atoi(value)
	if parseError != nil {
		return offDeckShift
	}

	return shift
}

// shifted is where a key lands once shifted.
func (selection copySelection) shifted(position layout.Position) layout.Position {
	return layout.Position{Row: position.Row + selection.shift.Row, Column: position.Column + selection.shift.Column}
}

// selectionFrom reads a copy's form. A tile that is not "row.column" is read
// as off the deck, which has no key to copy.
func selectionFrom(request *http.Request) copySelection {
	form := request.PostForm
	selection := copySelection{
		source: integerParameter(form, "page"), tiles: nil, targets: nil,
		shift: layout.Position{Row: shiftFrom(form.Get(rowShiftField)), Column: shiftFrom(form.Get(columnShiftField))},
	}

	for _, value := range form[copyTileField] {
		position := tileFrom(value)
		if !slices.Contains(selection.tiles, position) {
			selection.tiles = append(selection.tiles, position)
		}
	}

	for _, value := range form[copyTargetField] {
		target, parseError := strconv.Atoi(value)
		if parseError != nil {
			target = -1
		}

		if !slices.Contains(selection.targets, target) {
			selection.targets = append(selection.targets, target)
		}
	}

	return selection
}

// tileFrom reads a ticked tile's "row.column".
func tileFrom(value string) layout.Position {
	parts := strings.Split(value, ".")
	offGrid := layout.Position{Row: -1, Column: -1}

	if len(parts) != tileParts {
		return offGrid
	}

	row, rowError := strconv.Atoi(parts[0])
	column, columnError := strconv.Atoi(parts[1])

	if rowError != nil || columnError != nil {
		return offGrid
	}

	return layout.Position{Row: row, Column: column}
}

// What may be done with ticked keys: copied onto the ticked pages, moved
// onto them, which is a copy that takes the keys off the page they came from,
// or removed.
type tickedAction int

const (
	copyTicked tickedAction = iota
	moveTicked
	removeTicked
)

// copyTickedKeys copies the ticked keys onto the ticked pages.
func (pages *pageHandlers) copyTickedKeys(writer http.ResponseWriter, request *http.Request) {
	pages.actOnTicked(writer, request, copyTicked)
}

// moveTickedKeys moves the ticked keys onto the ticked pages.
func (pages *pageHandlers) moveTickedKeys(writer http.ResponseWriter, request *http.Request) {
	pages.actOnTicked(writer, request, moveTicked)
}

// removeTickedKeys removes the ticked keys, which Undo puts back.
func (pages *pageHandlers) removeTickedKeys(writer http.ResponseWriter, request *http.Request) {
	pages.actOnTicked(writer, request, removeTicked)
}

// actOnTicked copies, moves or removes the ticked keys through the same save
// path as every other change, and returns to the page they came from.
func (pages *pageHandlers) actOnTicked(writer http.ResponseWriter, request *http.Request, action tickedAction) {
	selection := selectionFrom(request)

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		keys, checkError := pages.checkTicked(*configuration, selection, action)
		if checkError != nil {
			return checkError
		}

		// A move takes the keys off first, so a key shifted across its own
		// page may land on a tile another is leaving.
		if action != copyTicked {
			source := &configuration.Pages[selection.source]
			source.Keys = slices.DeleteFunc(source.Keys, func(key config.Key) bool {
				return slices.Contains(selection.tiles, key.Position)
			})
		}

		for _, target := range selection.targets {
			for _, key := range keys {
				landed := key.Clone()
				landed.Position = selection.shifted(key.Position)
				configuration.Pages[target].Keys = append(configuration.Pages[target].Keys, landed)
			}
		}

		return nil
	})
	if saveError == nil {
		saved := map[tickedAction]string{copyTicked: copiedQuery, moveTicked: movedQuery, removeTicked: removedQuery}
		// Keys pasted onto the page open return there; anything else returns to
		// the page the keys came from. The redirect's only variable parts are a
		// formatted integer and a constant.
		returnTo := selection.source
		if showing := integerParameter(request.PostForm, showingField); slices.Contains(selection.targets, showing) {
			returnTo = showing
		}

		address := fmt.Sprintf("/page?page=%d&saved=%s", returnTo, saved[action])
		http.Redirect(writer, request, address, http.StatusSeeOther) //nolint:gosec // see above

		return
	}

	messages := saveProblems(saveError)
	if isCopyRefusal(saveError) {
		messages = []string{errorPrefix + saveError.Error()}
	}

	configuration, _ := pages.configuration()
	pages.renderPageSelecting(writer, request, http.StatusUnprocessableEntity, configuration, selection, messages)
}

// checkTicked returns the ticked keys once the action is known to be
// possible: for a removal, that each is there; for a copy or move, that each
// fits on every ticked page too.
func (pages *pageHandlers) checkTicked(
	configuration config.Configuration, selection copySelection, action tickedAction,
) ([]config.Key, error) {
	if action == removeTicked {
		if !pageExists(configuration, selection.source) {
			return nil, errCopyNoPage
		}

		if len(selection.tiles) == 0 {
			return nil, errCopyNoKeys
		}

		return tickedKeys(configuration.Pages[selection.source], selection.tiles)
	}

	return pages.checkCopy(configuration, selection, action == moveTicked)
}

// isCopyRefusal reports a refusal explained in its own words.
func isCopyRefusal(saveError error) bool {
	return errors.Is(saveError, errCopyNoPage) || errors.Is(saveError, errCopySamePage) ||
		errors.Is(saveError, errCopyNoKeys) || errors.Is(saveError, errCopyNoTargets) ||
		errors.Is(saveError, errCopyNoKeyAt) || errors.Is(saveError, errCopyBlocked) ||
		errors.Is(saveError, errMoveBlocked)
}

// checkCopy returns the ticked keys once every one is known to fit on every
// ticked page.
func (pages *pageHandlers) checkCopy(
	configuration config.Configuration, selection copySelection, moving bool,
) ([]config.Key, error) {
	requestError := checkCopyRequest(configuration, selection)
	if requestError != nil {
		return nil, requestError
	}

	keys, keysError := tickedKeys(configuration.Pages[selection.source], selection.tiles)
	if keysError != nil {
		return nil, keysError
	}

	var blocked []string

	for _, target := range selection.targets {
		blocked = append(blocked, pages.blockedTiles(configuration.Pages, target, keys, selection, moving)...)
	}

	if blocked != nil {
		refusal := errCopyBlocked
		if moving {
			refusal = errMoveBlocked
		}

		return nil, fmt.Errorf("%w: %s. Remove or move what is in the way, or untick those keys or pages",
			refusal, strings.Join(blocked, "; "))
	}

	return keys, nil
}

// tickedKeys returns the keys at the ticked tiles, refusing a tile with no key
// or with the page marker, which each page keeps as its own.
func tickedKeys(page config.Page, tiles []layout.Position) ([]config.Key, error) {
	keys := make([]config.Key, 0, len(tiles))

	for _, position := range tiles {
		key, found := keyAt(page, position)
		if !found || brick.IsPageMarker(key.Kind) {
			return nil, fmt.Errorf("%w at %s; the page may have changed since it was opened",
				errCopyNoKeyAt, tileWords(position))
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// checkCopyRequest refuses a copy with nothing ticked, or from or onto a page
// that does not exist, or onto the page copied from.
func checkCopyRequest(configuration config.Configuration, selection copySelection) error {
	switch {
	case !pageExists(configuration, selection.source):
		return errCopyNoPage
	case len(selection.tiles) == 0:
		return errCopyNoKeys
	case len(selection.targets) == 0:
		return errCopyNoTargets
	}

	for _, target := range selection.targets {
		switch {
		case !pageExists(configuration, target):
			return errCopyNoPage
		case target == selection.source && selection.shift == layout.Position{Row: 0, Column: 0}:
			return errCopySamePage
		}
	}

	return nil
}

// pageExists reports whether a page index is in the file.
func pageExists(configuration config.Configuration, pageIndex int) bool {
	return pageIndex >= 0 && pageIndex < len(configuration.Pages)
}

// blockedTiles names every tile of a page a copied key cannot go on, once
// shifted: one off the deck, one holding a key, one kept for paging or the
// application's icon, or one on a page that does not offer the key's
// application's own brick. Keys moving across their own page leave their
// tiles free for each other.
func (pages *pageHandlers) blockedTiles(
	allPages []config.Page, target int, keys []config.Key, selection copySelection, moving bool,
) []string {
	page := allPages[target]
	where := pages.copyPlaceName(allPages, target)
	leaving := moving && target == selection.source

	var blocked []string

	for _, key := range keys {
		landing := selection.shifted(key.Position)
		leftFree := leaving && slices.Contains(selection.tiles, landing)

		if problem := pages.landingProblem(page, key.Kind, landing, leftFree); problem != "" {
			blocked = append(blocked, where+", "+tileWords(landing)+", "+problem)
		}
	}

	return blocked
}

// landingProblem says why a key of a kind cannot land on a tile of a page,
// or nothing if it can; a tile left free is one another moving key leaves.
func (pages *pageHandlers) landingProblem(
	page config.Page, kindName string, landing layout.Position, leftFree bool,
) string {
	switch {
	case !pages.options.Grid.Contains(landing):
		return "is off the deck"
	case pages.options.Grid.IsReserved(landing):
		return "is kept for paging"
	case pages.options.Grid.IsReservedOn(landing, page.IsAppScreen()):
		return "is kept for the application's icon"
	case !pages.kindOfferedOn(kindName, page):
		return "cannot hold " + kindName + ", which belongs on its application's screen"
	}

	if _, taken := keyAt(page, landing); taken && !leftFree {
		return "already has a key"
	}

	return ""
}

// copyPlaceName names a page in a copy's refusal: "the GoLand screen" for an
// app screen, else "page 2, Play", numbered among the regular pages.
func (pages *pageHandlers) copyPlaceName(allPages []config.Page, pageIndex int) string {
	page := allPages[pageIndex]
	if page.IsAppScreen() {
		return "the " + pages.screenName(allPages, pageIndex) + " screen"
	}

	return fmt.Sprintf("page %d, %s", slices.Index(regularPages(allPages), pageIndex)+1, page.Name)
}

// selectTarget is one page select mode offers to copy onto.
type selectTarget struct {
	Value   int
	Label   string
	Checked bool
}

// selectTargets lists every page but the one copied from, regular pages by
// their number and app screens by their application, ticking those asked for.
func (pages *pageHandlers) selectTargets(allPages []config.Page, selection copySelection) []selectTarget {
	regular := regularPages(allPages)
	targets := make([]selectTarget, 0, len(allPages))

	for pageIndex, page := range allPages {
		if pageIndex == selection.source {
			continue
		}

		label := pages.screenName(allPages, pageIndex) + " screen"
		if !page.IsAppScreen() {
			label = fmt.Sprintf("Page %d: %s", slices.Index(regular, pageIndex)+1, page.Name)
		}

		targets = append(targets, selectTarget{
			Value: pageIndex, Label: label, Checked: slices.Contains(selection.targets, pageIndex),
		})
	}

	return targets
}
