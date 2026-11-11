package render

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/vector"
)

// circleSegments is how many straight edges approximate a circle; at key sizes
// more are invisible.
const circleSegments = 48

// glyphBox is the square a status glyph is drawn in, with shapes given in unit
// coordinates from 0 to 1 across it.
type glyphBox struct {
	destination *image.RGBA
	left        float32
	top         float32
	size        float32
}

func (box glyphBox) point(unitX float64, unitY float64) (float32, float32) {
	return box.left + float32(unitX)*box.size, box.top + float32(unitY)*box.size
}

func (box glyphBox) newRasterizer() *vector.Rasterizer {
	bounds := box.destination.Bounds()

	return vector.NewRasterizer(bounds.Dx(), bounds.Dy())
}

func (box glyphBox) fill(rasterizer *vector.Rasterizer, colour color.RGBA) {
	rasterizer.Draw(box.destination, box.destination.Bounds(), image.NewUniform(colour), image.Point{X: 0, Y: 0})
}

// polygon adds a solid closed polygon through unit points given as x, y pairs.
func (box glyphBox) polygon(rasterizer *vector.Rasterizer, unitPoints ...float64) {
	box.windPolygon(rasterizer, true, unitPoints)
}

// windPolygon adds a polygon wound one way for a solid shape and the other way
// for a hole. Overlapping solid shapes must share a winding, or the rasteriser
// subtracts one from the other and leaves a seam where they overlap, which is
// what the first TE4 contact sheet showed where a stroke met its round end.
func (box glyphBox) windPolygon(rasterizer *vector.Rasterizer, solid bool, unitPoints []float64) {
	if (signedArea(unitPoints) > 0) != solid {
		unitPoints = reversedPoints(unitPoints)
	}

	startX, startY := box.point(unitPoints[0], unitPoints[1])
	rasterizer.MoveTo(startX, startY)

	for pointIndex := 2; pointIndex+1 < len(unitPoints); pointIndex += 2 {
		pointX, pointY := box.point(unitPoints[pointIndex], unitPoints[pointIndex+1])
		rasterizer.LineTo(pointX, pointY)
	}

	rasterizer.ClosePath()
}

// circle adds a circle; clockwise true makes it solid, false makes it a hole
// when it lies inside another shape.
func (box glyphBox) circle(rasterizer *vector.Rasterizer, centreX, centreY, radius float64, clockwise bool) {
	unitPoints := make([]float64, 0, circleSegments*2)

	for segment := range circleSegments {
		angle := 2 * math.Pi * float64(segment) / circleSegments
		if !clockwise {
			angle = -angle
		}

		unitPoints = append(unitPoints, centreX+radius*math.Cos(angle), centreY+radius*math.Sin(angle))
	}

	box.windPolygon(rasterizer, clockwise, unitPoints)
}

// signedArea is the shoelace area of a polygon given as x, y pairs; its sign is
// the polygon's winding.
func signedArea(unitPoints []float64) float64 {
	area := 0.0
	pointCount := len(unitPoints) / 2

	for pointIndex := range pointCount {
		nextIndex := (pointIndex + 1) % pointCount
		area += unitPoints[2*pointIndex]*unitPoints[2*nextIndex+1] - unitPoints[2*nextIndex]*unitPoints[2*pointIndex+1]
	}

	return area / 2
}

func reversedPoints(unitPoints []float64) []float64 {
	reversed := make([]float64, 0, len(unitPoints))
	for pointIndex := len(unitPoints) - 2; pointIndex >= 0; pointIndex -= 2 {
		reversed = append(reversed, unitPoints[pointIndex], unitPoints[pointIndex+1])
	}

	return reversed
}

// stroke adds a thick line between two unit points with round ends.
func (box glyphBox) stroke(rasterizer *vector.Rasterizer, fromX, fromY, toX, toY, thickness float64) {
	length := math.Hypot(toX-fromX, toY-fromY)
	normalX := -(toY - fromY) / length * thickness / 2
	normalY := (toX - fromX) / length * thickness / 2

	box.polygon(rasterizer,
		fromX+normalX, fromY+normalY, toX+normalX, toY+normalY,
		toX-normalX, toY-normalY, fromX-normalX, fromY-normalY)
	box.circle(rasterizer, fromX, fromY, thickness/2, true)
	box.circle(rasterizer, toX, toY, thickness/2, true)
}

