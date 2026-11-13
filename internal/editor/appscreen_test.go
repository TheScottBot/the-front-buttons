package editor_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The application the app screen tests use, and its key's fields.
const (
	golandIdentifier  = "com.jetbrains.goland"
	openAppType       = "kind:open_application"
	applicationField  = "field.application"
	appScreenField    = "field.app_screen"
	golandBuildType   = "kind:goland_build"
	golandScreenIndex = "1"
	spotifyIdentifier = "com.spotify.client"
)

// golandRunParts are GoLand's run group's parts, in the drawer's first order.
//
//nolint:gochecknoglobals // a fixed list
var golandRunParts = []string{"goland_run", "goland_debug", "goland_stop", "goland_rerun"}

// appScreenConfiguration has two regular pages, Work and Play, with GoLand's
// app screen between them in the file, and a GoLand key on Work that opens it.
const appScreenConfiguration = `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 1
        column: 1
        kind: open_application
        parameters:
          application: com.jetbrains.goland
          app_screen: true
  - name: GoLand
    application: com.jetbrains.goland
    keys:
      - row: 1
        column: 1
        kind: goland_build
        parameters:
          label: Build
  - name: Play
    keys:
      - row: 0
        column: 0
        kind: open_url
        parameters:
          url: https://example.test/play
          label: Play
`

func newAppScreenHarness(t *testing.T) *editorHarness {
	t.Helper()

	harness := newHarness(t, nil)
	harness.replaceFile(t, appScreenConfiguration)

	return harness
}

func requireBody(t *testing.T, body string, wanted ...string) {
	t.Helper()

	for _, text := range wanted {
		if !strings.Contains(body, text) {
			t.Errorf("want %q in the page, got:\n%s", text, body)
		}
	}
}

func refuseBody(t *testing.T, body string, unwanted ...string) {
	t.Helper()

	for _, text := range unwanted {
		if strings.Contains(body, text) {
			t.Errorf("want no %q in the page, got:\n%s", text, body)
		}
	}
}

// TestEditor_CreatesTheAppScreen_WhenAKeyAsksForIt: ticking App screen on an
// app key creates the application's screen, named after the application, the
// first time; a second key asking for it reuses that screen.
func TestEditor_CreatesTheAppScreen_WhenAKeyAsksForIt(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, column := range []string{"1", "2"} {
		response := harness.post(t, "/key", url.Values{
			pageField: {"0"}, rowField: {"1"}, columnField: {column}, typeField: {openAppType},
			applicationField: {golandIdentifier}, appScreenField: {"on"},
		})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("want the key saved, got %d: %s", response.Code, response.Body.String())
		}
	}

	response := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"3"}, typeField: {openAppType},
		applicationField: {spotifyIdentifier}, appScreenField: {"on"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the key saved, got %d: %s", response.Code, response.Body.String())
	}

	saved := harness.onDisk(t).Pages
	if len(saved) != 3 {
		t.Fatalf("want Work and one screen for each application, got %+v", saved)
	}

	// GoLand is named by its own bricks' set; Spotify by the app picker.
	if saved[1].Name != "GoLand" || saved[1].Application != golandIdentifier || len(saved[1].Keys) != 0 {
		t.Errorf("want an empty GoLand screen, got %+v", saved[1])
	}

	if saved[2].Name != "Spotify" || saved[2].Application != spotifyIdentifier {
		t.Errorf("want a Spotify screen, got %+v", saved[2])
	}
}

// TestEditor_ListsAppScreensApartFromRegularPages: the page list numbers the
// regular pages alone and lists app screens under them, and paging in the grid
// skips app screens, as the deck does.
func TestEditor_ListsAppScreensApartFromRegularPages(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)

	play := harness.get(t, "/page?page=2").Body.String()
	requireBody(t, play,
		"Page 2 of 2: Play",
		`<h2 id="app-screens">App screens</h2>`,
		`<a href="/page?page=1">GoLand</a>`,
	)

	// Work pages forward to Play and back, round, to Play: never GoLand.
	work := harness.get(t, "/page?page=0").Body.String()
	requireBody(t, work, "Page 1 of 2: Work", "Forward to page 2, Play", "Back to page 2, Play")
	refuseBody(t, work, "to page 2, GoLand", "to page 3")
}

