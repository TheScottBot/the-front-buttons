package render

import (
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

func (renderer *Renderer) scaled(hundredths int) int {
	return renderer.keyPixels * hundredths / percent
}

func (renderer *Renderer) newKey(background color.RGBA) *image.RGBA {
	key := image.NewRGBA(image.Rect(0, 0, renderer.keyPixels, renderer.keyPixels))
	draw.Draw(key, key.Bounds(), image.NewUniform(background), image.Point{X: 0, Y: 0}, draw.Src)

	return key
}

func (renderer *Renderer) glyphBox(key *image.RGBA, sizeHundredths int) glyphBox {
	return renderer.glyphBoxAt(key, sizeHundredths, glyphTopHundredths)
}

func (renderer *Renderer) glyphBoxAt(key *image.RGBA, sizeHundredths int, topHundredths int) glyphBox {
	size := renderer.scaled(sizeHundredths)

	return glyphBox{
		destination: key,
		left:        float32(renderer.keyPixels-size) / centringDivisor,
		top:         float32(renderer.scaled(topHundredths)),
		size:        float32(size),
	}
}

func (renderer *Renderer) labelWidth() int {
	return renderer.keyPixels - centringDivisor*renderer.scaled(marginHundredths)
}

func (renderer *Renderer) fitLabel(label string) (string, int) {
	fitted, _, width := fitText(label, renderer.faces.labelSizes, renderer.labelWidth())

	return fitted, width
}

func (renderer *Renderer) drawLabel(key *image.RGBA, label string, foreground color.RGBA, baselineY int) {
	if label == "" {
		return
	}

	fitted, face, _ := fitText(label, renderer.faces.labelSizes, renderer.labelWidth())
	drawCentredText(key, face, fitted, foreground, baselineY)
}

func (renderer *Renderer) drawStatusGlyph(box glyphBox, status Status, background color.RGBA) {
	switch status {
	case StatusSuccess:
		drawSuccessGlyph(box, glyphColourFor(rgb(successPacked), background))
	case StatusFailure:
		drawFailureGlyph(box, glyphColourFor(rgb(failurePacked), background))
	case StatusInProgress:
		drawInProgressGlyph(box, glyphColourFor(rgb(inProgressPacked), background))
	case StatusQueued:
		drawQueuedGlyph(box, glyphColourFor(rgb(queuedPacked), background))
	case StatusCancelled:
		drawCancelledGlyph(box, glyphColourFor(rgb(cancelledPacked), background))
	case StatusUnknown, StatusNone:
		// A question mark reads as unknown in any language a person here speaks.
		questionColour := glyphColourFor(rgb(unknownPacked), background)
		drawCentredText(box.destination, renderer.faces.question, "?", questionColour,
			renderer.scaled(questionBaselineFraction))
	}
}

// drawIcon draws the bundled or user icon, or the missing icon marker when a
// named icon does not exist, and reports whether the key has an icon area at
// all.
func (renderer *Renderer) drawIcon(key *image.RGBA, box glyphBox, appearance Appearance, foreground color.RGBA) bool {
	// The icon is drawn on whole pixels, so a square whose spare width is odd
	// cannot be centred and fell half a pixel left; it is grown by a pixel
	// instead (evaluation, deck testing).
	size := int(box.size)
	if (renderer.keyPixels-size)%centringDivisor != 0 {
		size++
	}

	left, top := (renderer.keyPixels-size)/centringDivisor, int(box.top)
	iconRectangle := image.Rect(left, top, left+size, top+size)

	switch {
	case appearance.IconImage != nil:
		xdraw.CatmullRom.Scale(key, fitWithin(iconRectangle, appearance.IconImage.Bounds()),
			appearance.IconImage, appearance.IconImage.Bounds(), xdraw.Over, nil)

		return true
	case appearance.IconMissing:
		drawMissingIconGlyph(box, foreground)

		return true
	case appearance.Icon != "":
		mask, iconFound := renderer.icons.IconMask(appearance.Icon, iconRectangle.Dx())
		if !iconFound {
			drawMissingIconGlyph(box, foreground)

			return true
		}

		origin := image.Point{X: 0, Y: 0}
		draw.DrawMask(key, iconRectangle, image.NewUniform(foreground), origin, mask, origin, draw.Over)

		return true
	default:
		return false
	}
}

// fitWithin returns the largest rectangle of the image's proportions centred in
// the area, so a user's icon is never stretched.
func fitWithin(area image.Rectangle, imageBounds image.Rectangle) image.Rectangle {
	width, height := area.Dx(), area.Dy()
	if imageBounds.Dx()*height > imageBounds.Dy()*width {
		height = width * imageBounds.Dy() / imageBounds.Dx()
	} else {
		width = height * imageBounds.Dx() / imageBounds.Dy()
	}

	left := area.Min.X + (area.Dx()-width)/centringDivisor
	top := area.Min.Y + (area.Dy()-height)/centringDivisor

	return image.Rect(left, top, left+width, top+height)
}

// fillWithPicture covers the key with a picture, its middle square scaled to
// fill it edge to edge, as album art is shown.
func fillWithPicture(key *image.RGBA, picture image.Image) {
	bounds := picture.Bounds()
	side := min(bounds.Dx(), bounds.Dy())
	left := bounds.Min.X + (bounds.Dx()-side)/centringDivisor
	top := bounds.Min.Y + (bounds.Dy()-side)/centringDivisor
	square := image.Rect(left, top, left+side, top+side)

	xdraw.CatmullRom.Scale(key, key.Bounds(), picture, square, xdraw.Src, nil)
}
