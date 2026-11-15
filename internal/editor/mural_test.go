package editor_test

import (
	"strings"
	"testing"
)

// TestEditor_PagesScreenZoomsOutToItsNeighbours: the pages screen has a Zoom
// out button, shown by the script, which draws the pages around this one
// side by side in place of its grid, from the overview's grids, each marked
// with its application so an app screen's mural pages through its own
// screens.
func TestEditor_PagesScreenZoomsOutToItsNeighbours(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	page := harness.get(t, firstPagePath).Body.String()
	for _, want := range []string{
		`<button type="button" class="zoom-out" aria-keyshortcuts="-" hidden>Zoom out</button>`,
		`<tr><th scope="row">Zoom out to the pages around this one</th><td>-, then the arrows to move along; ` +
			`Escape or + to come back</td></tr>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page: want %s", want)
		}
	}

	overview := harness.get(t, overviewPath).Body.String()
	for _, want := range []string{
		`<table class="grid mini" data-overview-page="0" data-application="">`,
		`<table class="grid mini" data-overview-page="1" data-application="com.jetbrains.goland">`,
	} {
		if !strings.Contains(overview, want) {
			t.Errorf("overview: want %s", want)
		}
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"zoomOut", "mural-strip", `event.key === "-"`} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	if !strings.Contains(harness.get(t, "/static/editor.css").Body.String(), ".mural-strip {") {
		t.Error("stylesheet: want the mural's pages side by side")
	}
}
