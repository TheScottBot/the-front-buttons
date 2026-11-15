package editor_test

import (
	"fmt"
	"image"
	"image/color"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The icons of the first two regular pages, as the dashboard draws them.
const (
	firstPageIcon  = "number-1"
	secondPageIcon = "number-2"
)

// The dashboard preview's routes.
const (
	dashboardPath     = "/dashboard"
	dashboardKeyImage = "/dashboard/key.png?view=%s&key=%d"
)

// dashboardConfiguration has Work, with a page marker in its top left and a
// key beside it, and Play, then the green application's screen, with two
// world clocks.
const dashboardConfiguration = `schema_version: 1
world_clocks:
  - label: BER
    zone: Europe/Berlin
  - label: ARQ
    zone: America/Sao_Paulo
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: page_marker
        parameters:
          page_number: true
      - row: 0
        column: 1
        kind: open_url
        parameters:
          url: https://example.test/cut
          label: Cut
  - name: Play
    keys: []
  - name: Green
    application: com.example.green
    keys: []
`

func newDashboardHarness(t *testing.T, configuration string) *editorHarness {
	t.Helper()

	harness := newHarness(t, nil)
	harness.replaceFile(t, configuration)

	return harness
}

// dashboardKey is one key of a view of the dashboard preview.
func dashboardKey(t *testing.T, harness *editorHarness, view string, keyIndex int) image.Image {
	t.Helper()

	return previewImage(t, harness, fmt.Sprintf(dashboardKeyImage, view, keyIndex))
}

// requireSamePicture fails unless two pictures match pixel for pixel.
func requireSamePicture(t *testing.T, what string, got image.Image, want image.Image) {
	t.Helper()

	if got.Bounds() != want.Bounds() {
		t.Errorf("%s: want %v, got %v", what, want.Bounds(), got.Bounds())

		return
	}

	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			if color.RGBAModel.Convert(got.At(x, y)) != color.RGBAModel.Convert(want.At(x, y)) {
				t.Errorf("%s: want it drawn as the deck draws it, differing at %d, %d", what, x, y)

				return
			}
		}
	}
}

// TestEditor_DashboardPreviewShowsBothViews: the Dashboard section shows the
// page dashboard as the deck draws it, its pages and its world clocks, each
// key a picture.
func TestEditor_DashboardPreviewShowsBothViews(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, dashboardConfiguration)

	response := harness.get(t, dashboardPath)
	if response.Code != http.StatusOK {
		t.Fatalf("want the dashboard preview, got %d", response.Code)
	}

	body := response.Body.String()
	for _, want := range []string{
		`<h1>Dashboard</h1>`,
		`<table class="dashboard-preview" data-dashboard-view="pages">`,
		`<table class="dashboard-preview" data-dashboard-view="world">`,
		`<img src="/dashboard/key.png?view=pages&amp;key=0" alt="" width="96" height="96">`,
		// The clock keys are drawn again as often as their figure can change:
		// the seconds' second figure each second, its first each ten, the
		// rest and the date each minute.
		`<img src="/dashboard/key.png?view=pages&amp;key=6" alt="" width="96" height="96" data-period="1">`,
		`<img src="/dashboard/key.png?view=pages&amp;key=5" alt="" width="96" height="96" data-period="10">`,
		`<img src="/dashboard/key.png?view=pages&amp;key=1" alt="" width="96" height="96" data-period="60">`,
		`<img src="/dashboard/key.png?view=world&amp;key=31" alt="" width="96" height="96" data-period="60">`,
		`<img src="/dashboard/key.png?view=world&amp;key=7" alt="" width="96" height="96">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard: want %s", want)
		}
	}

	if count := strings.Count(body, `<img src="/dashboard/key.png?view=`); count != 64 {
		t.Errorf("want every key of both views, 64, got %d", count)
	}

	if strings.Contains(body, "No page has a page marker") {
		t.Error("with a marker: want no word that the deck cannot open it")
	}

	if !strings.Contains(harness.get(t, firstPagePath).Body.String(), `<li><a href="/dashboard">Dashboard</a></li>`) {
		t.Error("want Dashboard in the editor's sections")
	}
}

