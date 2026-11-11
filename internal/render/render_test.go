package render_test

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The labels the tests draw, written once.
const (
	docsLabel  = "Docs"
	buildLabel = "Build"
)

func allStatuses() []render.Status {
	return []render.Status{
		render.StatusSuccess, render.StatusFailure, render.StatusInProgress,
		render.StatusQueued, render.StatusCancelled, render.StatusUnknown,
	}
}

func TestRender_ProducesAKeySizedImage(t *testing.T) {
	t.Parallel()

	rendered := newTestRenderer(t).Render(render.Appearance{Label: docsLabel})

	if rendered.Bounds() != image.Rect(0, 0, testKeyPixels, testKeyPixels) {
		t.Fatalf("want a %d pixel square, got %v", testKeyPixels, rendered.Bounds())
	}
}

func TestRender_MatchesGoldenImages(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	userIcon := image.NewRGBA(image.Rect(0, 0, 20, 10))

	for x := range 20 {
		for y := range 10 {
			userIcon.Set(x, y, color.RGBA{R: 0xff, G: 0x80, B: 0x00, A: 0xff})
		}
	}

	goldenCases := map[string]render.Appearance{
		"label_only":              {Label: docsLabel},
		"label_with_bundled_icon": {Label: buildLabel, Icon: knownTestIconName},
		"missing_icon":            {Label: "Lost", Icon: "no-such-icon"},
		"long_label":              {Label: "A label far too long to fit on one key"},
		"user_icon":               {Label: "Mine", IconImage: userIcon},
		"light_background":        {Label: "Light", Background: color.RGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}},
		"error_state":             {Label: buildLabel, ErrorReason: "network"},
	}

	for _, status := range allStatuses() {
		goldenCases["status_"+string(status)] = render.Appearance{Label: buildLabel, Status: status}
	}

	for goldenName, appearance := range goldenCases {
		t.Run(goldenName, func(t *testing.T) {
			t.Parallel()

			compareWithGolden(t, goldenName, renderer.Render(appearance))
		})
	}
}

func TestRenderNavigation_MatchesGoldenImages(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	indication := layout.PageIndication{Number: 2, Count: 5, CurrentName: "Work", PreviousName: "Home", NextName: "Games"}

	compareWithGolden(t, "navigation_back", renderer.RenderNavigation(render.DirectionBack, indication, true))
	compareWithGolden(t, "navigation_forward", renderer.RenderNavigation(render.DirectionForward, indication, true))
	compareWithGolden(t, "navigation_no_pages", renderer.RenderNavigation(render.DirectionForward, indication, false))
	compareWithGolden(t, "blank", renderer.RenderBlank())
}

func TestRender_EveryStatusIsDistinguishableWithoutColour(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)

	// The glyph area alone, above the text, must tell statuses apart, so the
	// shape carries the meaning even before the word is read.
	glyphArea := image.Rect(0, 0, testKeyPixels, testKeyPixels*55/100)

	// At least this many pixels must differ between any two statuses: roughly
	// two percent of the glyph area, far beyond antialiasing noise.
	const minimumDifferingPixels = 100

	appearances := map[string]render.Appearance{"error": {Label: buildLabel, ErrorReason: "network"}}
	for _, status := range allStatuses() {
		appearances[string(status)] = render.Appearance{Label: buildLabel, Status: status}
	}

	masks := make(map[string][]bool, len(appearances))
	for appearanceName, appearance := range appearances {
		masks[appearanceName] = luminanceMask(renderer.Render(appearance), glyphArea)
	}

	for firstName, firstMask := range masks {
		for secondName, secondMask := range masks {
			if firstName >= secondName {
				continue
			}

			if differing := differingCount(firstMask, secondMask); differing < minimumDifferingPixels {
				t.Errorf("%s and %s differ by only %d pixels without colour", firstName, secondName, differing)
			}
		}
	}
}

func TestStatusWord_IsDistinctForEveryStatus(t *testing.T) {
	t.Parallel()

	seenWords := map[string]render.Status{}

	for _, status := range allStatuses() {
		word := render.StatusWord(status)
		if word == "" {
			t.Errorf("%s has no word", status)
		}

		if earlier, seen := seenWords[word]; seen {
			t.Errorf("%s and %s share the word %q", earlier, status, word)
		}

		seenWords[word] = status
	}
}

// relativeLuminance is the WCAG 2 definition, written here independently of the
// renderer so the test is an oracle rather than a restatement.
func relativeLuminance(colour color.RGBA) float64 {
	channel := func(value uint8) float64 {
		normalised := float64(value) / 0xff
		if normalised <= 0.03928 {
			return normalised / 12.92
		}

		return math.Pow((normalised+0.055)/1.055, 2.4)
	}

	return 0.2126*channel(colour.R) + 0.7152*channel(colour.G) + 0.0722*channel(colour.B)
}

func TestForegroundFor_MeetsWCAGContrastOnEveryBackground(t *testing.T) {
	t.Parallel()

	// WCAG AA for normal text. Black or white always reaches it against some
	// side of any background, so the renderer has no excuse to miss it.
	const minimumContrast = 4.5

	var backgrounds []color.RGBA

	for level := 0; level <= 0xff; level += 5 {
		value := uint8(level)
		backgrounds = append(backgrounds,
			color.RGBA{R: value, G: value, B: value, A: 0xff},
			color.RGBA{R: value, G: 0, B: 0, A: 0xff},
			color.RGBA{R: 0, G: value, B: 0, A: 0xff},
			color.RGBA{R: 0, G: 0, B: value, A: 0xff})
	}

	for _, background := range backgrounds {
		foreground := render.ForegroundFor(background)

		lighter := math.Max(relativeLuminance(foreground), relativeLuminance(background))
		darker := math.Min(relativeLuminance(foreground), relativeLuminance(background))

		if contrast := (lighter + 0.05) / (darker + 0.05); contrast < minimumContrast {
			t.Errorf("background %v with foreground %v has contrast %.2f", background, foreground, contrast)
		}
	}
}

func TestFitText_ShortensALongLabelToFit_WithAnEllipsis(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)

	fitted, fittedWidth := render.FitLabelForTest(renderer, "A label far too long to fit on one key")

	if !strings.HasSuffix(fitted, "…") || fittedWidth > render.LabelWidthForTest(renderer) {
		t.Fatalf("want an ellipsis and a fit, got %q at %d pixels", fitted, fittedWidth)
	}

	short, _ := render.FitLabelForTest(renderer, "Docs")
	if short != "Docs" {
		t.Fatalf("want a short label unchanged, got %q", short)
	}
}

func TestParseColour_AcceptsHashSixHexDigitsOnly(t *testing.T) {
	t.Parallel()

	colour, parseError := render.ParseColour("#102aFf")
	if parseError != nil || colour != (color.RGBA{R: 0x10, G: 0x2a, B: 0xff, A: 0xff}) {
		t.Fatalf("want #102aFf parsed, got %v (%v)", colour, parseError)
	}

	for _, invalid := range []string{"", "red", "#12345", "#1234567", "102030", "#gg0000"} {
		_, invalidError := render.ParseColour(invalid)
		if invalidError == nil {
			t.Errorf("want %q refused", invalid)
		}
	}
}

func TestNewRenderer_RefusesImplausibleKeySizes(t *testing.T) {
	t.Parallel()

	for _, keyPixels := range []int{0, 8, 4096} {
		_, rendererError := render.NewRenderer(keyPixels, iconSetTestDouble{})
		if rendererError == nil {
			t.Errorf("want a %d pixel key refused", keyPixels)
		}
	}
}