// The status glyphs. Each is a different shape, so statuses are told apart with
// no colour at all (datum 0.16), and a different word is drawn beneath.

const glyphStrokeThickness = 0.16

func drawSuccessGlyph(box glyphBox, colour color.RGBA) {
	rasterizer := box.newRasterizer()
	box.stroke(rasterizer, 0.14, 0.52, 0.40, 0.78, glyphStrokeThickness)
	box.stroke(rasterizer, 0.40, 0.78, 0.88, 0.22, glyphStrokeThickness)
	box.fill(rasterizer, colour)
}

func drawFailureGlyph(box glyphBox, colour color.RGBA) {
	rasterizer := box.newRasterizer()
	box.stroke(rasterizer, 0.18, 0.18, 0.82, 0.82, glyphStrokeThickness)
	box.stroke(rasterizer, 0.82, 0.18, 0.18, 0.82, glyphStrokeThickness)
	box.fill(rasterizer, colour)
}

func drawInProgressGlyph(box glyphBox, colour color.RGBA) {
	rasterizer := box.newRasterizer()
	for _, centreX := range []float64{0.17, 0.50, 0.83} {
		box.circle(rasterizer, centreX, 0.5, 0.13, true)
	}

	box.fill(rasterizer, colour)
}

func drawQueuedGlyph(box glyphBox, colour color.RGBA) {
	rasterizer := box.newRasterizer()
	box.polygon(rasterizer, 0.18, 0.10, 0.82, 0.10, 0.50, 0.50)
	box.polygon(rasterizer, 0.50, 0.50, 0.82, 0.90, 0.18, 0.90)
	box.fill(rasterizer, colour)
}

func drawCancelledGlyph(box glyphBox, colour color.RGBA) {
	rasterizer := box.newRasterizer()
	box.circle(rasterizer, 0.5, 0.5, 0.44, true)
	box.circle(rasterizer, 0.5, 0.5, 0.30, false)
	box.stroke(rasterizer, 0.22, 0.22, 0.78, 0.78, glyphStrokeThickness*0.85)
	box.fill(rasterizer, colour)
}

func drawErrorGlyph(box glyphBox, colour color.RGBA, background color.RGBA) {
	rasterizer := box.newRasterizer()
	box.polygon(rasterizer, 0.50, 0.06, 0.96, 0.92, 0.04, 0.92)
	box.fill(rasterizer, colour)

	mark := box.newRasterizer()
	box.polygon(mark, 0.45, 0.32, 0.55, 0.32, 0.53, 0.64, 0.47, 0.64)
	box.circle(mark, 0.5, 0.78, 0.06, true)
	box.fill(mark, background)
}

func drawMissingIconGlyph(box glyphBox, colour color.RGBA) {
	rasterizer := box.newRasterizer()
	box.polygon(rasterizer, 0.10, 0.10, 0.90, 0.10, 0.90, 0.90, 0.10, 0.90)
	box.windPolygon(rasterizer, false, []float64{0.18, 0.18, 0.18, 0.82, 0.82, 0.82, 0.82, 0.18})
	box.stroke(rasterizer, 0.18, 0.82, 0.82, 0.18, glyphStrokeThickness*0.5)
	box.fill(rasterizer, colour)
}

func drawChevronGlyph(box glyphBox, colour color.RGBA, pointsLeft bool) {
	rasterizer := box.newRasterizer()
	tipX, tailX := 0.28, 0.72

	if !pointsLeft {
		tipX, tailX = tailX, tipX
	}

	box.stroke(rasterizer, tailX, 0.12, tipX, 0.5, glyphStrokeThickness)
	box.stroke(rasterizer, tipX, 0.5, tailX, 0.88, glyphStrokeThickness)
	box.fill(rasterizer, colour)
}
