package render

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"regexp"
	"strconv"
)

// ErrBadColour reports a colour not written #RRGGBB.
var ErrBadColour = errors.New("colour must be written #RRGGBB")

var hexColourPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// The WCAG 2 contrast thresholds the renderer meets: 4.5 to 1 for text, and 3 to
// 1 for a status glyph, which is a graphical object (WCAG 1.4.11).
const (
	textContrastMinimum  = 4.5
	glyphContrastMinimum = 3.0
)

// The WCAG 2 relative luminance constants.
const (
	luminanceLinearLimit  = 0.03928
	luminanceLinearScale  = 12.92
	luminanceGammaOffset  = 0.055
	luminanceGammaScale   = 1.055
	luminanceGammaPower   = 2.4
	luminanceRedWeight    = 0.2126
	luminanceGreenWeight  = 0.7152
	luminanceBlueWeight   = 0.0722
	contrastFlare         = 0.05
	maximumChannelValue   = 0xff
	hexColourDigitsOffset = 1
)

// The fixed colours, packed as 0xRRGGBB. Status colours are supplementary: every
// status is also told apart by its glyph's shape and its word (datum 0.16), and a
// status colour is replaced by the foreground colour wherever it would be too
// faint to see.
const (
	blackPacked                = 0x000000
	whitePacked                = 0xffffff
	navigationBackgroundPacked = 0x24242a
	successPacked              = 0x3fb950
	failurePacked              = 0xf85149
	inProgressPacked           = 0xe3b341
	queuedPacked               = 0x58a6ff
	cancelledPacked            = 0xb1bac4
	unknownPacked              = 0xd2a8ff
	errorPacked                = 0xff7b72
)

// The bit positions of each channel in a packed colour.
const (
	redShift   = 16
	greenShift = 8
)

// rgb unpacks a 0xRRGGBB colour as opaque.
func rgb(packed uint32) color.RGBA {
	return color.RGBA{
		R: uint8(packed >> redShift & maximumChannelValue),
		G: uint8(packed >> greenShift & maximumChannelValue),
		B: uint8(packed & maximumChannelValue),
		A: maximumChannelValue,
	}
}

func defaultBackground() color.RGBA { return rgb(blackPacked) }

// ParseColour reads a colour written #RRGGBB, the form the appearance's
// background field takes.
func ParseColour(written string) (color.RGBA, error) {
	if !hexColourPattern.MatchString(written) {
		return color.RGBA{R: 0, G: 0, B: 0, A: 0}, fmt.Errorf("%w: got %d characters", ErrBadColour, len(written))
	}

	packed, parseError := strconv.ParseUint(written[hexColourDigitsOffset:], 16, 32)
	if parseError != nil {
		return color.RGBA{R: 0, G: 0, B: 0, A: 0}, fmt.Errorf("%w: %w", ErrBadColour, parseError)
	}

	return rgb(uint32(packed)), nil
}

// ForegroundFor returns black or white, whichever contrasts more with the
// background, so a label is legible on any background a person chooses. One of
// the two always reaches 4.5 to 1.
func ForegroundFor(background color.RGBA) color.RGBA {
	if contrastRatio(rgb(whitePacked), background) >= contrastRatio(rgb(blackPacked), background) {
		return rgb(whitePacked)
	}

	return rgb(blackPacked)
}

// glyphColourFor keeps a status colour only where it contrasts enough with the
// background to be seen as a graphic, and otherwise uses the foreground colour.
func glyphColourFor(statusColour color.RGBA, background color.RGBA) color.RGBA {
	if contrastRatio(statusColour, background) >= glyphContrastMinimum {
		return statusColour
	}

	return ForegroundFor(background)
}

func contrastRatio(first color.RGBA, second color.RGBA) float64 {
	lighter := math.Max(relativeLuminance(first), relativeLuminance(second))
	darker := math.Min(relativeLuminance(first), relativeLuminance(second))

	return (lighter + contrastFlare) / (darker + contrastFlare)
}

func relativeLuminance(colour color.RGBA) float64 {
	channel := func(value uint8) float64 {
		normalised := float64(value) / maximumChannelValue
		if normalised <= luminanceLinearLimit {
			return normalised / luminanceLinearScale
		}

		return math.Pow((normalised+luminanceGammaOffset)/luminanceGammaScale, luminanceGammaPower)
	}

	return luminanceRedWeight*channel(colour.R) + luminanceGreenWeight*channel(colour.G) +
		luminanceBlueWeight*channel(colour.B)
}
