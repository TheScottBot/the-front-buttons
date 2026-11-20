package editor

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The navigation keys' colour: clicking a paging tile on the grid pages, as
// the deck does, so the tiles cannot be opened to edit like a key. A shortcut
// beside the grid opens a drawer under it instead, where one colour is chosen
// for both navigation keys on every page and app screen, and for the
// dashboard's paging tiles in the same corners, or taken out of the file for
// the usual black. It saves through the same path as every other change, so
// the store checks it and it can be undone.

// The navigation colour's query, which opens its drawer, and its message.
const (
	navigationQuery   = "navigation"
	navigationMessage = "Back and forward keys coloured. The deck shows the change within a second."
)

// navigationColourValue is the colour the drawer's colour input starts at:
// the navigation keys' own, or the usual black, in the lowercase a colour
// input gives back.
func navigationColourValue(configuration config.Configuration) string {
	colour := render.NavigationBackgroundFrom(configuration.NavigationBackground)

	return fmt.Sprintf("#%02x%02x%02x", colour.R, colour.G, colour.B)
}

// navigationDrawer is the drawer's part of a page's view: whether it is
// open, the colour it starts at, and whether the file names one.
func navigationDrawer(request *http.Request, configuration config.Configuration) (bool, string, bool) {
	return request.URL.Query().Get(navigationQuery) == "1", navigationColourValue(configuration),
		configuration.NavigationBackground != ""
}

// colourNavigationKeys sets the navigation keys' background, or with Back to
// the usual black takes it out of the file.
func (pages *pageHandlers) colourNavigationKeys(writer http.ResponseWriter, request *http.Request) {
	pageIndex := integerParameter(request.PostForm, "page")
	colour := strings.ToUpper(strings.TrimSpace(request.PostForm.Get(backgroundFieldName)))

	if request.PostForm.Get(clearColourField) != "" {
		colour = ""
	}

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		configuration.NavigationBackground = colour

		return nil
	})
	if saveError == nil {
		// The redirect's only variable parts are a formatted integer and
		// constants, so it can only ever name one of the editor's own pages.
		address := fmt.Sprintf("/page?page=%d&%s=1&saved=%s#navigation-colour",
			pageIndex, navigationQuery, navigationQuery)
		http.Redirect(writer, request, address, http.StatusSeeOther) //nolint:gosec // see above

		return
	}

	configuration, _ := pages.configuration()
	view := pages.pageViewFor(request, configuration, pageIndex, saveProblems(saveError))
	view.NavigationOpen = true

	pages.render(writer, http.StatusUnprocessableEntity, "page.html", view)
}
