package render_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The badges the tests draw: a count and Discord's pip.
const (
	countBadge = "12"
	pipBadge   = "•"
)

// badgeRed is the red of a Dock badge.
var badgeRed = color.RGBA{R: 0xff, G: 0x3b, B: 0x30, A: 0xff} //nolint:gochecknoglobals // a fixed colour

// redPixels counts the badge's red pixels in an area of a key.
func redPixels(key image.Image, area image.Rectangle) int {
	count := 0

	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if color.RGBAModel.Convert(key.At(x, y)) == badgeRed {
				count++
			}
		}
	}

	return count
}

// TestRender_DrawsABadgeTopRight: a key showing its application's badge has
// it top right in the Dock's red, a count with its number and a pip as a dot,
// and the rest of the key as it was.
func TestRender_DrawsABadgeTopRight(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)
	plain := render.Appearance{Label: docsLabel}

	for goldenName, badge := range map[string]string{"badge_count": countBadge, "badge_pip": pipBadge} {
		badged := plain
		badged.Badge = badge
		rendered := renderer.Render(badged)

		compareWithGolden(t, goldenName, rendered)

		half := testKeyPixels / 2
		if redPixels(rendered, image.Rect(half, 0, testKeyPixels, half)) == 0 {
			t.Errorf("%s: want the badge in the top right quarter", goldenName)
		}

		if redPixels(rendered, image.Rect(0, 0, half, testKeyPixels)) != 0 ||
			redPixels(rendered, image.Rect(half, half, testKeyPixels, testKeyPixels)) != 0 {
			t.Errorf("%s: want nothing red outside the top right quarter", goldenName)
		}
	}

	// A count is wider than a pip, and the count shows its number in white.
	count, pip := plain, plain
	count.Badge, pip.Badge = countBadge, pipBadge
	topRight := image.Rect(testKeyPixels/2, 0, testKeyPixels, testKeyPixels/2)

	if redPixels(renderer.Render(count), topRight) <= redPixels(renderer.Render(pip), topRight) {
		t.Error("want a count's badge bigger than a pip")
	}
}

// TestRender_ErrorsAndStatusesOutrankABadge: a key in error or showing a
// status draws that alone; the badge waits until the key is plain again.
func TestRender_ErrorsAndStatusesOutrankABadge(t *testing.T) {
	t.Parallel()

	renderer := newTestRenderer(t)

	for name, appearance := range map[string]render.Appearance{
		"error":  {Label: buildLabel, ErrorReason: networkReason, Badge: countBadge},
		"status": {Label: buildLabel, Status: render.StatusSuccess, Badge: countBadge},
	} {
		without := appearance
		without.Badge = ""

		if !testsupport.SameImage(renderer.Render(appearance), renderer.Render(without)) {
			t.Errorf("%s: want the badge left off", name)
		}
	}
}
