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
	docsLabel     = "Docs"
	buildLabel    = "Build"
	networkReason = "network"
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
		"missing_icon":            {Label: lostLabel, Icon: unknownIconName},
		"long_label":              {Label: "A label far too long to fit on one key"},
		"user_icon":               {Label: "Mine", IconImage: userIcon},
		"light_background":        {Label: "Light", Background: color.RGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}},
		"error_state":             {Label: buildLabel, ErrorReason: networkReason},
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

	appearances := map[string]render.Appearance{"error": {Label: buildLabel, ErrorReason: networkReason}}
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

// inkRows returns the first and last rows holding a pixel that differs from the
// key's background, the drawn content's vertical extent.
func inkRows(rendered image.Image) (int, int) {
	bounds := rendered.Bounds()
	background := rendered.At(bounds.Min.X, bounds.Min.Y)
	first, last := -1, -1

	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for column := bounds.Min.X; column < bounds.Max.X; column++ {
			if rendered.At(column, row) != background {
				if first < 0 {
					first = row
				}

				last = row

				break
			}
		}
	}

	return first, last
}

// TestRender_CentresAnIconAndLabelVertically_WhenThereIsNoStatusWord checks the
// space above an icon key's content against the space below it. The first icons
// seen on a real deck sat high, about a sixteenth of the key from the top and a
// quarter from the bottom, because their layout kept room for a status word
// they never show (evaluation, deck testing).
func TestRender_CentresAnIconAndLabelVertically_WhenThereIsNoStatusWord(t *testing.T) {
	t.Parallel()

	const allowedImbalancePixels = 4

	renderer := newTestRenderer(t)

	for _, appearance := range []render.Appearance{
		{Label: buildLabel, Icon: knownTestIconName},
		{Label: lostLabel, Icon: unknownIconName},
		{Icon: knownTestIconName},
	} {
		first, last := inkRows(renderer.Render(appearance))
		above, below := first, testKeyPixels-1-last

		if above-below > allowedImbalancePixels || below-above > allowedImbalancePixels {
			t.Errorf("%+v: %d pixels above the content and %d below, want within %d of each other",
				appearance, above, below, allowedImbalancePixels)
		}
	}
}

// TestRender_DrawsTheMissingMarker_ForAnIconFileThatCouldNotBeLoaded: a key
// whose icon_file is missing or unreadable looks like a key whose bundled icon
// name is unknown, so both mistakes read the same on the deck.
func TestRender_DrawsTheMissingMarker_ForAnIconFileThatCouldNotBeLoaded(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)

	unloadable := renderer.Render(render.Appearance{Label: lostLabel, IconMissing: true})
	unknownName := renderer.Render(render.Appearance{Label: lostLabel, Icon: unknownIconName})

	if differing := differingCount(luminanceMask(unloadable, unloadable.Bounds()),
		luminanceMask(unknownName, unknownName.Bounds())); differing != 0 {
		t.Fatalf("want the missing marker drawn, %d pixels differ", differing)
	}
}

// The frame the framed renderer tests use: an 84 pixel square, 8 from the left
// and 10 from the top of a 96 pixel key.
var testInset = render.Inset{ //nolint:gochecknoglobals // a fixed test value
	Left: 8, Top: 9, ContentPixels: 82, Bulge: 1.5, CornerRadius: 6,
}

func framedRenderer(t *testing.T) *render.Renderer {
	t.Helper()

	renderer, rendererError := render.NewFramedRenderer(testKeyPixels, iconSetTestDouble{}, testInset)
	if rendererError != nil {
		t.Fatal(rendererError)
	}

	return renderer
}

func isBlack(colour color.Color) bool {
	red, green, blue, _ := colour.RGBA()

	return red == 0 && green == 0 && blue == 0
}

