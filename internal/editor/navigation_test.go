package editor_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The navigation colour's route and the page it comes back to.
const (
	navigationColourPath = "/navigation/colour"
	navigationSaved      = "/page?page=0&navigation=1&saved=navigation"
	navigationShortcut   = `href="/page?page=0&amp;navigation=1#navigation-colour"`
)

// TestEditor_PreviewsThePagingTilesInTheNavigationColour: with a colour
// named for the navigation keys, the paging tiles are previewed in it, as
// the deck draws them.
func TestEditor_PreviewsThePagingTilesInTheNavigationColour(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, strings.Replace(deckRenderConfiguration, "pages:\n",
		"navigation_background: \"#FFE066\"\npages:\n", 1))

	onWork := layout.PageIndication{Number: 1, Count: 2, CurrentName: workName, PreviousName: playName, NextName: playName}
	want := previewRenderer(t).RenderNavigation(render.DirectionBack, onWork, true,
		render.NavigationBackgroundFrom("#FFE066"))

	if !testsupport.SameImage(previewImage(t, harness, backPreview), want) {
		t.Error("want Work's back tile previewed in the navigation colour")
	}
}

// TestEditor_ColoursTheNavigationKeys: the form saves the colour for the
// navigation keys, written as the file writes colours, comes back to the
// page with its drawer open, and Back to the usual grey takes it out again.
func TestEditor_ColoursTheNavigationKeys(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	response := harness.post(t, navigationColourPath, url.Values{pageField: {"0"}, backgroundField: {"#1f6feb"}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != navigationSaved+"#navigation-colour" {
		t.Fatalf("want a redirect to the page, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}

	if got := harness.onDisk(t).NavigationBackground; got != "#1F6FEB" {
		t.Fatalf("want the colour saved as #1F6FEB, got %q", got)
	}

	page := harness.get(t, navigationSaved).Body.String()
	requireBody(t, page, `<details class="maintenance navigation-colour" id="navigation-colour" open>`,
		`id="navigation-background" name="background" value="#1f6feb"`, "Back and forward keys coloured.")

	harness.post(t, navigationColourPath, url.Values{
		pageField: {"0"}, backgroundField: {"#1f6feb"}, clearColourField: {"1"},
	})

	if got := harness.onDisk(t).NavigationBackground; got != "" {
		t.Fatalf("want the colour taken out of the file, got %q", got)
	}
}

// TestEditor_RefusesABadNavigationColour: a colour not written #RRGGBB is
// refused, the file unchanged, with the drawer open and saying why.
func TestEditor_RefusesABadNavigationColour(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	response := harness.post(t, navigationColourPath, url.Values{pageField: {"0"}, backgroundField: {"blue"}})
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want the colour refused, got %d", response.Code)
	}

	requireBody(t, response.Body.String(), "#RRGGBB", `id="navigation-colour" open>`)

	if got := harness.onDisk(t).NavigationBackground; got != "" {
		t.Fatalf("want the file unchanged, got %q", got)
	}
}

// TestEditor_OffersAShortcutToTheNavigationColour: clicking a paging tile
// pages, so beside the grid a shortcut opens the drawer for their colour,
// shut until it is followed, filled in with the usual grey.
func TestEditor_OffersAShortcutToTheNavigationColour(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	page := harness.get(t, "/page?page=0").Body.String()
	requireBody(t, page, navigationShortcut, `<details class="maintenance navigation-colour" id="navigation-colour">`,
		`id="navigation-background" name="background" value="#24242a"`)

	opened := harness.get(t, "/page?page=0&navigation=1").Body.String()
	requireBody(t, opened, `<details class="maintenance navigation-colour" id="navigation-colour" open>`)
}

// TestEditor_DashboardPreviewPagingTilesTakeTheNavigationColour: with more
// app screens than a row holds, the dashboard preview's paging tiles are
// drawn in the navigation keys' colour, as the deck draws them.
func TestEditor_DashboardPreviewPagingTilesTakeTheNavigationColour(t *testing.T) {
	t.Parallel()

	configuration := strings.Replace(dashboardConfiguration, "pages:\n",
		"navigation_background: \"#FFE066\"\npages:\n", 1) +
		strings.Repeat("  - name: Green\n    application: com.example.green\n    keys: []\n", 8)

	harness := newDashboardHarness(t, configuration)
	rows := layout.PageIndication{Number: 1, Count: 2, CurrentName: "", PreviousName: "", NextName: ""}

	requireSamePicture(t, "the forward paging tile", dashboardKey(t, harness, "pages", 31),
		previewRenderer(t).RenderNavigation(render.DirectionForward, rows, true, render.NavigationBackgroundFrom("#FFE066")))
}
