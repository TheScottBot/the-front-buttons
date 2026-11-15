package render_test

import (
	"image"
	"image/color"
	"testing"
)

// Thyme's colours, from its theme, which the clock keys take: its panel's
// navy, its time's pink, and its zone labels' dimmer pink.
func clockNavy() color.RGBA      { return color.RGBA{R: 0x12, G: 0x13, B: 0x1c, A: 0xff} }
func clockPink() color.RGBA      { return color.RGBA{R: 0xf8, G: 0x94, B: 0xf8, A: 0xff} }
func clockLabelPink() color.RGBA { return color.RGBA{R: 0xd5, G: 0x83, B: 0xd7, A: 0xff} }

// inkBounds is the box of every pixel not the background.
func inkBounds(rendered image.Image, background color.RGBA) image.Rectangle {
	ink := image.Rect(0, 0, 0, 0)
	bounds := rendered.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if color.RGBAModel.Convert(rendered.At(x, y)) != background {
				ink = ink.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}

	return ink
}

// hasColour reports whether any pixel is exactly a colour.
func hasColour(rendered image.Image, want color.RGBA) bool {
	bounds := rendered.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if color.RGBAModel.Convert(rendered.At(x, y)) == want {
				return true
			}
		}
	}

	return false
}

// requireCentred fails unless the ink sits in the middle of the key, both
// ways, to within a few pixels, and well clear of its edges.
func requireCentred(t *testing.T, name string, rendered image.Image) {
	t.Helper()

	const slack = 4

	ink := inkBounds(rendered, clockNavy())
	if ink.Empty() {
		t.Fatalf("%s: want something drawn", name)
	}

	left, right := ink.Min.X, testKeyPixels-ink.Max.X
	top, bottom := ink.Min.Y, testKeyPixels-ink.Max.Y

	if diff := left - right; diff > slack || diff < -slack || left < slack {
		t.Errorf("%s: want it centred across, %d left and %d right", name, left, right)
	}

	if diff := top - bottom; diff > slack || diff < -slack || top < slack {
		t.Errorf("%s: want it centred down, %d above and %d below", name, top, bottom)
	}
}

// TestRenderClock_DrawsADigitLargeInThymesColours: a clock digit is one
// large figure, pink on navy, in the middle of its key, each figure its own.
func TestRenderClock_DrawsADigitLargeInThymesColours(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	five := renderer.RenderClockDigit("5")

	if got := color.RGBAModel.Convert(five.At(1, 1)); got != clockNavy() {
		t.Errorf("want a navy key, got %v", got)
	}

	if !hasColour(five, clockPink()) {
		t.Error("want the figure drawn pink")
	}

	requireCentred(t, "5", five)

	if ink := inkBounds(five, clockNavy()); ink.Dy() < testKeyPixels/2 {
		t.Errorf("want the figure at least half the key high, got %d", ink.Dy())
	}

	if distanceBetween(five, renderer.RenderClockDigit("8")) == 0 {
		t.Error("want each figure drawn as itself")
	}
}

// TestRenderClock_DrawsALabelAndAnIcon: a clock's label, such as a time
// zone's, is drawn in Thyme's dimmer label pink and its icon in the time's
// pink, each in the middle of the key.
func TestRenderClock_DrawsALabelAndAnIcon(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)

	for name, check := range map[string]struct {
		rendered image.Image
		colour   color.RGBA
	}{
		"label": {renderer.RenderClockLabel("UTC"), clockLabelPink()},
		"icon":  {renderer.RenderClockIcon(knownTestIconName), clockPink()},
	} {
		if !hasColour(check.rendered, check.colour) {
			t.Errorf("%s: want it drawn in %v", name, check.colour)
		}

		requireCentred(t, name, check.rendered)
	}

	if ink := inkBounds(renderer.RenderClockIcon(unknownIconName), clockNavy()); !ink.Empty() {
		t.Error("want an icon there is none of drawn as a plain navy key")
	}
}

// distanceBetween counts the pixels two images differ at.
func distanceBetween(first image.Image, second image.Image) int {
	differing := 0
	bounds := first.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if color.RGBAModel.Convert(first.At(x, y)) != color.RGBAModel.Convert(second.At(x, y)) {
				differing++
			}
		}
	}

	return differing
}
