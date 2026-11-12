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
	errUndoTileTaken = errors.New("the tile the key came from now holds another key; move that one first")
	errUndoPageGone  = errors.New("the page the key came from no longer exists")
)

// removalMemory is the last key removed and the page it came from, so a key
// dropped in the bin by mistake can be put back (the author's request for a
// bin that cannot easily be used by accident, evaluation, deck testing). Only
// the last removal is kept, in memory, until it is undone or replaced.
type removalMemory struct {
	mutex     sync.Mutex
	held      bool
	pageIndex int
	key       config.Key
}

func newRemovalMemory() *removalMemory {
	return &removalMemory{
		mutex: sync.Mutex{}, held: false, pageIndex: 0,
		key: config.Key{Position: layout.Position{Row: 0, Column: 0}, Kind: "", Parameters: nil, Preset: nil},
	}
}

func (memory *removalMemory) remember(pageIndex int, key config.Key) {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	memory.held, memory.pageIndex, memory.key = true, pageIndex, key.Clone()
}

func (memory *removalMemory) holding() bool {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	return memory.held
}

func (memory *removalMemory) last() (int, config.Key, bool) {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	return memory.pageIndex, memory.key.Clone(), memory.held
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

	pages.removals.remember(pageIndex, removed[0])

	// The redirect's only variable part is a formatted integer, so it can only
	// ever name one of the editor's own pages.
	http.Redirect(writer, request, fmt.Sprintf("/page?page=%d&saved=removed", pageIndex), //nolint:gosec // see above
		http.StatusSeeOther)
}

// undoRemoval puts the last removed key back where it was, if its page still
// exists and its tile is still free.
func (pages *pageHandlers) undoRemoval(writer http.ResponseWriter, request *http.Request) {
	pageIndex, key, held := pages.removals.last()
	if !held {
		pages.afterPageChange(writer, request, 0, errNothingToUndo)

		return
	}

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if pageIndex >= len(configuration.Pages) {
			return errUndoPageGone
		}

		page := &configuration.Pages[pageIndex]
		if _, taken := keyAt(*page, key.Position); taken {
			return errUndoTileTaken
		}

		page.Keys = append(page.Keys, key)

		return nil
	})
	if saveError == nil {
		pages.removals.forget()
	}

	pages.afterPageChange(writer, request, pageIndex, saveError)
}
