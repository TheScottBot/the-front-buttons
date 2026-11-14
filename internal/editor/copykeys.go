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
// overwritten. The form posts the ticked tiles as "row.column", so a later
// way of ticking, such as Shift and a click, only has to post the same.

// The fields of the copy, and the query that turns select mode on.
const (
	copyTileField   = "tile"
	copyTargetField = "to"
	selectQuery     = "select"
	copiedQuery     = "copied"
	copiedMessage   = "Copied. The deck shows the change within a second."
	tileParts       = 2
)

// The reasons a copy of ticked keys is refused, each a whole sentence after
// "Error: ".
var (
	errCopyNoKeys    = errors.New("no keys were ticked; tick the keys to copy")
	errCopyNoTargets = errors.New("no page was ticked to copy onto; tick at least one")
	errCopyNoKeyAt   = errors.New("there is no key")
	errCopyBlocked   = errors.New("nothing was copied")
)

// copySelection is what a copy asks for: the page copied from, its ticked
// tiles and the pages ticked to copy onto.
type copySelection struct {
	source  int
	tiles   []layout.Position
	targets []int
}

// selectionFrom reads a copy's form. A tile that is not "row.column" is read
// as off the deck, which has no key to copy.
func selectionFrom(request *http.Request) copySelection {
	form := request.PostForm
	selection := copySelection{source: integerParameter(form, "page"), tiles: nil, targets: nil}

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

// copyTickedKeys copies the ticked keys onto the ticked pages, through the
// same save path as every other change, and returns to the page copied from.
func (pages *pageHandlers) copyTickedKeys(writer http.ResponseWriter, request *http.Request) {
	selection := selectionFrom(request)

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		keys, checkError := pages.checkCopy(*configuration, selection)
		if checkError != nil {
			return checkError
		}

		for _, target := range selection.targets {
			for _, key := range keys {
				configuration.Pages[target].Keys = append(configuration.Pages[target].Keys, key.Clone())
			}
		}

		return nil
	})
	if saveError == nil {
		// The redirect's only variable part is a formatted integer.
		address := fmt.Sprintf("/page?page=%d&saved=%s", selection.source, copiedQuery)
		http.Redirect(writer, request, address, http.StatusSeeOther)

		return
	}

	messages := saveProblems(saveError)
	if isCopyRefusal(saveError) {
		messages = []string{errorPrefix + saveError.Error()}
	}

	configuration, _ := pages.configuration()
	pages.renderPageSelecting(writer, request, http.StatusUnprocessableEntity, configuration, selection, messages)
}

// isCopyRefusal reports a refusal explained in its own words.
func isCopyRefusal(saveError error) bool {
	return errors.Is(saveError, errCopyNoPage) || errors.Is(saveError, errCopySamePage) ||
		errors.Is(saveError, errCopyNoKeys) || errors.Is(saveError, errCopyNoTargets) ||
		errors.Is(saveError, errCopyNoKeyAt) || errors.Is(saveError, errCopyBlocked)
}

// checkCopy returns the ticked keys once every one is known to fit on every
// ticked page.
func (pages *pageHandlers) checkCopy(
	configuration config.Configuration, selection copySelection,
) ([]config.Key, error) {
	requestError := checkCopyRequest(configuration, selection)
	if requestError != nil {
		return nil, requestError
	}

	keys := make([]config.Key, 0, len(selection.tiles))

	for _, position := range selection.tiles {
		key, found := keyAt(configuration.Pages[selection.source], position)
		if !found || brick.IsPageMarker(key.Kind) {
			return nil, fmt.Errorf("%w at %s to copy; the page may have changed since it was opened",
				errCopyNoKeyAt, tileWords(position))
		}

		keys = append(keys, key)
	}

	var blocked []string

	for _, target := range selection.targets {
		blocked = append(blocked, pages.blockedTiles(configuration.Pages, target, keys)...)
	}

	if blocked != nil {
		return nil, fmt.Errorf("%w: %s. Remove or move what is in the way, or untick those keys or pages",
			errCopyBlocked, strings.Join(blocked, "; "))
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
		case target == selection.source:
			return errCopySamePage
		}
	}

	return nil
}

// pageExists reports whether a page index is in the file.
func pageExists(configuration config.Configuration, pageIndex int) bool {
	return pageIndex >= 0 && pageIndex < len(configuration.Pages)
}

// blockedTiles names every tile of a page a copied key cannot go on: one
// holding a key, one kept for paging or the application's icon, or one on a
// page that does not offer the key's application's own brick.
func (pages *pageHandlers) blockedTiles(allPages []config.Page, target int, keys []config.Key) []string {
	page := allPages[target]
	where := pages.copyPlaceName(allPages, target)

	var blocked []string

	for _, key := range keys {
		tile := where + ", " + tileWords(key.Position) + ", "

		switch {
		case pages.options.Grid.IsReserved(key.Position):
			blocked = append(blocked, tile+"is kept for paging")
		case pages.options.Grid.IsReservedOn(key.Position, page.IsAppScreen()):
			blocked = append(blocked, tile+"is kept for the application's icon")
		case !pages.kindOfferedOn(key.Kind, page):
			blocked = append(blocked, tile+"cannot hold "+key.Kind+", which belongs on its application's screen")
		default:
			if _, taken := keyAt(page, key.Position); taken {
				blocked = append(blocked, tile+"already has a key")
			}
		}
	}

	return blocked
}

// copyPlaceName names a page in a copy's refusal: "the GoLand screen" for an
// app screen, else "page 2, Play", numbered among the regular pages.
func (pages *pageHandlers) copyPlaceName(allPages []config.Page, pageIndex int) string {
	page := allPages[pageIndex]
	if page.IsAppScreen() {
		return "the " + pages.screenTitle(page) + " screen"
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

		label := pages.screenTitle(page) + " screen"
		if !page.IsAppScreen() {
			label = fmt.Sprintf("Page %d: %s", slices.Index(regular, pageIndex)+1, page.Name)
		}

		targets = append(targets, selectTarget{
			Value: pageIndex, Label: label, Checked: slices.Contains(selection.targets, pageIndex),
		})
	}

	return targets
}