// TestFramedRenderer_ContainsEveryKeyInABlackSquare: the XL draws each key's
// image a little up and left of its window and its edges bow, so a light key
// showed a black strip at the bottom and right only. The author asked for every
// key to sit in a black square instead (evaluation, deck testing): the content
// is drawn smaller, placed by the deck's measured inset, with black around it.
func TestFramedRenderer_ContainsEveryKeyInABlackSquare(t *testing.T) {
	t.Parallel()

	renderer := framedRenderer(t)
	light := color.RGBA{R: 0x2E, G: 0xE6, B: 0xFF, A: 0xFF}

	keys := map[string]image.Image{
		"key": renderer.Render(render.Appearance{Label: buildLabel, Background: light}),
		"navigation": renderer.RenderNavigation(render.DirectionForward, layout.PageIndication{
			Number: 1, Count: 1, CurrentName: "", PreviousName: "", NextName: "",
		}, false),
		"blank": renderer.RenderBlank(),
	}

	inner, bulge := testInset.ContentPixels, int(math.Ceil(testInset.Bulge))
	middle := testInset.Left + inner/2
	frame := []image.Point{
		{X: 0, Y: 0},
		{X: testInset.Left - bulge - 1, Y: middle},
		{X: middle, Y: testInset.Top - bulge - 1},
		{X: testInset.Left + inner + bulge, Y: middle},
		{X: middle, Y: testInset.Top + inner + bulge},
		{X: 95, Y: 95},
		{X: testInset.Left, Y: testInset.Top},
		{X: testInset.Left + inner - 1, Y: testInset.Top + inner - 1},
	}

	for name, key := range keys {
		if key.Bounds().Dx() != testKeyPixels || key.Bounds().Dy() != testKeyPixels {
			t.Fatalf("%s: want a %d pixel key, got %v", name, testKeyPixels, key.Bounds())
		}

		for _, point := range frame {
			if !isBlack(key.At(point.X, point.Y)) {
				t.Errorf("%s: want black frame at %v, got %v", name, point, key.At(point.X, point.Y))
			}
		}
	}

	inside := image.Point{X: testInset.Left + testInset.CornerRadius, Y: testInset.Top + testInset.CornerRadius}
	if got := keys["key"].At(inside.X, inside.Y); got != color.Color(light) {
		t.Errorf("want the key's own colour inside the frame, got %v", got)
	}

	// The key cap pulls each edge in at its middle, so the edges are drawn
	// bulging out there, by a pixel and a half, to look straight through it.
	// The pixel the edge crosses is blended with black, not stepped: a hard
	// one pixel staircase showed through the cap (evaluation, deck testing).
	assertBlended(t, keys["key"], light, []image.Point{
		{X: testInset.Left - bulge, Y: middle},
		{X: middle, Y: testInset.Top - bulge},
		{X: testInset.Left + inner + bulge - 1, Y: middle},
		{X: middle, Y: testInset.Top + inner + bulge - 1},
	})

	if isBlack(keys["navigation"].At(testInset.Left+3, testInset.Top+3)) {
		t.Error("want the navigation key drawn inside the frame")
	}
}

// assertBlended checks each point is part way between black and the colour.
func assertBlended(t *testing.T, key image.Image, full color.RGBA, points []image.Point) {
	t.Helper()

	for _, point := range points {
		got, _ := color.RGBAModel.Convert(key.At(point.X, point.Y)).(color.RGBA)
		if got.B == 0 || got.B >= full.B {
			t.Errorf("want the edge at %v blended between black and the key's colour, got %v", point, got)
		}
	}
}

func TestNewFramedRenderer_RefusesAFrameThatDoesNotFit(t *testing.T) {
	t.Parallel()

	for name, inset := range map[string]render.Inset{
		"too small to draw in": {Left: 0, Top: 0, ContentPixels: 40, Bulge: 0, CornerRadius: 0},
		"off the right":        {Left: 20, Top: 0, ContentPixels: 84, Bulge: 0, CornerRadius: 0},
		"off the bottom":       {Left: 0, Top: 20, ContentPixels: 84, Bulge: 0, CornerRadius: 0},
		"negative":             {Left: -1, Top: 0, ContentPixels: 84, Bulge: 0, CornerRadius: 0},
		"bulge past the edge":  {Left: 2, Top: 8, ContentPixels: 82, Bulge: 3, CornerRadius: 0},
		"corner too round":     {Left: 8, Top: 8, ContentPixels: 80, Bulge: 0, CornerRadius: 41},
	} {
		_, rendererError := render.NewFramedRenderer(testKeyPixels, iconSetTestDouble{}, inset)
		if rendererError == nil {
			t.Errorf("%s: want the frame refused", name)
		}
	}
}

// TestRenderer_FillsTheKeyWithAPicture: the album key is its art, edge to
// edge, with no label or icon; an error still shows in its place, so a key
// that cannot read the player says so.
func TestRenderer_FillsTheKeyWithAPicture(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	red := image.NewRGBA(image.Rect(0, 0, 10, 10))

	for y := range 10 {
		for x := range 10 {
			red.Set(x, y, color.RGBA{R: 0xFF, G: 0, B: 0, A: 0xFF})
		}
	}

	key := renderer.Render(render.Appearance{Label: docsLabel, Picture: red})

	for _, point := range []image.Point{{X: 1, Y: 1}, {X: 48, Y: 48}, {X: 94, Y: 94}} {
		got, _ := color.RGBAModel.Convert(key.At(point.X, point.Y)).(color.RGBA)
		if got.R < 0xF0 || got.G > 0x10 || got.B > 0x10 {
			t.Errorf("want the picture at %v, got %v", point, got)
		}
	}

	failed := renderer.Render(render.Appearance{Picture: red, ErrorReason: "permission"})
	if got, _ := color.RGBAModel.Convert(failed.At(48, 48)).(color.RGBA); got.R > 0xF0 && got.G < 0x10 {
		t.Error("want an error shown instead of the picture")
	}
}
