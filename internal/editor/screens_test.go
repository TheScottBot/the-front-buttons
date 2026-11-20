package editor_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The route that adds a screen, and the second screen's place once added,
// straight after GoLand's first.
const (
	addScreenPath   = "/page/screen/add"
	golandSecond    = "GoLand 2"
	golandFirstName = "GoLand"
)

// addSecondScreen adds a second GoLand screen from the first.
func addSecondScreen(t *testing.T, harness *editorHarness) {
	t.Helper()

	response := harness.post(t, addScreenPath, url.Values{pageField: {golandScreenIndex}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/page?page=2&saved=key" {
		t.Fatalf("want a redirect to the new screen, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}
}

// TestEditor_AddsAnotherScreenForAnApplication: an app screen offers to add
// another for its application, placed straight after its last, named for it
// with its number; a regular page offers none and is refused one.
func TestEditor_AddsAnotherScreenForAnApplication(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)

	requireBody(t, harness.get(t, "/page?page="+golandScreenIndex).Body.String(),
		`<form method="post" action="/page/screen/add" class="inline">`, "Add another screen for GoLand")
	refuseBody(t, harness.get(t, "/page?page=0").Body.String(), `action="/page/screen/add"`)

	addSecondScreen(t, harness)

	saved := harness.onDisk(t).Pages
	if len(saved) != 4 || saved[2].Name != golandSecond || saved[2].Application != golandIdentifier ||
		saved[3].Name != "Play" {
		t.Fatalf("want GoLand 2 straight after GoLand, before Play, got %+v", saved)
	}

	if refused := harness.post(t, addScreenPath, url.Values{pageField: {"0"}}); refused.Code == http.StatusSeeOther {
		t.Error("want a screen refused for a regular page")
	}
}

// TestEditor_PagesBetweenAnApplicationsScreens: on one of two screens, the
// grid's paging tiles link to the other screen, as the deck pages between
// them, and are drawn numbering the screens; the caption and every list name
// the screen by its own name.
func TestEditor_PagesBetweenAnApplicationsScreens(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)
	addSecondScreen(t, harness)

	first := harness.get(t, "/page?page="+golandScreenIndex).Body.String()
	requireBody(t, first,
		`<a class="reserved" href="/page?page=2" aria-label="Row 4, column 8: Forward to screen 2, GoLand 2">`,
		`<a class="reserved" href="/page?page=2" aria-label="Row 4, column 1: Back to screen 2, GoLand 2">`,
		"Screen 1 of 2 for GoLand: GoLand, opened from its app key on the deck.",
	)

	selecting := harness.get(t, "/page?page=0&select=1").Body.String()
	requireBody(t, selecting, `<label for="copy-to-1">GoLand screen</label>`,
		`<label for="copy-to-2">GoLand 2 screen</label>`)

	onFirst := layout.PageIndication{
		Number: 1, Count: 2, CurrentName: golandFirstName, PreviousName: golandSecond, NextName: golandSecond,
	}
	want := previewRenderer(t).RenderNavigation(render.DirectionForward, onFirst, true,
		render.DefaultNavigationBackground())

	if !testsupport.SameImage(previewImage(t, harness, "/preview.png?page=1&row=3&column=7"), want) {
		t.Error("want the screen's forward tile drawn numbering the screens")
	}
}

// TestEditor_PreviewsAPagingTileOfNoPage: a paging tile asked for on a page
// that does not exist is drawn without a page count, not refused.
func TestEditor_PreviewsAPagingTileOfNoPage(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)

	if response := harness.get(t, "/preview.png?page=99&row=3&column=7"); response.Code != http.StatusOK {
		t.Fatalf("want the tile drawn, got %d", response.Code)
	}
}
