package editor

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// An application may have several screens, in the order of the file; the
// deck pages between them with back and forward. An app screen offers to add
// another for its application, placed straight after its last, so an
// application's screens stay together. Where an application has more than
// one, each is named by its own page name.

// pagedScreens is how many screens an application needs before the deck pages
// between them rather than leaving them.
const pagedScreens = 2

// errNotAScreen refuses a screen added from a page that is not one.
var errNotAScreen = errors.New("that page is not an app screen; add a screen from one of the application's screens")

// screensOf lists an application's screens by their index in the file.
func screensOf(allPages []config.Page, application string) []int {
	var screens []int

	for pageIndex, page := range allPages {
		if page.IsAppScreen() && page.Application == application {
			screens = append(screens, pageIndex)
		}
	}

	return screens
}

// screenName names an app screen: by its application, such as GoLand, when it
// is the application's only screen, and by its own page name when it is one
// of several.
func (pages *pageHandlers) screenName(allPages []config.Page, pageIndex int) string {
	page := allPages[pageIndex]
	if len(screensOf(allPages, page.Application)) > 1 {
		return page.Name
	}

	return pages.screenTitle(page)
}

// screenCaption is the grid's caption for an app screen: which of its
// application's screens it is, when there are several.
func (pages *pageHandlers) screenCaption(allPages []config.Page, pageIndex int) string {
	page := allPages[pageIndex]
	screens := screensOf(allPages, page.Application)

	if len(screens) < pagedScreens {
		return "App screen for " + pages.screenTitle(page)
	}

	return fmt.Sprintf("Screen %d of %d for %s: %s",
		slices.Index(screens, pageIndex)+1, len(screens), pages.screenTitle(page), page.Name)
}

// addAppScreen adds another screen for the application of the screen it is
// asked from, straight after the application's last, and shows it.
func (pages *pageHandlers) addAppScreen(writer http.ResponseWriter, request *http.Request) {
	from := integerParameter(request.PostForm, "page")

	var added int

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if !pageExists(*configuration, from) || !configuration.Pages[from].IsAppScreen() {
			return errNotAScreen
		}

		application := configuration.Pages[from].Application
		screens := screensOf(configuration.Pages, application)
		name := pageNameCut(fmt.Sprintf("%s %d", pages.screenTitle(configuration.Pages[from]), len(screens)+1))

		added = screens[len(screens)-1] + 1
		configuration.Pages = slices.Insert(configuration.Pages, added,
			config.Page{Name: name, Application: application, Keys: nil})

		return nil
	})
	if saveError == nil {
		redirectToSavedPage(writer, request, added)

		return
	}

	messages := saveProblems(saveError)
	if errors.Is(saveError, errNotAScreen) {
		messages = []string{errorPrefix + saveError.Error()}
	}

	configuration, _ := pages.configuration()
	pages.renderPage(writer, request, http.StatusUnprocessableEntity, configuration, from, messages)
}

// pageNameCut cuts a name to a page name's length.
func pageNameCut(name string) string {
	runes := []rune(name)
	if len(runes) > config.MaximumPageNameCharacters {
		runes = runes[:config.MaximumPageNameCharacters]
	}

	return string(runes)
}

// screenStep is where an app screen's paging tile goes when its application
// has more than one screen: the screen a step away, wrapping round, by its
// number and name.
func screenStep(allPages []config.Page, pageIndex int, step int) (int, string, bool) {
	screens := screensOf(allPages, allPages[pageIndex].Application)
	showing := slices.Index(screens, pageIndex)

	if !allPages[pageIndex].IsAppScreen() || len(screens) < pagedScreens || showing < 0 {
		return 0, "", false
	}

	target := (showing + step + len(screens)) % len(screens)

	return screens[target], fmt.Sprintf("screen %d, %s", target+1, allPages[screens[target]].Name), true
}
