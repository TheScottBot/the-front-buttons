package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestEditor_PageHoldsAHiddenBrickDrawer: the author asked for a drawer in
// the left gutter that opens on a click and offers bricks to drag onto an
// empty tile. It lists every kind and preset, as the key form does; dragging
// needs the script, so it stays hidden until the script shows it, and the
// grid's empty tiles stay the way to add a key from the keyboard.
func TestEditor_PageHoldsAHiddenBrickDrawer(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	body := harness.get(t, firstPagePath).Body.String()

	drawer := regexp.MustCompile(`(?s)<aside id="brick-drawer"[^>]*>.*?</aside>`).FindString(body)
	if drawer == "" {
		t.Fatal("want the brick drawer on the page")
	}

	for _, want := range []string{
		`<aside id="brick-drawer" class="brick-drawer" aria-label="Bricks" hidden>`,
		`<button type="button" class="drawer-toggle" aria-expanded="false" aria-controls="brick-list">`,
		`<div id="brick-list" class="drawer-panel" hidden>`,
		`<li draggable="true" data-type="kind:open_url">`,
		`<li draggable="true" data-type="preset:discord_channel">`,
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("drawer: want %s", want)
		}
	}

	harness.replaceFile(t, "schema_version: 1\npages: []\n")

	if strings.Contains(harness.get(t, firstPagePath).Body.String(), "brick-drawer") {
		t.Error("want no drawer with no page to drop onto")
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"application/x-front-buttons-brick", "placing", "carrying", "brick-drawer"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	// While a brick is carried, a tile it cannot go on shows a cross.
	stylesheet := harness.get(t, "/static/editor.css").Body.String()
	if !strings.Contains(stylesheet, `.grid.placing td[data-state="key"]`) ||
		!strings.Contains(stylesheet, `.grid.placing td[data-state="reserved"]`) {
		t.Error("stylesheet: want a cross on keys and reserved tiles while a brick is carried")
	}

	// And the open panel lets the drag through to tiles it covers.
	if !regexp.MustCompile(`(?s)\.brick-drawer\.carrying \{[^}]*pointer-events: none`).MatchString(stylesheet) {
		t.Error("stylesheet: want the drawer to let a carried brick through to the tiles")
	}
}