// TestEditor_DashboardPreviewDrawsItsKeysAsTheDeck: each key of the preview
// is the deck's own picture: the marker held, its page marked among the
// regular pages, the app screens by their icons, the world icon, and the
// file's world clocks with the rows past them blank.
func TestEditor_DashboardPreviewDrawsItsKeysAsTheDeck(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, dashboardConfiguration)
	renderer := previewRenderer(t)
	marked, _ := render.ParseColour("#4b2050")

	requireSamePicture(t, "the marker", dashboardKey(t, harness, "pages", 0),
		previewImage(t, harness, "/preview.png?page=0&row=0&column=0"))
	requireSamePicture(t, "Work, marked", dashboardKey(t, harness, "pages", 8),
		renderer.Render(render.Appearance{Label: workName, Icon: firstPageIcon, Background: marked}))
	requireSamePicture(t, "Play", dashboardKey(t, harness, "pages", 9),
		renderer.Render(render.Appearance{Label: playName, Icon: secondPageIcon}))
	requireSamePicture(t, "the green screen", dashboardKey(t, harness, "pages", 16),
		renderer.Render(render.Appearance{Label: "Green", IconImage: greenAppIcon, BackgroundFromIcon: true}))

	for _, view := range []string{"pages", "world"} {
		requireSamePicture(t, view+": the world icon", dashboardKey(t, harness, view, 7), renderer.RenderClockIcon("world"))
	}

	requireSamePicture(t, "Berlin", dashboardKey(t, harness, "world", 0), renderer.RenderClockLabel("BER"))
	requireSamePicture(t, "Araraquara", dashboardKey(t, harness, "world", 8), renderer.RenderClockLabel("ARQ"))
	requireSamePicture(t, "past the clocks", dashboardKey(t, harness, "world", 16), renderer.RenderBlank())
}

// TestEditor_DashboardPreviewWithoutAMarkerOrClocks: with no page marker the
// preview says the deck cannot open the dashboard yet, and shows it from the
// first page, its top left blank; with no world clocks in the file, the
// world view shows the four defaults.
func TestEditor_DashboardPreviewWithoutAMarkerOrClocks(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, copyKeysConfiguration)
	renderer := previewRenderer(t)
	marked, _ := render.ParseColour("#4b2050")

	if body := harness.get(t, dashboardPath).Body.String(); !strings.Contains(body, "No page has a page marker") {
		t.Error("without a marker: want word that the deck cannot open the dashboard yet")
	}

	requireSamePicture(t, "the top left", dashboardKey(t, harness, "pages", 0), renderer.RenderBlank())
	requireSamePicture(t, "Work, marked", dashboardKey(t, harness, "pages", 8),
		renderer.Render(render.Appearance{Label: workName, Icon: firstPageIcon, Background: marked}))
	requireSamePicture(t, "UTC", dashboardKey(t, harness, "world", 0), renderer.RenderClockLabel("UTC"))
	requireSamePicture(t, "Sao Paulo", dashboardKey(t, harness, "world", 24), renderer.RenderClockLabel("SAO"))
}

// TestEditor_DashboardPreviewRefusesAKeyNotOnTheDeck: a view or key the deck
// has not is not found.
func TestEditor_DashboardPreviewRefusesAKeyNotOnTheDeck(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, dashboardConfiguration)

	for _, path := range []string{
		fmt.Sprintf(dashboardKeyImage, "pages", 32), fmt.Sprintf(dashboardKeyImage, "pages", -1),
		fmt.Sprintf(dashboardKeyImage, "moon", 0), "/dashboard/key.png?view=world&key=x",
	} {
		if code := harness.get(t, path).Code; code != http.StatusNotFound {
			t.Errorf("GET %s: want not found, got %d", path, code)
		}
	}
}

