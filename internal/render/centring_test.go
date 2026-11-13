package render_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/render/tablericons"
)

// The XL's measured frame (internal/deck/model.go), so the test draws what the
// deck shows.
func xlInset() render.Inset {
	return render.Inset{Left: 8, Top: 9, ContentPixels: 82, Bulge: 1.5, CornerRadius: 6}
}

// labelRowsStart is the first key row below the icon and above the label, in
// hundredths of the content, splitting the key into the icon's rows and the
// label's.
const labelRowsStart = 64

// centringTolerance is how far, in pixels, ink may sit from the content's centre:
// half a pixel, because ink an odd number of columns wide cannot be centred on
// an even width in whole pixels. The off key's icon was a pixel and a half out.
const centringTolerance = 0.5

// inkColumns reports the first and last columns within the rows that hold light
// pixels.
func inkColumns(rendered image.Image, firstRow int, endRow int) (int, int, bool) {
	const lightThreshold = 0x80

	first, last := rendered.Bounds().Max.X, -1

	for row := firstRow; row < endRow; row++ {
		for column := rendered.Bounds().Min.X; column < rendered.Bounds().Max.X; column++ {
			gray, _ := color.GrayModel.Convert(rendered.At(column, row)).(color.Gray)
			if gray.Y >= lightThreshold {
				first, last = min(first, column), max(last, column)
			}
		}
	}

	return first, last, last >= 0
}

// The author's photograph showed the off key's power icon visibly left of the
// key's centre and its label slightly left too (evaluation, deck testing).
func TestRender_CentresAnIconAndItsLabelHorizontally(t *testing.T) {
	t.Parallel()

	icons, iconsError := tablericons.Load()
	if iconsError != nil {
		t.Fatalf("loading the icons: %v", iconsError)
	}

	framed, framedError := render.NewFramedRenderer(testKeyPixels, icons, xlInset())
	if framedError != nil {
		t.Fatalf("building the framed renderer: %v", framedError)
	}

	plain, plainError := render.NewRenderer(testKeyPixels, icons)
	if plainError != nil {
		t.Fatalf("building the renderer: %v", plainError)
	}

	renderers := map[string]struct {
		renderer *render.Renderer
		inset    render.Inset
	}{
		"framed": {framed, xlInset()},
		"plain":  {plain, render.Inset{Left: 0, Top: 0, ContentPixels: testKeyPixels, Bulge: 0, CornerRadius: 0}},
	}

	appearances := []render.Appearance{
		{Label: "Off", Icon: "power"},
		{Label: "F13", Icon: "keyboard"},
		{Label: "Wand", Icon: "wand"},
		{Label: "Mute", Icon: "circle"},
	}

	for rendererName, under := range renderers {
		inset := under.inset
		centre := float64(inset.Left) + float64(inset.ContentPixels)/2
		labelRow := inset.Top + inset.ContentPixels*labelRowsStart/100

		for _, appearance := range appearances {
			rendered := under.renderer.Render(appearance)

			parts := map[string][2]int{
				"icon":  {inset.Top, labelRow},
				"label": {labelRow, inset.Top + inset.ContentPixels},
			}

			for partName, rows := range parts {
				first, last, found := inkColumns(rendered, rows[0], rows[1])
				if !found {
					t.Fatalf("%s %s %q: no ink", rendererName, partName, appearance.Label)
				}

				inkCentre := float64(first+last+1) / 2
				if inkCentre < centre-centringTolerance || inkCentre > centre+centringTolerance {
					t.Errorf("%s %s of %q/%s: ink spans columns %d to %d, centre %.1f, want %.1f within %.1f",
						rendererName, partName, appearance.Label, appearance.Icon, first, last, inkCentre,
						centre, centringTolerance)
				}
			}
		}
	}
}