// TestEditor_AppScreenKeepsItsIconTile: an app screen's top left tile is kept
// for the application's icon, which returns to the regular pages on the deck,
// and no key may be moved or grouped onto it.
func TestEditor_AppScreenKeepsItsIconTile(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)

	screen := harness.get(t, "/page?page="+golandScreenIndex).Body.String()
	requireBody(t, screen,
		"App screen for GoLand",
		`data-tile="1.0.0" data-state="reserved"`,
		"Row 1, column 1: reserved for the application&#39;s icon, which goes back to the regular pages",
		`src="/preview.png?page=1&amp;row=0&amp;column=0"`,
	)

	// The move form offers every other free tile of the screen, never its
	// top left.
	keyForm := harness.get(t, "/key?page=0&row=1&column=1").Body.String()
	requireBody(t, keyForm, `value="1.0.1"`)
	refuseBody(t, keyForm, `value="1.0.0"`)

	moved := harness.post(t, "/key/move", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, toField: {"1.0.0"}, actionField: {"move"},
	})
	if moved.Code == http.StatusSeeOther {
		t.Error("want a move onto the icon tile refused")
	}

	grouped := harness.post(t, "/group", url.Values{
		pageField: {golandScreenIndex}, rowField: {"0"}, columnField: {"0"}, sizeField: {"goland.run"},
		orderField: golandRunParts,
	})
	if grouped.Code == http.StatusSeeOther || !strings.Contains(grouped.Body.String(), "application&#39;s icon") {
		t.Errorf("want a group over the icon tile refused by name, got %d", grouped.Code)
	}

	if saved := harness.onDisk(t).Pages[1].Keys; len(saved) != 1 {
		t.Errorf("want only the build key on the screen, got %+v", saved)
	}
}

// TestEditor_OffersAnAppsBricksOnItsScreenOnly: GoLand's bricks and groups
// are in the drawer and the key form on GoLand's screen, beside every other
// brick, and on no other page.
func TestEditor_OffersAnAppsBricksOnItsScreenOnly(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)

	screen := harness.get(t, "/page?page="+golandScreenIndex).Body.String()
	requireBody(t, screen,
		`data-type="`+golandBuildType+`"`, `data-type="kind:open_url"`,
		`data-group="goland.run" data-width="2" data-height="2"`, "<summary>GoLand</summary>",
	)

	screenForm := harness.get(t, "/key?page=1&row=2&column=2").Body.String()
	requireBody(t, screenForm, `value="`+golandBuildType+`"`)

	for _, path := range []string{"/page?page=0", "/key?page=0&row=2&column=2"} {
		refuseBody(t, harness.get(t, path).Body.String(), "goland_")
	}

	// An app's brick asked for by name on another page is refused.
	if response := harness.get(t, "/key?page=0&row=2&column=2&type="+golandBuildType); response.Code !=
		http.StatusBadRequest {
		t.Errorf("want GoLand's brick refused off its screen, got %d", response.Code)
	}
}

// TestEditor_PlacesAnAppGroup: an app's group places each of its keys, as the
// drawer's other groups do.
func TestEditor_PlacesAnAppGroup(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)

	form := harness.get(t, "/group?page=1&row=1&column=2&size=goland.run")
	if form.Code != http.StatusOK {
		t.Fatalf("want the group's form, got %d", form.Code)
	}

	response := harness.post(t, "/group", url.Values{
		pageField: {golandScreenIndex}, rowField: {"1"}, columnField: {"2"}, sizeField: {"goland.run"},
		orderField: {golandRunParts[1], golandRunParts[0], golandRunParts[2], golandRunParts[3]},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the group placed, got %d: %s", response.Code, response.Body.String())
	}

	want := map[layout.Position]string{
		{Row: 1, Column: 2}: golandRunParts[1], {Row: 1, Column: 3}: golandRunParts[0],
		{Row: 2, Column: 2}: golandRunParts[2], {Row: 2, Column: 3}: golandRunParts[3],
	}

	for _, key := range harness.onDisk(t).Pages[1].Keys {
		if kindName, placed := want[key.Position]; placed && kindName != key.Kind {
			t.Errorf("%+v: want %s, got %s", key.Position, kindName, key.Kind)
		}
	}

	if placed := len(harness.onDisk(t).Pages[1].Keys); placed != 1+len(want) {
		t.Errorf("want the group's four keys beside the build key, got %d keys", placed)
	}
}

// TestEditor_LinksAnAppKeyToItsScreen: the key form of a key that opens an app
// screen links to the screen, so its keys are a step away.
func TestEditor_LinksAnAppKeyToItsScreen(t *testing.T) {
	t.Parallel()

	harness := newAppScreenHarness(t)

	keyForm := harness.get(t, "/key?page=0&row=1&column=1").Body.String()
	requireBody(t, keyForm, `<a href="/page?page=1">Edit the GoLand screen</a>`)
}

// TestEditor_PreviewsTheReturnTileAsTheDeckDrawsIt: the return tile's picture
// is the application's icon alone, on its edge colour, as the deck draws it.
func TestEditor_PreviewsTheReturnTileAsTheDeckDrawsIt(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, startingConfiguration+`  - name: Green
    application: `+previewedApplication+`
    keys:
      - row: 1
        column: 1
        kind: open_url
        parameters:
          url: https://example.test/green
`)

	want := previewRenderer(t).Render(render.Appearance{IconImage: greenAppIcon, BackgroundFromIcon: true})
	if !sameImage(previewImage(t, harness, "/preview.png?page=1&row=0&column=0"), want) {
		t.Error("want the return tile previewed as the application's icon alone")
	}

	// A free tile on a regular page has no icon.
	if sameImage(previewImage(t, harness, "/preview.png?page=0&row=1&column=0"), want) {
		t.Error("want no icon on a regular page's free tile")
	}
}
