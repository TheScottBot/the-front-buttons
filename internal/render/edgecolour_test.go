package render_test

import (
	"image"
	"image/color"
	"math/rand/v2"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The size of the synthetic icons and the transparent margin around their
// shape, as a macOS icon leaves around its rounded square.
const (
	syntheticIconPixels = 64
	syntheticMargin     = 8
)

// mailLabel is the label the rendered keys carry.
const mailLabel = "Mail"

// Colours the synthetic icons are painted in.
var (
	edgeBlue    = color.NRGBA{R: 0x1e, G: 0x6f, B: 0xd9, A: 0xff} //nolint:gochecknoglobals // a fixed test colour
	centreWhite = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} //nolint:gochecknoglobals // a fixed test colour
	softShadow  = color.NRGBA{R: 0, G: 0, B: 0, A: 0x40}          //nolint:gochecknoglobals // a fixed test colour
)

// squareIcon is a square of the edge colour inside a margin holding a faint
// shadow, as a macOS icon is drawn, with an optional differently coloured
// centre.
func squareIcon(edge color.NRGBA, centre *color.NRGBA) *image.NRGBA {
	icon := image.NewNRGBA(image.Rect(0, 0, syntheticIconPixels, syntheticIconPixels))

	for row := range syntheticIconPixels {
		for column := range syntheticIconPixels {
			icon.SetNRGBA(column, row, squareIconPixel(column, row, edge, centre))
		}
	}

	return icon
}

// squareIconPixel is one pixel of squareIcon: the centre, the shape, the
// shadow just outside it, or nothing.
func squareIconPixel(column int, row int, edge color.NRGBA, centre *color.NRGBA) color.NRGBA {
	shapeMinimum, shapeMaximum := syntheticMargin, syntheticIconPixels-syntheticMargin

	switch {
	case centre != nil && within(column, row, syntheticIconPixels/4, syntheticIconPixels*3/4):
		return *centre
	case within(column, row, shapeMinimum, shapeMaximum):
		return edge
	case within(column, row, shapeMinimum-2, shapeMaximum+2):
		return softShadow
	default:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	}
}

// within reports whether a pixel lies in the square from minimum up to, but
// not including, maximum on both axes.
func within(column int, row int, minimum int, maximum int) bool {
	return column >= minimum && column < maximum && row >= minimum && row < maximum
}

func closeTo(got color.RGBA, want color.NRGBA, tolerance int) bool {
	near := func(first uint8, second uint8) bool {
		difference := int(first) - int(second)

		return difference <= tolerance && difference >= -tolerance
	}

	return got.A == 0xff && near(got.R, want.R) && near(got.G, want.G) && near(got.B, want.B)
}

func TestEdgeColour_ReadsASolidSquare_PastItsTransparentMarginAndShadow(t *testing.T) {
	t.Parallel()

	colour, found := render.EdgeColour(squareIcon(edgeBlue, nil))
	if !found || !closeTo(colour, edgeBlue, 0) {
		t.Fatalf("want the square's blue, got %v (found %v)", colour, found)
	}
}

func TestEdgeColour_ReadsTheEdge_NotADifferentlyColouredCentre(t *testing.T) {
	t.Parallel()

	white := centreWhite

	colour, found := render.EdgeColour(squareIcon(edgeBlue, &white))
	if !found || !closeTo(colour, edgeBlue, 0) {
		t.Fatalf("want the edge's blue, not the centre's white, got %v (found %v)", colour, found)
	}
}

func TestEdgeColour_FindsNothing_InAFullyTransparentImage(t *testing.T) {
	t.Parallel()

	_, found := render.EdgeColour(image.NewNRGBA(image.Rect(0, 0, syntheticIconPixels, syntheticIconPixels)))
	if found {
		t.Fatal("want no edge colour for an image with nothing visible in it")
	}
}

// TestEdgeColour_TakesTheDominantColour_OfANoisyEdge: an edge with a fifth of
// its pixels speckled in random colours still reads as its main colour, which
// a plain average would not.
func TestEdgeColour_TakesTheDominantColour_OfANoisyEdge(t *testing.T) {
	t.Parallel()

	icon := squareIcon(edgeBlue, nil)
	source := rand.NewChaCha8([32]byte{1, 2})
	noise := rand.New(source) //nolint:gosec // a repeatable test pattern, not a secret

	for row := range syntheticIconPixels {
		for column := range syntheticIconPixels {
			if icon.NRGBAAt(column, row).A != 0xff || noise.IntN(5) != 0 {
				continue
			}

			var channels [3]byte

			_, _ = source.Read(channels[:])
			icon.SetNRGBA(column, row, color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: 0xff})
		}
	}

	colour, found := render.EdgeColour(icon)
	if !found || !closeTo(colour, edgeBlue, 0x10) {
		t.Fatalf("want about the edge's blue, got %v (found %v)", colour, found)
	}
}

