package editor

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// Reordering pages: a page's name dragged onto another's in the page list
// takes that page's place, the pages between moving along one. Regular pages
// and app screens are listed apart, so a page moves only among its own sort:
// the order of the regular pages is the order the deck pages through, and an
// application's screens are paged between in theirs. A page marker shows its
// page's number wherever the page goes. The form on each page does the same
// from the keyboard, and the script's drop posts the same fields.

// The reorder's fields and its message.
const (
	showingField     = "showing"
	reorderedQuery   = "reordered"
	reorderedMessage = "Page order saved. The deck shows the change within a second."
)

// The reasons a reorder is refused.
var (
	errReorderNoPage = errors.New("that page does not exist; the pages may have changed since this was opened")
	errReorderAcross = errors.New(
		"a page can only take the place of another of its own sort: a regular page among the regular pages, " +
			"an app screen among the app screens")
)

// reorderPlace is one place the keyboard's move offers.
type reorderPlace struct {
	Value    int
	Label    string
	Selected bool
}

// reorderPage moves a page into another's place and returns to the page on
// show, wherever it went.
func (pages *pageHandlers) reorderPage(writer http.ResponseWriter, request *http.Request) {
	form := request.PostForm
	from, destination, showing := integerParameter(form, "page"), integerParameter(form, copyTargetField),
		integerParameter(form, showingField)

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if !pageExists(*configuration, from) || !pageExists(*configuration, destination) {
			return errReorderNoPage
		}

		if configuration.Pages[from].IsAppScreen() != configuration.Pages[destination].IsAppScreen() {
			return errReorderAcross
		}

		configuration.Pages = movedPage(configuration.Pages, from, destination)

		return nil
	})
	if saveError != nil {
		pages.afterPageChange(writer, request, showing, saveError)

		return
	}

	// The redirect's only variable parts are a formatted integer and a
	// constant, so it can only ever name one of the editor's own pages.
	address := fmt.Sprintf("/page?page=%d&saved=%s", placeAfterMove(showing, from, destination), reorderedQuery)
	http.Redirect(writer, request, address, http.StatusSeeOther) //nolint:gosec // see above
}

// movedPage takes the page at from out and puts it in at destination.
func movedPage(allPages []config.Page, from int, destination int) []config.Page {
	moving := allPages[from]
	rest := slices.Delete(slices.Clone(allPages), from, from+1)

	return slices.Insert(rest, destination, moving)
}

// placeAfterMove is where the page at an index ends up once the page at from
// is moved to destination.
func placeAfterMove(pageIndex int, from int, destination int) int {
	switch {
	case pageIndex == from:
		return destination
	case from < destination && pageIndex > from && pageIndex <= destination:
		return pageIndex - 1
	case destination < from && pageIndex >= destination && pageIndex < from:
		return pageIndex + 1
	default:
		return pageIndex
	}
}

// reorderPlaces lists the places a page may move to: those of every page of
// its own sort, numbered from one, with the page there now named.
func reorderPlaces(allPages []config.Page, pageIndex int) []reorderPlace {
	page := pageAt(config.Configuration{SchemaVersion: 0, WorldClocks: nil, Pages: allPages}, pageIndex)
	places := make([]reorderPlace, 0, len(allPages))

	for index, candidate := range allPages {
		if candidate.IsAppScreen() != page.IsAppScreen() {
			continue
		}

		places = append(places, reorderPlace{
			Value: index, Label: fmt.Sprintf("Place %d, now %s", len(places)+1, candidate.Name), Selected: index == pageIndex,
		})
	}

	return places
}
