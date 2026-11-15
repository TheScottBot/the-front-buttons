package editor_test

import (
	"regexp"
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

// TestEditor_ZoomedOutPagesReachAcrossTheWindow: the zoomed out pages screen
// and All pages reach across a wide window, and the zoom button sits outside
// Undo and Redo, which a change made in place redraws.
func TestEditor_ZoomedOutPagesReachAcrossTheWindow(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	page := harness.get(t, firstPagePath).Body.String()

	stylesheet := harness.get(t, "/static/editor.css").Body.String()
	for _, want := range []string{
		".mural,\n.overview {\n  margin-right: min(0px, calc(50% - 50vw + 3rem));",
	} {
		if !strings.Contains(stylesheet, want) {
			t.Errorf("stylesheet: want %s", want)
		}
	}

	// The button is outside Undo and Redo, which a change made in place
	// redraws, and which would hide it again.
	group := regexp.MustCompile(`(?s)<div class="history">.*?</div>`).FindString(page)
	if group == "" || strings.Contains(group, "zoom-out") {
		t.Error("want the zoom button outside the Undo and Redo group")
	}
}
