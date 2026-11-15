package editor_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The routes for moving and removing ticked keys, and where each returns.
const (
	moveKeysPath   = "/keys/move"
	removeKeysPath = "/keys/remove"
	firstPageMoved = "/page?page=0&saved=moved"
	pasteTile      = "0.2"
	// The shift a drag of ticked keys adds.
	rowOffsetField    = "row_offset"
	columnOffsetField = "column_offset"
)

// TestEditor_MovesTickedKeysOntoThePagesTicked: moving copies the ticked keys
// onto every page ticked, each on its own tile, and takes them off the page
// they came from.
func TestEditor_MovesTickedKeysOntoThePagesTicked(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, moveKeysPath, url.Values{
		pageField: {workPage}, tileField: {cutTile, pasteTile}, toField: {pycharmScreen, playPage},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != firstPageMoved {
		t.Fatalf("want a redirect to the page moved from, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}

	saved := harness.onDisk(t)

	for _, position := range []layout.Position{{Row: 0, Column: 1}, {Row: 0, Column: 2}} {
		if _, left := keyAtPosition(saved.Pages[0], position); left {
			t.Errorf("want the key at %v taken off Work", position)
		}

		for _, target := range []int{2, 3} {
			if _, landed := keyAtPosition(saved.Pages[target], position); !landed {
				t.Errorf("want the key at %v on page %d", position, target)
			}
		}
	}

	if _, kept := keyAtPosition(saved.Pages[0], layout.Position{Row: 1, Column: 1}); !kept {
		t.Error("want the unticked key left where it was")
	}

	if !strings.Contains(harness.get(t, firstPageMoved).Body.String(), "Moved.") {
		t.Error("want the page to say the keys were moved")
	}
}

// TestEditor_RefusesAMoveThatClashes_MovingNothing: a move that would land on
// a key anywhere moves nothing, so the keys stay where they were, and names
// the tile, as a copy does.
func TestEditor_RefusesAMoveThatClashes_MovingNothing(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	response := harness.post(t, moveKeysPath, url.Values{
		pageField: {workPage}, tileField: {musicTile}, toField: {golandScreen},
	})
	if response.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(response.Body.String(),
			"nothing was moved: the GoLand screen, row 2, column 2, already has a key") {
		t.Fatalf("want the move refused naming the tile, got %d: %s", response.Code, response.Body.String())
	}

	requireUnchanged(t, harness, before)
}

// TestEditor_RemovesTickedKeys_AndUndoPutsThemAllBack: removing takes every
// ticked key off the page and offers Undo, which puts all of them back.
func TestEditor_RemovesTickedKeys_AndUndoPutsThemAllBack(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	response := harness.post(t, removeKeysPath, url.Values{pageField: {workPage}, tileField: {cutTile, pasteTile}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != removedPage {
		t.Fatalf("want a redirect to the page, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}

	saved := harness.onDisk(t)
	if len(saved.Pages[0].Keys) != len(before.Pages[0].Keys)-2 {
		t.Fatalf("want two keys removed, got %+v", saved.Pages[0].Keys)
	}

	if !strings.Contains(harness.get(t, removedPage).Body.String(), `action="/undo"`) {
		t.Fatal("want Undo offered")
	}

	undone := harness.post(t, undoPath, url.Values{})
	if undone.Code != http.StatusSeeOther {
		t.Fatalf("want the undo saved, got %d: %s", undone.Code, undone.Body.String())
	}

	restored := harness.onDisk(t)
	for _, position := range []layout.Position{{Row: 0, Column: 1}, {Row: 0, Column: 2}} {
		if _, back := keyAtPosition(restored.Pages[0], position); !back {
			t.Errorf("want the key at %v put back", position)
		}
	}
}

// TestEditor_RefusesARemovalWithNothingTicked: removing needs a key ticked,
// but no page to go to.
func TestEditor_RefusesARemovalWithNothingTicked(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	response := harness.post(t, removeKeysPath, url.Values{pageField: {workPage}})
	if response.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(response.Body.String(), "no keys were ticked") {
		t.Fatalf("want the removal refused, got %d: %s", response.Code, response.Body.String())
	}

	requireUnchanged(t, harness, before)
}

// TestEditor_SelectModeOffersCopyMoveAndRemove: select mode's form sends the
// ticked keys to each of the three, by the button pressed.
func TestEditor_SelectModeOffersCopyMoveAndRemove(t *testing.T) {
	t.Parallel()

	body := newCopyKeysHarness(t).get(t, "/page?page=0&select=1").Body.String()

	for _, want := range []string{
		`<legend>Copy or move the ticked keys onto</legend>`,
		`<button type="submit">Copy the ticked keys</button>`,
		`<button type="submit" formaction="/keys/move">Move the ticked keys</button>`,
		`<button type="submit" formaction="/keys/remove" formnovalidate>Remove the ticked keys</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("select mode: want %s", want)
		}
	}

	if !strings.Contains(newCopyKeysHarness(t).get(t, firstPagePath).Body.String(),
		"Edit mode: tick keys on the grid to copy, move, remove or colour them") {
		t.Error("want edit mode to name what it is for")
	}
}

// TestEditor_MovesTickedKeysAcrossTheirOwnPage: the ticked keys dragged to
// another tile of their page move together, each shifted as far as the key
// dragged, even onto a tile one of them is leaving.
func TestEditor_MovesTickedKeysAcrossTheirOwnPage(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, moveKeysPath, url.Values{
		pageField: {workPage}, tileField: {cutTile, pasteTile}, toField: {workPage}, columnOffsetField: {"1"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the move saved, got %d: %s", response.Code, response.Body.String())
	}

	saved := harness.onDisk(t)

	for position, want := range map[layout.Position]string{
		{Row: 0, Column: 2}: "https://example.test/cut", {Row: 0, Column: 3}: "https://example.test/paste",
	} {
		key, found := keyAtPosition(saved.Pages[0], position)
		if !found || !strings.Contains(fmt.Sprintf("%+v", key.Parameters), want) {
			t.Errorf("want %s at %v, got %+v", want, position, key)
		}
	}

	if _, left := keyAtPosition(saved.Pages[0], layout.Position{Row: 0, Column: 1}); left {
		t.Error("want the tile the first key left empty")
	}

	if len(saved.Pages[0].Keys) != 4 {
		t.Errorf("want four keys still on Work, got %+v", saved.Pages[0].Keys)
	}
}

// TestEditor_ShiftsTickedKeysOntoAnotherPage: the shift applies on another
// page too, as when the drag opened that page and the key was dropped there.
func TestEditor_ShiftsTickedKeysOntoAnotherPage(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, copyKeysPath, url.Values{
		pageField: {workPage}, tileField: {cutTile, pasteTile}, toField: {playPage},
		rowOffsetField: {"2"}, columnOffsetField: {"-1"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the copy saved, got %d: %s", response.Code, response.Body.String())
	}

	play := harness.onDisk(t).Pages[3]
	for _, position := range []layout.Position{{Row: 2, Column: 0}, {Row: 2, Column: 1}} {
		if _, landed := keyAtPosition(play, position); !landed {
			t.Errorf("want a key at %v on Play", position)
		}
	}
}

// TestEditor_RefusesAShiftOffTheDeckOrOntoAKey: a shift that would put a key
// off the deck, on a key staying where it is, or on a paging tile moves
// nothing and names the tile; and a copy onto its own page still needs a
// shift.
func TestEditor_RefusesAShiftOffTheDeckOrOntoAKey(t *testing.T) {
	t.Parallel()

	for _, refused := range []struct {
		path   string
		tiles  []string
		row    string
		column string
		want   string
	}{
		{moveKeysPath, []string{cutTile}, "-1", "0", "row 0, column 2, is off the deck"},
		{moveKeysPath, []string{cutTile}, "1", "0", "row 2, column 2, already has a key"},
		{moveKeysPath, []string{musicTile}, "2", "-1", "row 4, column 1, is kept for paging"},
		{copyKeysPath, []string{cutTile}, "0", "0", "a page cannot be copied onto itself"},
	} {
		harness := newCopyKeysHarness(t)
		before := harness.onDisk(t)

		response := harness.post(t, refused.path, url.Values{
			pageField: {workPage}, tileField: refused.tiles, toField: {workPage},
			rowOffsetField: {refused.row}, columnOffsetField: {refused.column},
		})
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), refused.want) {
			t.Errorf("%+v: want refused naming %q, got %d: %s", refused, refused.want, response.Code, response.Body.String())
		}

		requireUnchanged(t, harness, before)
	}
}

// TestEditor_ScriptDragsTickedKeysTogether: in select mode each key's layer
// can be picked up, and the script carries every ticked key, shifting them by
// the drag through the same routes as select mode's buttons, dragging a
// picture of them all and drawing a ghost of each where it would land.
func TestEditor_ScriptDragsTickedKeysTogether(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{
		rowOffsetField, columnOffsetField, moveKeysPath, removeKeysPath, "setDragImage",
		"group-drag-image", "ghost-fits", "ghost-blocked",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}
}

// TestEditor_StylesTheGhostOfCarriedKeys: the stylesheet draws a carried
// key's ghost outlined where it fits and crossed where it does not.
func TestEditor_StylesTheGhostOfCarriedKeys(t *testing.T) {
	t.Parallel()

	stylesheet := newCopyKeysHarness(t).get(t, "/static/editor.css").Body.String()
	for _, want := range []string{
		".grid td .ghost {", ".grid td.ghost-blocked .ghost {", ".group-drag-image .select-key {",
	} {
		if !strings.Contains(stylesheet, want) {
			t.Errorf("stylesheet: want %s", want)
		}
	}
}
