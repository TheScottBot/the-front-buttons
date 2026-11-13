package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// ellipsis ends a label shortened to fit a key.
const ellipsis = "…"

// typographicDotsPerInch makes a face's size in points equal its size in pixels.
const typographicDotsPerInch = 72

// textFaces holds every face the renderer draws with. A label is tried at each
// label size from largest to smallest before it is shortened, so a label that
// fits at a smaller size keeps every letter.
type textFaces struct {
	labelSizes []font.Face
	wordSizes  []font.Face
	pageNumber font.Face
	question   font.Face
}

// faceSizes are the sizes, as fractions of the key's width in hundredths. They
// are provisional until the TE4 legibility gate on the physical deck.
type faceSizes struct {
	labelLargest  int
	labelSmallest int
	labelStep     int
	word          int
	pageNumber    int
	question      int
}

// The provisional face sizes in hundredths of the key's width, pending the TE4
// legibility gate.
const (
	labelLargestHundredths  = 17
	labelSmallestHundredths = 12
	labelStepHundredths     = 1
	wordHundredths          = 15
	pageNumberHundredths    = 26
	questionHundredths      = 44
)

func provisionalFaceSizes() faceSizes {
	return faceSizes{
		labelLargest:  labelLargestHundredths,
		labelSmallest: labelSmallestHundredths,
		labelStep:     labelStepHundredths,
		word:          wordHundredths,
		pageNumber:    pageNumberHundredths,
		question:      questionHundredths,
	}
}

func loadFaces(keyPixels int) (textFaces, error) {
	medium, mediumError := opentype.Parse(gomedium.TTF)
	bold, boldError := opentype.Parse(gobold.TTF)

	if mediumError != nil || boldError != nil {
		return textFaces{labelSizes: nil, wordSizes: nil, pageNumber: nil, question: nil},
			fmt.Errorf("parsing the compiled in Go fonts: %w %w", mediumError, boldError)
	}

	sizes := provisionalFaceSizes()

	makeFace := func(parsed *opentype.Font, hundredths int) (font.Face, error) {
		face, faceError := opentype.NewFace(parsed, &opentype.FaceOptions{
			Size:    float64(keyPixels*hundredths) / percent,
			DPI:     typographicDotsPerInch,
			Hinting: font.HintingFull,
		})
		if faceError != nil {
			return nil, fmt.Errorf("building a %d hundredths face: %w", hundredths, faceError)
		}

		return face, nil
	}

	faces := textFaces{labelSizes: nil, wordSizes: nil, pageNumber: nil, question: nil}

	var sizeError error

	faces.labelSizes, sizeError = makeFaceRange(makeFace, medium, sizes.labelLargest, sizes.labelSmallest, sizes.labelStep)
	if sizeError != nil {
		return faces, sizeError
	}

	faces.wordSizes, sizeError = makeFaceRange(makeFace, bold, sizes.word, sizes.labelSmallest, sizes.labelStep)
	if sizeError != nil {
		return faces, sizeError
	}

	var pageError, questionError error

	faces.pageNumber, pageError = makeFace(bold, sizes.pageNumber)
	faces.question, questionError = makeFace(bold, sizes.question)

	if pageError != nil || questionError != nil {
		return faces, fmt.Errorf("building faces: %w %w", pageError, questionError)
	}

	return faces, nil
}

// makeFaceRange builds faces from the largest size down to the smallest, the
// order fitText tries them in.
func makeFaceRange(
	makeFace func(parsed *opentype.Font, hundredths int) (font.Face, error),
	parsed *opentype.Font,
	largest int,
	smallest int,
	step int,
) ([]font.Face, error) {
	var faces []font.Face

	for hundredths := largest; hundredths >= smallest; hundredths -= step {
		face, faceError := makeFace(parsed, hundredths)
		if faceError != nil {
			return nil, faceError
		}

		faces = append(faces, face)
	}

	return faces, nil
}

// fitText returns the text as drawn in the first face it fits, or, if it fits in
// none, shortened with an ellipsis in the smallest face. It returns the face and
// the drawn width in pixels.
//
//nolint:ireturn // font.Face is the font library's own abstraction over faces
func fitText(text string, faces []font.Face, availableWidth int) (string, font.Face, int) {
	for _, face := range faces {
		if width := measure(face, text); width <= availableWidth {
			return text, face, width
		}
	}

	smallest := faces[len(faces)-1]
	runes := []rune(text)

	for keptRunes := len(runes) - 1; keptRunes > 0; keptRunes-- {
		shortened := strings.TrimRight(string(runes[:keptRunes]), " ") + ellipsis
		if width := measure(smallest, shortened); width <= availableWidth {
			return shortened, smallest, width
		}
	}

	return ellipsis, smallest, measure(smallest, ellipsis)
}

func measure(face font.Face, text string) int {
	return font.MeasureString(face, text).Ceil()
}

// drawCentredText draws text centred horizontally with its baseline at baselineY.
// The ink is centred, not the advance: the advance rounded up, its spare space
// rounded down and the letters' uneven side bearings left a label up to a pixel
// left of the icon above it (evaluation, deck testing). The start is rounded to
// a whole pixel so hinted letters stay sharp.
func drawCentredText(destination *image.RGBA, face font.Face, text string, colour color.RGBA, baselineY int) {
	ink, _ := font.BoundString(face, text)
	left := (fixed.I(destination.Bounds().Dx()) - ink.Min.X - ink.Max.X) / centringDivisor
	drawer := font.Drawer{
		Dst:  destination,
		Src:  image.NewUniform(colour),
		Face: face,
		Dot:  fixed.P(left.Round(), baselineY),
	}
	drawer.DrawString(text)
}
