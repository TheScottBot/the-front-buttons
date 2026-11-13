package render

import (
	"image"
	"image/color"
	"math"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// A badge is drawn as the Dock draws one: a red pill, top right, holding its
// label in white, or a red dot for a label with no letter or digit, such as
// Discord's pip. Sizes are hundredths of the key's width.
const (
	badgePacked                 = 0xff3b30
	badgeMarginHundredths       = 5
	badgeHeightHundredths       = 30
	badgePipHundredths          = 20
	badgeMaximumWidthHundredths = 60
	badgePaddingHundredths      = 8
)

// drawBadge draws a badge's pill or dot in the key's top right corner.
func (renderer *Renderer) drawBadge(key *image.RGBA, badge string) {
	if badge == "" {
		return
	}

	margin := float64(renderer.scaled(badgeMarginHundredths))
	right := float64(renderer.keyPixels) - margin
	box := glyphBox{destination: key, left: 0, top: 0, size: float32(renderer.keyPixels)}
	rasterizer := box.newRasterizer()

	if !hasLetterOrDigit(badge) {
		radius := float64(renderer.scaled(badgePipHundredths)) / centringDivisor
		box.circle(rasterizer, (right-radius)/float64(renderer.keyPixels), (margin+radius)/float64(renderer.keyPixels),
			radius/float64(renderer.keyPixels), true)
		box.fill(rasterizer, rgb(badgePacked))

		return
	}

	height := float64(renderer.scaled(badgeHeightHundredths))
	padding := float64(renderer.scaled(badgePaddingHundredths))
	fitted, face, textWidth := fitText(badge, renderer.faces.wordSizes,
		renderer.scaled(badgeMaximumWidthHundredths)-int(centringDivisor*padding))
	width := math.Max(height, float64(textWidth)+centringDivisor*padding)

	box.windPolygon(rasterizer, true, stadium(right-width, margin, width, height, float64(renderer.keyPixels)))
	box.fill(rasterizer, rgb(badgePacked))

	drawTextCentredAt(key, face, fitted, rgb(whitePacked), right-width/centringDivisor, margin+height/centringDivisor)
}

// hasLetterOrDigit reports a label with something to write, rather than a
// pip.
func hasLetterOrDigit(label string) bool {
	for _, character := range label {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			return true
		}
	}

	return false
}

// stadium is a pill's outline, a rectangle with round ends, as unit points
// across a key of the given size.
func stadium(left float64, top float64, width float64, height float64, keySize float64) []float64 {
	radius := height / centringDivisor
	centreY := top + radius
	points := make([]float64, 0, circleSegments*centringDivisor+centringDivisor)

	ends := []struct{ centreX, from float64 }{
		{left + width - radius, -math.Pi / centringDivisor},
		{left + radius, math.Pi / centringDivisor},
	}

	for _, end := range ends {
		for segment := range circleSegments/centringDivisor + 1 {
			angle := end.from + math.Pi*float64(segment)/(circleSegments/centringDivisor)
			points = append(points,
				(end.centreX+radius*math.Cos(angle))/keySize, (centreY+radius*math.Sin(angle))/keySize)
		}
	}

	return points
}

// fixedPointUnit is one pixel in the font package's 26.6 fixed point.
const fixedPointUnit = 1 << 6

// drawTextCentredAt draws text with its ink centred on a point.
func drawTextCentredAt(destination *image.RGBA, face font.Face, text string, colour color.RGBA, centreX float64,
	centreY float64,
) {
	ink, _ := font.BoundString(face, text)
	drawer := font.Drawer{
		Dst:  destination,
		Src:  image.NewUniform(colour),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.Int26_6(centreX*fixedPointUnit) - (ink.Min.X+ink.Max.X)/centringDivisor,
			Y: fixed.Int26_6(centreY*fixedPointUnit) - (ink.Min.Y+ink.Max.Y)/centringDivisor,
		},
	}
	drawer.DrawString(text)
}
