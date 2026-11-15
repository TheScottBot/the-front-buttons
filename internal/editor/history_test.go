package editor_test

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// The redo route.
const redoPath = "/redo"

// renamePage renames Play through the editor.
func renamePage(t *testing.T, harness *editorHarness, name string) {
	t.Helper()

	response := harness.post(t, "/page/rename", url.Values{pageField: {playPage}, "name": {name}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the rename saved, got %d: %s", response.Code, response.Body.String())
	}
}

// TestEditor_UndoesAndRedoesAnyChange: any change saved in the editor can be
// undone, here a rename, and redone, returning to the page on show.
func TestEditor_UndoesAndRedoesAnyChange(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	renamePage(t, harness, "Games")
	after := harness.onDisk(t)

	undone := harness.post(t, undoPath, url.Values{showingField: {playPage}})
	if undone.Code != http.StatusSeeOther || undone.Header().Get("Location") != "/page?page=3&saved=undone" {
		t.Fatalf("want a redirect to the page on show, got %d %s: %s",
			undone.Code, undone.Header().Get("Location"), undone.Body.String())
	}

	if got := harness.onDisk(t); !reflect.DeepEqual(got, before) {
		t.Fatalf("want the rename undone, got %+v", got.Pages)
	}

	redone := harness.post(t, redoPath, url.Values{showingField: {playPage}})
	if redone.Code != http.StatusSeeOther || redone.Header().Get("Location") != "/page?page=3&saved=redone" {
		t.Fatalf("want a redirect after redoing, got %d %s", redone.Code, redone.Header().Get("Location"))
	}

	if got := harness.onDisk(t); !reflect.DeepEqual(got, after) {
		t.Fatalf("want the rename redone, got %+v", got.Pages)
	}

	for path, want := range map[string]string{
		"/page?page=3&saved=undone": "Undone.", "/page?page=3&saved=redone": "Redone.",
	} {
		if !strings.Contains(harness.get(t, path).Body.String(), want) {
			t.Errorf("%s: want %s", path, want)
		}
	}
}

// TestEditor_ANewChangeClearsRedo: a change made after an undo starts a new
// line of changes, so what was undone can no longer be redone.
func TestEditor_ANewChangeClearsRedo(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	renamePage(t, harness, "Games")
	harness.post(t, undoPath, url.Values{})
	renamePage(t, harness, "Fun")

	response := harness.post(t, redoPath, url.Values{})
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "nothing to redo") {
		t.Fatalf("want redo refused, got %d: %s", response.Code, response.Body.String())
	}

	if name := harness.onDisk(t).Pages[3].Name; name != "Fun" {
		t.Errorf("want the new change kept, got %s", name)
	}
}

// TestEditor_UndoRefusesAfterAHandEdit: undoing would write over the file,
// so after the file is changed outside the editor, undo and redo refuse
// rather than throw that change away.
func TestEditor_UndoRefusesAfterAHandEdit(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	renamePage(t, harness, "Games")
	harness.replaceFile(t, strings.Replace(copyKeysConfiguration, "name: Play", "name: Edited", 1))
	before := harness.onDisk(t)

	response := harness.post(t, undoPath, url.Values{})
	if response.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(response.Body.String(), "changed outside the editor") {
		t.Fatalf("want undo refused after a hand edit, got %d: %s", response.Code, response.Body.String())
	}

	requireUnchanged(t, harness, before)
}

// TestEditor_UndoKeepsTheLastTwentyChanges: the history holds the last
// twenty changes; older ones cannot be undone.
func TestEditor_UndoKeepsTheLastTwentyChanges(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	for change := range 21 {
		renamePage(t, harness, "Play "+strconv.Itoa(change))
	}

	for undo := range 20 {
		if response := harness.post(t, undoPath, url.Values{}); response.Code != http.StatusSeeOther {
			t.Fatalf("undo %d: want it saved, got %d", undo+1, response.Code)
		}
	}

	if response := harness.post(t, undoPath, url.Values{}); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want a twenty first undo refused, got %d", response.Code)
	}

	if name := harness.onDisk(t).Pages[3].Name; name != "Play 0" {
		t.Errorf("want the first change kept, got %s", name)
	}
}

// TestEditor_PageOffersUndoAndRedo: the pages screen has Undo and Redo
// buttons, each turned off while there is nothing to do, sending the page on
// show so the editor returns to it.
func TestEditor_PageOffersUndoAndRedo(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	fresh := harness.get(t, "/page?page=3").Body.String()
	for _, want := range []string{
		`<div class="history">`,
		`<form method="post" action="/undo" class="in-place">`,
		`<input type="hidden" name="showing" value="3">`,
		`<button type="submit" disabled>Undo</button>`,
		`<button type="submit" disabled>Redo</button>`,
	} {
		if !strings.Contains(fresh, want) {
			t.Errorf("fresh page: want %s", want)
		}
	}

	renamePage(t, harness, "Games")

	changed := harness.get(t, "/page?page=3").Body.String()
	if !strings.Contains(changed, `<button type="submit">Undo</button>`) ||
		!strings.Contains(changed, `<button type="submit" disabled>Redo</button>`) {
		t.Error("want Undo on and Redo off after a change")
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"form.in-place", `"/undo"`, `"/redo"`, ".history"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}
}
