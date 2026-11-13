package editor_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// TestEditor_GridMarksEveryTileForDragAndDrop: the author asked to drag a key
// to a new tile (evaluation, deck testing). Each tile says where it is and what
// it holds, only a key can be picked up, the page loads the drag script, and
// the drag hint stays hidden unless the script runs, so a page without it
// promises nothing.
func TestEditor_GridMarksEveryTileForDragAndDrop(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	for _, want := range []string{
		`data-tile="0.0.0" data-state="key"`,
		`data-tile="0.1.1" data-state="empty"`,
		`data-tile="0.3.0" data-state="reserved"`,
		`data-tile="0.3.7" data-state="reserved"`,
		`<script src="/static/editor.js" defer></script>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s in the page", want)
		}
	}

	draggable := regexp.MustCompile(`<a [^>]*draggable="true"[^>]*>`).FindAllString(body, -1)
	if len(draggable) != 1 || !strings.Contains(draggable[0], "row=0&amp;column=0") {
		t.Errorf("want only the stored key draggable, got %q", draggable)
	}

	if !regexp.MustCompile(`<p class="drag-help"[^>]* hidden>`).MatchString(body) {
		t.Error("want the drag hint hidden until the script shows it")
	}
}

func TestEditor_ServesTheEditorScript(t *testing.T) {
	t.Parallel()

	recorder := newHarness(t, nil).get(t, "/static/editor.js")

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("want the script served as JavaScript, got %d %s",
			recorder.Code, recorder.Header().Get("Content-Type"))
	}

	if !strings.Contains(recorder.Body.String(), "/key/move") {
		t.Fatal("want the script to post moves to /key/move")
	}
}

// TestEditor_PageHoldsTheKeyDialog: the author asked for a key to open over the
// grid rather than on a page of its own (evaluation, deck testing). The page
// carries an empty modal dialog, named by its heading and with a close button,
// for the script to fill with the key's editor; without the script the grid's
// links still open the key's own page.
func TestEditor_PageHoldsTheKeyDialog(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	dialog := regexp.MustCompile(`(?s)<dialog[^>]*id="key-dialog"[^>]*>.*?</dialog>`).FindString(body)
	if dialog == "" {
		t.Fatal("want a key dialog on the page")
	}

	for _, want := range []string{
		`aria-labelledby="key-dialog-title"`,
		`<h2 id="key-dialog-title">`,
		`<button type="button" class="dialog-close">Close</button>`,
		`<div id="key-dialog-body"></div>`,
	} {
		if !strings.Contains(dialog, want) {
			t.Errorf("want %s in the key dialog", want)
		}
	}

	if !strings.Contains(body, `href="/key?page=0&amp;row=0&amp;column=0"`) {
		t.Error("want the grid's links to still open the key's own page")
	}
}

// TestEditor_KeyPageWrapsEveryFormInItsEditor checks the part the dialog lifts
// out: the key's forms and preview all sit inside #key-editor.
func TestEditor_KeyPageWrapsEveryFormInItsEditor(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, "/key?page=0&row=0&column=0").Body.String()

	editor := regexp.MustCompile(`(?s)<div id="key-editor">.*</figure>\s*</div>`).FindString(body)
	for _, form := range []string{`id="type-form"`, `id="key-form"`, `action="/key/delete"`, `action="/key/move"`} {
		if !strings.Contains(editor, form) {
			t.Errorf("want %s inside #key-editor", form)
		}
	}
}
