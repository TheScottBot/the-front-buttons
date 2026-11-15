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

	if !regexp.MustCompile(`<p class="shortcuts-help"[^>]* hidden>`).MatchString(body) {
		t.Error("want the shortcuts hint hidden until the script shows it")
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

// TestEditor_PageNamesAreDropTargets: every page and app screen in the page
// list names its page for the drag script, so a key dropped on one moves to
// the same tile there, and the shortcuts sheet says so and that holding over
// a name opens that page.
func TestEditor_PageNamesAreDropTargets(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)
	body := harness.get(t, "/page?page=0").Body.String()

	for _, want := range []string{
		`<a href="/page?page=0" data-page="0" aria-current="page">Work</a>`,
		`<a href="/page?page=2" data-page="2">Play</a>`,
		`<a href="/page?page=1" data-page="1">GoLand</a>`,
		`<tr><th scope="row">Copy a key</th><td>Hold Command as you drop it</td></tr>`,
		`<tr><th scope="row">Move a key to another page</th><td>Drop it on the page's name, ` +
			`or hold it there to open the page and place it</td></tr>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s in the page", want)
		}
	}
}

// TestEditor_ArrowKeysMoveAroundTheGrid: the script moves focus between
// tiles with the arrow keys and removes a key with Delete, through the same
// removal as its Remove button, and the shortcuts sheet says so.
func TestEditor_ArrowKeysMoveAroundTheGrid(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"ArrowUp", "ArrowRight", "/key/delete"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	body := harness.get(t, firstPagePath).Body.String()
	for _, want := range []string{
		`<tr><th scope="row">Move around the grid</th><td>Arrow keys, and Enter to open a tile</td></tr>`,
		`<tr><th scope="row">Remove a key</th><td>Drop it in the bin, or press Delete on its tile</td></tr>`,
		`<tr><th scope="row">Copy, cut or paste keys</th><td>Command-C, Command-X, Command-V</td></tr>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("shortcuts: want %s", want)
		}
	}
}

// TestEditor_KeepsTheGridInPlaceAcrossReloads: every change on the pages
// screen loads the page again; the script keeps where the grid sat in the
// window and puts the next page's grid there, with room below where the page
// is too short to scroll so far.
func TestEditor_KeepsTheGridInPlaceAcrossReloads(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"front-buttons-grid-place", "pagehide", "--keep-room"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	if !regexp.MustCompile(`(?s)body::after \{[^}]*height: var\(--keep-room, 0\)`).
		MatchString(harness.get(t, "/static/editor.css").Body.String()) {
		t.Error("stylesheet: want room below the page for a grid kept in place")
	}
}

// TestEditor_HintsSitBelowEverythingElse: the page lists lead straight into
// the caption and the grid; the page's forms and the pointer to the shortcuts
// sit below it, out of the way.
func TestEditor_HintsSitBelowEverythingElse(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	grid := strings.Index(body, `<table class="grid">`)
	addPage := strings.Index(body, `<button type="submit">Add page</button>`)

	if at := strings.Index(body, `class="shortcuts-help"`); at < addPage || at < grid {
		t.Error("want the shortcuts pointer after the grid and the new page form")
	}

	if between := body[strings.Index(body, `aria-label="Pages"`):grid]; strings.Contains(between, `<p class=`) {
		t.Errorf("want nothing but the caption between the page lists and the grid, got %s", between)
	}
}