func TestEdgeColour_ReadsAnImageWhoseBoundsDoNotStartAtZero(t *testing.T) {
	t.Parallel()

	offset := squareIcon(edgeBlue, nil).SubImage(image.Rect(4, 4, syntheticIconPixels, syntheticIconPixels))

	colour, found := render.EdgeColour(offset)
	if !found || !closeTo(colour, edgeBlue, 0) {
		t.Fatalf("want the square's blue, got %v (found %v)", colour, found)
	}
}

// cornerColour reads a rendered key's top left pixel, which no icon or label
// reaches, so it is the key's background.
func cornerColour(rendered image.Image) color.RGBA {
	corner, _ := color.RGBAModel.Convert(rendered.At(rendered.Bounds().Min.X, rendered.Bounds().Min.Y)).(color.RGBA)

	return corner
}

func TestRender_ColoursTheKeyToMatchTheIconsEdge_WhenAsked(t *testing.T) {
	t.Parallel()

	rendered := newTestRenderer(t).Render(render.Appearance{
		Label: mailLabel, IconImage: squareIcon(edgeBlue, nil), BackgroundFromIcon: true,
	})

	if corner := cornerColour(rendered); !closeTo(corner, edgeBlue, 0) {
		t.Fatalf("want the key coloured the icon's blue, got %v", corner)
	}
}

func TestRender_KeepsAnExplicitBackground_OverTheIconsEdge(t *testing.T) {
	t.Parallel()

	explicit := color.RGBA{R: 0x80, G: 0x10, B: 0x10, A: 0xff}

	rendered := newTestRenderer(t).Render(render.Appearance{
		Label: mailLabel, IconImage: squareIcon(edgeBlue, nil), BackgroundFromIcon: true, Background: explicit,
	})

	if corner := cornerColour(rendered); corner != explicit {
		t.Fatalf("want the explicit background %v, got %v", explicit, corner)
	}
}

func TestRender_UsesTheDefaultBackground_WhenTheIconHasNoVisibleEdge(t *testing.T) {
	t.Parallel()

	rendered := newTestRenderer(t).Render(render.Appearance{
		Label:              mailLabel,
		IconImage:          image.NewNRGBA(image.Rect(0, 0, syntheticIconPixels, syntheticIconPixels)),
		BackgroundFromIcon: true,
	})

	if corner := cornerColour(rendered); corner != (color.RGBA{R: 0, G: 0, B: 0, A: 0xff}) {
		t.Fatalf("want the default black background, got %v", corner)
	}
}

// TestRender_KeepsTheLabelLegible_OnAPaleIconsEdge: the label's colour follows
// whatever background the icon gives the key, so a pale icon gets dark text.
func TestRender_KeepsTheLabelLegible_OnAPaleIconsEdge(t *testing.T) {
	t.Parallel()

	pale := color.NRGBA{R: 0xf2, G: 0xf2, B: 0xf2, A: 0xff}
	rendered := newTestRenderer(t).Render(render.Appearance{
		Label: "Notes", IconImage: squareIcon(pale, nil), BackgroundFromIcon: true,
	})

	if corner := cornerColour(rendered); !closeTo(corner, pale, 0) {
		t.Fatalf("want the key coloured the icon's pale grey, got %v", corner)
	}

	wantForeground := render.ForegroundFor(color.RGBA{R: pale.R, G: pale.G, B: pale.B, A: 0xff})
	if wantForeground != (color.RGBA{R: 0, G: 0, B: 0, A: 0xff}) {
		t.Fatalf("want black chosen for a pale key, got %v", wantForeground)
	}

	labelRows := image.Rect(0, testKeyPixels*3/4, testKeyPixels, testKeyPixels)

	for y := labelRows.Min.Y; y < labelRows.Max.Y; y++ {
		for x := labelRows.Min.X; x < labelRows.Max.X; x++ {
			pixel, _ := color.RGBAModel.Convert(rendered.At(x, y)).(color.RGBA)
			if pixel == wantForeground {
				return
			}
		}
	}

	t.Fatalf("want the label drawn in %v on the pale key", wantForeground)
}
