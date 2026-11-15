package editor

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// removedMessage announces a removal; the page offers Undo beside it.
const removedMessage = "Removed. The deck shows the change within a second."

// The reasons an undo is refused, shown after the save error's prefix.
var (
	errNothingToUndo = errors.New("there is nothing to undo")
	errUndoTileTaken = errors.New("a tile a key came from now holds another key; move that one first")
	errUndoPageGone  = errors.New("the page the key came from no longer exists")
)

// removalMemory is the last removal's keys and the page they came from, so a
// key dropped in the bin by mistake, or keys ticked and removed, can be put
// back (the author's request for a bin that cannot easily be used by
// accident, evaluation, deck testing). Only the last removal is kept, in
// memory, until it is undone or replaced.
type removalMemory struct {
	mutex     sync.Mutex
	held      bool
	pageIndex int
	keys      []config.Key
}

func newRemovalMemory() *removalMemory {
	return &removalMemory{mutex: sync.Mutex{}, held: false, pageIndex: 0, keys: nil}
}

func (memory *removalMemory) remember(pageIndex int, keys []config.Key) {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	memory.held, memory.pageIndex, memory.keys = true, pageIndex, clonedAll(keys)
}

// clonedAll deep copies keys, page markers among them.
func clonedAll(keys []config.Key) []config.Key {
	cloned := make([]config.Key, 0, len(keys))
	for _, key := range keys {
		cloned = append(cloned, key.Clone())
	}

	return cloned
}

func (memory *removalMemory) holding() bool {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	return memory.held
}

func (memory *removalMemory) last() (int, []config.Key, bool) {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	return memory.pageIndex, clonedAll(memory.keys), memory.held
}

// follow keeps the removal's page as pages move, so Undo puts keys back on
// the page they came from.
func (memory *removalMemory) follow(placeOf func(pageIndex int) int) {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	memory.pageIndex = placeOf(memory.pageIndex)
}

func (memory *removalMemory) forget() {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	memory.held = false
}

func (pages *pageHandlers) deleteKey(writer http.ResponseWriter, request *http.Request) {
	form := request.PostForm
	pageIndex := integerParameter(form, "page")
	position := layout.Position{Row: integerParameter(form, "row"), Column: integerParameter(form, "column")}

	var removed []config.Key

	saveError := pages.changePage(pageIndex, func(page *config.Page) {
		page.Keys = slices.DeleteFunc(page.Keys, func(existing config.Key) bool {
			if existing.Position == position {
				removed = append(removed, existing)

				return true
			}

			return false
		})
	})
	if saveError != nil || len(removed) == 0 {
		pages.afterPageChange(writer, request, pageIndex, saveError)

		return
	}

	pages.removals.remember(pageIndex, removed)

	// The redirect's only variable part is a formatted integer, so it can only
	// ever name one of the editor's own pages.
	http.Redirect(writer, request, fmt.Sprintf("/page?page=%d&saved=removed", pageIndex), //nolint:gosec // see above
		http.StatusSeeOther)
}

// undoRemoval puts the last removed keys back where they were, if their page
// still exists and every one of their tiles is still free.
func (pages *pageHandlers) undoRemoval(writer http.ResponseWriter, request *http.Request) {
	pageIndex, keys, held := pages.removals.last()
	if !held {
		pages.afterPageChange(writer, request, 0, errNothingToUndo)

		return
	}

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if pageIndex >= len(configuration.Pages) {
			return errUndoPageGone
		}

		page := &configuration.Pages[pageIndex]
		for _, key := range keys {
			if _, taken := keyAt(*page, key.Position); taken {
				return errUndoTileTaken
			}
		}

		page.Keys = append(page.Keys, keys...)

		return nil
	})
	if saveError == nil {
		pages.removals.forget()
	}

	pages.afterPageChange(writer, request, pageIndex, saveError)
}
