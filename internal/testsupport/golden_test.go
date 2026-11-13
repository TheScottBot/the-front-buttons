package testsupport_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// grey is a small image of one shade.
func grey(level uint8) *image.RGBA {
	picture := image.NewRGBA(image.Rect(0, 0, 4, 4))

	for y := range 4 {
		for x := range 4 {
			picture.Set(x, y, color.RGBA{R: level, G: level, B: level, A: 0xFF})
		}
	}

	return picture
}

// TestDifferingPixels_IgnoresRoundingButNotDrawing: the goldens are made on
// the author's Mac, an arm64 machine, and CI checks them on Windows, an amd64
// one, where the icon rasteriser's arithmetic rounds a curve's edge a shade
// differently; the brand-github golden failed on four pixels so (evaluation,
// CI). A shade or two either way is not a change to what is drawn; a pixel
// moved is.
func TestDifferingPixels_IgnoresRoundingButNotDrawing(t *testing.T) {
	t.Parallel()

	const (
		middle   = 0x80
		rounding = 2
	)

	golden := grey(middle)

	// Every pixel a couple of shades off, as rounding leaves them, is the same.
	if differing, largest := testsupport.DifferingPixels(golden, grey(middle+rounding)); differing != 0 ||
		largest != rounding {
		t.Errorf("want shades %d off counted alike, got %d differing by %d", rounding, differing, largest)
	}

	// One pixel just past the tolerance and one far off are both different.
	drawn := grey(middle)
	drawn.Set(1, 2, color.RGBA{R: middle + testsupport.ShadeTolerance + 1, G: middle, B: middle, A: 0xFF})
	drawn.Set(3, 3, color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})

	differing, largest := testsupport.DifferingPixels(golden, drawn)
	if differing != 2 || largest != 0xFF-middle {
		t.Errorf("want 2 pixels differing, the largest by %d, got %d by %d", 0xFF-middle, differing, largest)
	}

	// The tolerance stays well below a moved edge, which changes a shade by
	// tens or hundreds.
	if testsupport.ShadeTolerance < rounding || testsupport.ShadeTolerance > 16 {
		t.Errorf("want a tolerance between %d and 16 shades, got %d", rounding, testsupport.ShadeTolerance)
	}
}