// TestEditor_DashboardPreviewOpensFromThePageOfTheFirstMarker: with the only
// page marker on Play, the preview is opened from Play, which is marked.
func TestEditor_DashboardPreviewOpensFromThePageOfTheFirstMarker(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, `schema_version: 1
pages:
  - name: Work
    keys: []
  - name: Play
    keys:
      - row: 1
        column: 2
        kind: page_marker
`)
	renderer := previewRenderer(t)
	marked, _ := render.ParseColour("#4b2050")

	requireSamePicture(t, "the marker", dashboardKey(t, harness, "pages", 0),
		previewImage(t, harness, "/preview.png?page=1&row=1&column=2"))
	requireSamePicture(t, "Work", dashboardKey(t, harness, "pages", 8),
		renderer.Render(render.Appearance{Label: workName, Icon: firstPageIcon}))
	requireSamePicture(t, "Play, marked", dashboardKey(t, harness, "pages", 9),
		renderer.Render(render.Appearance{Label: playName, Icon: secondPageIcon, Background: marked}))
}

// twoMarkers has a page marker on Work and another on Play.
const twoMarkers = `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: page_marker
  - name: Play
    keys:
      - row: 1
        column: 2
        kind: page_marker
`

// TestEditor_DashboardPreviewOpensFromThePageTheDeckShows: once the deck has
// said which page it shows, the preview is opened from that page, marked,
// with its marker in the top left; a page the file no longer has is taken as
// none said.
func TestEditor_DashboardPreviewOpensFromThePageTheDeckShows(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, twoMarkers)
	renderer := previewRenderer(t)
	marked, _ := render.ParseColour("#4b2050")

	harness.mirror.ShowPage(1)
	requireSamePicture(t, "Play's marker", dashboardKey(t, harness, "pages", 0),
		previewImage(t, harness, "/preview.png?page=1&row=1&column=2"))
	requireSamePicture(t, "Work", dashboardKey(t, harness, "pages", 8),
		renderer.Render(render.Appearance{Label: workName, Icon: firstPageIcon}))
	requireSamePicture(t, "Play, marked", dashboardKey(t, harness, "pages", 9),
		renderer.Render(render.Appearance{Label: playName, Icon: secondPageIcon, Background: marked}))

	if showing := harness.get(t, "/dashboard/showing").Body.String(); showing != "1" {
		t.Errorf("want the page the deck shows said as 1, got %q", showing)
	}

	harness.mirror.ShowPage(7)
	requireSamePicture(t, "Work, marked", dashboardKey(t, harness, "pages", 8),
		renderer.Render(render.Appearance{Label: workName, Icon: firstPageIcon, Background: marked}))
}

// TestEditor_DashboardPreviewDrawsItsClocksAtTheMomentAsked: a clock key
// asked for at a moment is drawn as the deck would draw it then, the local
// time in the top row and each world clock's in its zone, so the page's
// script can draw them again each second.
func TestEditor_DashboardPreviewDrawsItsClocksAtTheMomentAsked(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, dashboardConfiguration)
	renderer := previewRenderer(t)
	moment := time.Date(2026, time.March, 4, 9, 27, 58, 0, time.UTC)
	berlin, _ := time.LoadLocation("Europe/Berlin")
	atMoment := fmt.Sprintf("&at=%d", moment.Unix())

	local := time.Unix(moment.Unix(), 0).Format("150405")
	requireSamePicture(t, "the local hour",
		previewImage(t, harness, fmt.Sprintf(dashboardKeyImage, "pages", 1)+atMoment),
		renderer.RenderClockDigit(local[0:1]))
	requireSamePicture(t, "Berlin's seconds",
		previewImage(t, harness, fmt.Sprintf(dashboardKeyImage, "world", 6)+atMoment),
		renderer.RenderClockSecondDigit(moment.In(berlin).Format("05")[1:], false))

	if script := harness.get(t, "/static/editor.js").Body.String(); !strings.Contains(script, "dataset.period") ||
		!strings.Contains(script, "/dashboard/showing") {
		t.Error("want the script to draw the clocks again and follow the deck's page")
	}
}
