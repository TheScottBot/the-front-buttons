package render_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// testKeyPixels is the XL's key size as read from source in the Plan stage. TE5's
// model table becomes the real source of this value.
const testKeyPixels = 96

// lostLabel and unknownIconName make a key whose icon cannot be drawn.
const (
	lostLabel       = "Lost"
	unknownIconName = "no-such-icon"
)

// knownTestIconName is the one icon the fake icon set has.
const knownTestIconName = "test-circle"

// iconSetTestDouble has exactly one icon, a filled circle, so tests can tell a
// drawn icon from a missing one without the real icon font.
type iconSetTestDouble struct{}

func (iconSetTestDouble) IconMask(iconName string, sizePixels int) (*image.Alpha, bool) {
	if iconName != knownTestIconName {
		return nil, false
	}

	mask := image.NewAlpha(image.Rect(0, 0, sizePixels, sizePixels))
	radius := sizePixels / 2

	for y := range sizePixels {
		for x := range sizePixels {
			deltaX, deltaY := x-radius, y-radius
			if deltaX*deltaX+deltaY*deltaY <= radius*radius {
				mask.SetAlpha(x, y, color.Alpha{A: 0xff})
			}
		}
	}

	return mask, true
}

func newTestRenderer(t *testing.T) *render.Renderer {
	t.Helper()

	renderer, rendererError := render.NewRenderer(testKeyPixels, iconSetTestDouble{})
	if rendererError != nil {
		t.Fatalf("building the renderer: %v", rendererError)
	}

	return renderer
}

// compareWithGolden checks a rendered key against its golden image through the
// shared helper.
func compareWithGolden(t *testing.T, goldenName string, rendered image.Image) {
	t.Helper()

	testsupport.CompareWithGolden(t, goldenName, rendered)
}

// luminanceMask reduces an image to which pixels are light, removing colour
// entirely, as a viewer with no colour vision sees it.
func luminanceMask(rendered image.Image, region image.Rectangle) []bool {
	const lightThreshold = 0x80

	mask := make([]bool, 0, region.Dx()*region.Dy())

	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			gray, _ := color.GrayModel.Convert(rendered.At(x, y)).(color.Gray)
			mask = append(mask, gray.Y >= lightThreshold)
		}
	}

	return mask
}

func differingCount(firstMask []bool, secondMask []bool) int {
	differing := 0

	for pixelIndex := range firstMask {
		if firstMask[pixelIndex] != secondMask[pixelIndex] {
			differing++
		}
	}

	return differing
}
