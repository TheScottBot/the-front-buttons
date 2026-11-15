package render_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// holdPink is the pink of a hold's bar, the clock's.
var holdPink = color.RGBA{R: 0xf8, G: 0x94, B: 0xf8, A: 0xff} //nolint:gochecknoglobals // a fixed colour

// countPixels counts the pixels in an area of a key that pass a test.
func countPixels(key image.Image, area image.Rectangle, counted func(colour color.RGBA) bool) int {
	count := 0

	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			colour, _ := color.RGBAModel.Convert(key.At(x, y)).(color.RGBA)
			if counted(colour) {
				count++
			}
		}
	}

	return count
}

// isPink reports the hold bar's pink.
func isPink(colour color.RGBA) bool { return colour == holdPink }

// isLight reports a pixel nearer white than black.
func isLight(colour color.RGBA) bool { return colour.R > 0x80 && colour.G > 0x80 && colour.B > 0x80 }

// TestRenderer_PressedDimsTheKeyAndPushesItIn: a key pressed is drawn darker
// and a little smaller, black showing round its edge, so the press is seen.
func TestRenderer_PressedDimsTheKeyAndPushesItIn(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	white := render.Appearance{Background: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}}
	pressed := renderer.Pressed(renderer.Render(white), 0)

	if pressed.Bounds() != image.Rect(0, 0, testKeyPixels, testKeyPixels) {
		t.Fatalf("want the key's own size, got %v", pressed.Bounds())
	}

	centre, _ := color.RGBAModel.Convert(pressed.At(testKeyPixels/2, testKeyPixels/2)).(color.RGBA)
	if centre.R < 0x60 || centre.R > 0xb0 {
		t.Errorf("want the middle dimmed from white to a mid grey, got %v", centre)
	}

	corner, _ := color.RGBAModel.Convert(pressed.At(1, 1)).(color.RGBA)
	if corner.R > 0x10 {
		t.Errorf("want black at the edge of a key pushed in, got %v", corner)
	}
}

// TestRenderer_PressedDrawsAHoldsBarAcrossItsBottom: a key held towards a
// hold shows a bar along its bottom, filled in pink from the left as far as
// the hold has gone, and no bar while it is not held towards one.
func TestRenderer_PressedDrawsAHoldsBarAcrossItsBottom(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	key := renderer.Render(render.Appearance{Label: docsLabel})
	half := testKeyPixels / 2
	bottomLeft := image.Rect(0, testKeyPixels*4/5, half, testKeyPixels)
	bottomRight := image.Rect(half, testKeyPixels*4/5, testKeyPixels, testKeyPixels)
	notBlack := func(colour color.RGBA) bool { return colour.R+colour.G+colour.B > 0x30 }

	halfway := renderer.Pressed(key, 0.5)
	if countPixels(halfway, bottomLeft, isPink) == 0 {
		t.Error("halfway: want the bar's left half pink")
	}

	if countPixels(halfway, bottomRight, isPink) != 0 {
		t.Error("halfway: want no pink in the bar's right half")
	}

	if countPixels(halfway, bottomRight, notBlack) == 0 {
		t.Error("halfway: want the rest of the bar's track drawn")
	}

	if countPixels(renderer.Pressed(key, 0), image.Rect(0, 0, testKeyPixels, testKeyPixels), isPink) != 0 {
		t.Error("not held towards a hold: want no bar")
	}
}

// TestRender_DrawsHowFarAPressCountHasGone: a key counting presses towards
// an action shows a dot for each press it needs, top left, those made
// filled, and none while no count is under way.
func TestRender_DrawsHowFarAPressCountHasGone(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	topLeft := image.Rect(0, 0, testKeyPixels/2, testKeyPixels/4)
	lit := func(done int) int {
		appearance := render.Appearance{Label: docsLabel, Presses: render.PressCount{Done: done, Of: 3}}

		return countPixels(renderer.Render(appearance), topLeft, isLight)
	}

	none, one, two := lit(0), lit(1), lit(2)
	if none != 0 {
		t.Errorf("no count under way: want nothing top left, got %d light pixels", none)
	}

	if one == 0 || two <= one {
		t.Errorf("want more of the top left lit with each press, got %d then %d", one, two)
	}
}
