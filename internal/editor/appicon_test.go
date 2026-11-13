package editor_test

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The application the fake system has an icon for.
const previewedApplication = "com.example.green"

// greenAppIcon is a green square in a transparent margin, as macOS draws an
// application's icon.
var greenAppIcon = func() image.Image { //nolint:gochecknoglobals // a fixed test picture
	picture := image.NewNRGBA(image.Rect(0, 0, 32, 32))

	for x := 4; x < 28; x++ {
		for y := 4; y < 28; y++ {
			picture.SetNRGBA(x, y, color.NRGBA{R: 0x10, G: 0xa0, B: 0x40, A: 0xff})
		}
	}

	return picture
}()

// applicationIcons knows one application's icon.
type applicationIcons struct{}

func (applicationIcons) Icon(_ context.Context, identifier string) (image.Image, error) {
	if identifier == previewedApplication {
		return greenAppIcon, nil
	}

	return nil, fmt.Errorf("%w: %s", platform.ErrApplicationNotFound, identifier)
}

func previewImage(t *testing.T, harness *editorHarness, path string) image.Image {
	t.Helper()

	recorder := harness.get(t, path)

	preview, decodeError := png.Decode(recorder.Body)
	if decodeError != nil {
		t.Fatalf("GET %s: want a PNG, got %d (%v)", path, recorder.Code, decodeError)
	}

	return preview
}

func previewRenderer(t *testing.T) *render.Renderer {
	t.Helper()

	renderer, rendererError := render.NewRenderer(96, noIcons{})
	if rendererError != nil {
		t.Fatal(rendererError)
	}

	return renderer
}

const appIconConfiguration = `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: open_application
        parameters:
          application: com.example.green
          label: Green
          use_app_icon: true
      - row: 0
        column: 1
        kind: open_application
        parameters:
          application: com.example.gone
          label: Gone
          use_app_icon: true
`

// TestEditor_PreviewsTheApplicationsOwnIcon_AsTheDeckDrawsIt: the key preview
// shows what the deck shows: the application's icon on its edge colour, or the
// missing marker when the icon cannot be read.
func TestEditor_PreviewsTheApplicationsOwnIcon_AsTheDeckDrawsIt(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	writeError := os.WriteFile(harness.configPath, []byte(appIconConfiguration), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}

	renderer := previewRenderer(t)

	withIcon := renderer.Render(render.Appearance{Label: "Green", IconImage: greenAppIcon, BackgroundFromIcon: true})
	if !testsupport.SameImage(previewImage(t, harness, "/preview.png?page=0&row=0&column=0"), withIcon) {
		t.Error("want the saved key previewed with its application's icon on the icon's green")
	}

	missing := renderer.Render(render.Appearance{Label: "Gone", IconMissing: true, BackgroundFromIcon: true})
	if !testsupport.SameImage(previewImage(t, harness, "/preview.png?page=0&row=0&column=1"), missing) {
		t.Error("want the missing marker previewed for an icon that cannot be read")
	}

	unsaved := "/preview.png?type=kind:open_application&field.label=Green&field.use_app_icon=on" +
		"&field.application=" + previewedApplication
	if !testsupport.SameImage(previewImage(t, harness, unsaved), withIcon) {
		t.Error("want an unsaved form previewed with its application's icon too")
	}
}
