package editor

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The reasons a page copy is refused. Each is a whole sentence shown to the
// author after "Error: "; as with a single key, nothing is ever silently
// overwritten.
var (
	errCopyNoPage = errors.New(
		"that page no longer exists; the pages may have changed since this one was opened")
	errCopySamePage = errors.New("a page cannot be copied onto itself; choose another page, or copy to a new one")
	errCopyClash    = errors.New("the page copied onto already has keys")
)

// copyTarget is one page a page can be copied onto: its index, and its number
// and name as a person reads them.
type copyTarget struct {
	Value int
	Label string
}

// copyTargets lists every page but the one showing.
func copyTargets(configuration config.Configuration, pageIndex int) []copyTarget {
	var targets []copyTarget

	for index, page := range configuration.Pages {
		if index != pageIndex {
			targets = append(targets, copyTarget{Value: index, Label: fmt.Sprintf("Page %d: %s", index+1, page.Name)})
		}
	}

	return targets
}

// copyPageToNew copies every key of a page to a new page at the end, named as
// given, the author's request (evaluation, deck testing).
func (pages *pageHandlers) copyPageToNew(writer http.ResponseWriter, request *http.Request) {
	source := integerParameter(request.PostForm, "page")
	name := strings.TrimSpace(request.PostForm.Get("name"))

	var newIndex int

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if source < 0 || source >= len(configuration.Pages) {
			return errCopyNoPage
		}

		configuration.Pages = append(configuration.Pages, config.Page{
			Name: name, Keys: clonedKeys(configuration.Pages[source].Keys),
		})
		newIndex = len(configuration.Pages) - 1

		return nil
	})

	pages.afterCopy(writer, request, source, newIndex, saveError)
}

// copyPageOnto copies every key of a page onto another page, beside the keys
// already there. A key landing on a tile already holding one refuses the whole
// copy, naming each such tile, so the author chooses what goes.
func (pages *pageHandlers) copyPageOnto(writer http.ResponseWriter, request *http.Request) {
	source := integerParameter(request.PostForm, "page")
	destination := integerParameter(request.PostForm, "to")

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		switch {
		case source < 0 || source >= len(configuration.Pages) ||
			destination < 0 || destination >= len(configuration.Pages):
			return errCopyNoPage
		case source == destination:
			return errCopySamePage
		}

		target := &configuration.Pages[destination]

		clashError := clashes(configuration.Pages[source], *target, destination)
		if clashError != nil {
			return clashError
		}

		target.Keys = append(target.Keys, clonedKeys(configuration.Pages[source].Keys)...)

		return nil
	})

	pages.afterCopy(writer, request, source, destination, saveError)
}

// clashes names every tile where a copied key would land on a key already
// there, or returns nil if there is none.
func clashes(source config.Page, target config.Page, targetIndex int) error {
	var tiles []string

	for _, key := range source.Keys {
		if _, taken := keyAt(target, key.Position); taken {
			tiles = append(tiles, tileWords(key.Position))
		}
	}

	if tiles == nil {
		return nil
	}

	return fmt.Errorf("%w at %s; page %d, %s, is unchanged: remove those keys first, or copy to a new page",
		errCopyClash, strings.Join(tiles, " and "), targetIndex+1, target.Name)
}

// tileWords names a tile as a person counts it, from one.
func tileWords(position layout.Position) string {
	return fmt.Sprintf("row %d, column %d", position.Row+1, position.Column+1)
}

// clonedKeys deep copies keys, so a copied page shares no parameters with
// its original.
func clonedKeys(keys []config.Key) []config.Key {
	cloned := make([]config.Key, 0, len(keys))

	for _, key := range keys {
		cloned = append(cloned, key.Clone())
	}

	return cloned
}

// afterCopy shows the page copied to, or the page copied from again with why
// the copy was refused: in its own words for a refusal of the copy's, else as
// a failed save.
func (pages *pageHandlers) afterCopy(
	writer http.ResponseWriter,
	request *http.Request,
	source int,
	destination int,
	saveError error,
) {
	if saveError == nil {
		redirectToSavedPage(writer, request, destination)

		return
	}

	messages := saveProblems(saveError)
	if errors.Is(saveError, errCopyNoPage) || errors.Is(saveError, errCopySamePage) || errors.Is(saveError, errCopyClash) {
		messages = []string{errorPrefix + saveError.Error()}
	}

	configuration, _ := pages.configuration()
	pages.renderPage(writer, request, http.StatusUnprocessableEntity, configuration, source, messages)
}
