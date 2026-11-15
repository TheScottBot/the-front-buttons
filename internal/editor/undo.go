package editor

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// removedMessage announces a removal; the page offers Undo beside it.
const removedMessage = "Removed. The deck shows the change within a second."

// Undo and redo: every change the editor saves is kept, the configuration
// before it and as it was written, so it can be undone and then redone, the
// last historyDepth changes at a time, in memory, until the daemon stops (the
// author's request). A new change after an undo starts a new line, so what was
// undone can no longer be redone. Undoing writes over the file, so if the file
// is no longer as the editor left it, changed by hand, undo and redo refuse
// rather than throw that change away.
const historyDepth = 20

// The fields and messages of undo and redo.
const (
	undoneQuery   = "undone"
	redoneQuery   = "redone"
	undoneMessage = "Undone. The deck shows the change within a second."
	redoneMessage = "Redone. The deck shows the change within a second."
)

// The reasons an undo or redo is refused, shown after the save error's prefix.
var (
	errNothingToUndo = errors.New("there is nothing to undo")
	errNothingToRedo = errors.New("there is nothing to redo")
	errHistoryStale  = errors.New("the configuration file has changed outside the editor since; " +
		"undoing or redoing now would throw that change away")
)

// historyStep is one change: the configuration before it and after it, each
// as decoded from the file.
type historyStep struct {
	before config.Configuration
	after  config.Configuration
}

// changeHistory is the changes that can be undone, and those undone that can
// be redone, newest last.
type changeHistory struct {
	mutex sync.Mutex
	undo  []historyStep
	redo  []historyStep
}

func newChangeHistory() *changeHistory {
	return &changeHistory{mutex: sync.Mutex{}, undo: nil, redo: nil}
}

// record keeps a change, forgetting the oldest past the depth and anything
// that could be redone.
func (history *changeHistory) record(before config.Configuration, after config.Configuration) {
	history.mutex.Lock()
	defer history.mutex.Unlock()

	history.undo = append(history.undo, historyStep{before: before.Clone(), after: after.Clone()})
	if len(history.undo) > historyDepth {
		history.undo = history.undo[len(history.undo)-historyDepth:]
	}

	history.redo = nil
}

// can reports whether there is a change to undo and one to redo.
func (history *changeHistory) can() (bool, bool) {
	history.mutex.Lock()
	defer history.mutex.Unlock()

	return len(history.undo) > 0, len(history.redo) > 0
}

// step undoes the newest change, or redoes the newest undone, writing the
// configuration it returns to through save, provided the file is as that
// change left it.
func (history *changeHistory) step(
	undoing bool, onDisk config.Configuration, save func(config.Configuration) error,
) error {
	history.mutex.Lock()
	defer history.mutex.Unlock()

	from, onto, nothing := &history.undo, &history.redo, errNothingToUndo
	if !undoing {
		from, onto, nothing = &history.redo, &history.undo, errNothingToRedo
	}

	if len(*from) == 0 {
		return nothing
	}

	change := (*from)[len(*from)-1]
	expected, target := change.after, change.before

	if !undoing {
		expected, target = change.before, change.after
	}

	// Compared as copies, as the history keeps them, so an empty page reads
	// the same either way.
	if !reflect.DeepEqual(onDisk.Clone(), expected) {
		return errHistoryStale
	}

	saveError := save(target.Clone())
	if saveError != nil {
		return saveError
	}

	*from = (*from)[:len(*from)-1]

	*onto = append(*onto, change)

	return nil
}

// undoChange undoes the newest change.
func (pages *pageHandlers) undoChange(writer http.ResponseWriter, request *http.Request) {
	pages.stepHistory(writer, request, true)
}

// redoChange redoes the newest change undone.
func (pages *pageHandlers) redoChange(writer http.ResponseWriter, request *http.Request) {
	pages.stepHistory(writer, request, false)
}

// stepHistory undoes or redoes a change and returns to the page on show, or
// the first page if that page is gone.
func (pages *pageHandlers) stepHistory(writer http.ResponseWriter, request *http.Request, undoing bool) {
	showing := max(integerParameter(request.PostForm, showingField), 0)

	onDisk, editableError := pages.editable()

	stepError := editableError
	if stepError == nil {
		stepError = pages.history.step(undoing, onDisk, pages.options.Store.Save)
	}

	configuration, _ := pages.configuration()
	if !pageExists(configuration, showing) {
		showing = 0
	}

	if stepError != nil {
		pages.renderPage(writer, request, http.StatusUnprocessableEntity, configuration, showing,
			[]string{errorPrefix + stepError.Error()})

		return
	}

	saved := undoneQuery
	if !undoing {
		saved = redoneQuery
	}

	// The redirect's only variable parts are a formatted integer and a
	// constant, so it can only ever name one of the editor's own pages.
	address := fmt.Sprintf("/page?page=%d&saved=%s", showing, saved)
	http.Redirect(writer, request, address, http.StatusSeeOther) //nolint:gosec // see above
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

	// The redirect's only variable part is a formatted integer, so it can only
	// ever name one of the editor's own pages.
	http.Redirect(writer, request, fmt.Sprintf("/page?page=%d&saved=removed", pageIndex), //nolint:gosec // see above
		http.StatusSeeOther)
}
