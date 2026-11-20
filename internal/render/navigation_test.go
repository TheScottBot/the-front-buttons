package render_test

import (
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// navigationBlack is the navigation keys' usual background.
var navigationBlack = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff} //nolint:gochecknoglobals // a fixed colour

// TestNavigationBackgroundFrom: a configured colour is used as written, and
// none, or one the renderer cannot read, leaves the usual black.
func TestNavigationBackgroundFrom(t *testing.T) {
	t.Parallel()

	if got := render.DefaultNavigationBackground(); got != navigationBlack {
		t.Errorf("want the usual black by default, got %v", got)
	}

	for written, want := range map[string]color.RGBA{
		"#1F6FEB": {R: 0x1f, G: 0x6f, B: 0xeb, A: 0xff},
		"#ffe066": {R: 0xff, G: 0xe0, B: 0x66, A: 0xff},
		"":        navigationBlack,
		"blue":    navigationBlack,
	} {
		if got := render.NavigationBackgroundFrom(written); got != want {
			t.Errorf("%q: want %v, got %v", written, want, got)
		}
	}
}

// TestRenderNavigation_TakesItsBackground: a navigation key is filled with the
// background it is given, and its chevron and words are drawn in whichever of
// black or white contrasts more with it, as an ordinary key's label is.
func TestRenderNavigation_TakesItsBackground(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	indication := layout.PageIndication{Number: 2, Count: 5, CurrentName: "Work", PreviousName: "Home", NextName: "Games"}
	black := color.RGBA{R: 0, G: 0, B: 0, A: 0xff}

	for name, check := range map[string]struct {
		background color.RGBA
		ink        color.RGBA
	}{
		"dark blue":  {color.RGBA{R: 0x10, G: 0x20, B: 0x60, A: 0xff}, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
		"pale amber": {color.RGBA{R: 0xff, G: 0xe0, B: 0x66, A: 0xff}, black},
	} {
		key := renderer.RenderNavigation(render.DirectionBack, indication, true, check.background)

		if got := color.RGBAModel.Convert(key.At(1, 1)); got != check.background {
			t.Errorf("%s: want the corner %v, got %v", name, check.background, got)
		}

		ink := check.ink
		if countPixels(key, key.Bounds(), func(colour color.RGBA) bool { return colour == ink }) == 0 {
			t.Errorf("%s: want the chevron and words drawn in %v", name, check.ink)
		}
	}
}
