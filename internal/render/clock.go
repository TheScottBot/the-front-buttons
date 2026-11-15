package render

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Clock keys, in the look of Thyme, the author's clock, taken from its theme
// (internal/theme/theme.go there): its panel's navy, its time's pink, and the
// dimmer pink of its zone labels, so the eye reads the times first; and its
// typeface, JetBrains Mono Regular. A time is drawn a figure to a key, each
// pair drawn towards each other with a small colon after the hours and
// minutes, so six keys in a row read as 15:31:42; an icon takes the time's
// pink.
const (
	clockBackgroundPacked = 0x12131c
	clockForegroundPacked = 0xf894f8
	clockLabelPacked      = 0xd583d7
	// clockDatePacked is Thyme's date, a clear step dimmer than the time.
	clockDatePacked = 0xe48be6
	// clockDateInsetHundredths is how far a date part sits in from its key's
	// top and right edges.
	clockDateInsetHundredths = 10
	// clockIconHundredths is a clock icon's size, as a share of the key.
	clockIconHundredths = 60
	// clockFigureWidthHundredths is the share of the key a figure is
	// centred in, at its right for a pair's first and its left for the
	// second, so a pair sits together; the rest of the second's key holds
	// the colon.
	clockFigureWidthHundredths = 76
)

// RenderClockDigit draws the first figure of a pair of a time, centred
// towards the key's right, so it sits close to the pair's second.
func (renderer *Renderer) RenderClockDigit(figure string) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderClockDigit(figure))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	key := renderer.newKey(rgb(clockBackgroundPacked))
	drawClockText(key, renderer.faces.clockFigure, figure, rgb(clockForegroundPacked),
		renderer.keyPixels-renderer.scaled(clockFigureWidthHundredths), renderer.keyPixels, renderer.keyPixels)

	return key
}

// RenderClockSecondDigit draws the second figure of a pair of a time,
// centred towards the key's left, so it sits close to the pair's first, and
// with colon a small colon in the room left at the key's right, as after the
// hours and the minutes.
func (renderer *Renderer) RenderClockSecondDigit(figure string, colon bool) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderClockSecondDigit(figure, colon))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	key := renderer.newKey(rgb(clockBackgroundPacked))
	foreground := rgb(clockForegroundPacked)
	figureWidth := renderer.scaled(clockFigureWidthHundredths)

	drawClockText(key, renderer.faces.clockFigure, figure, foreground, 0, figureWidth, renderer.keyPixels)

	if colon {
		drawClockText(key, renderer.faces.clockColon, ":", foreground, figureWidth, renderer.keyPixels, renderer.keyPixels)
	}

	return key
}

// RenderClockDate draws a part of a date, such as its year's two figures,
// small and in the key's top right corner, in Thyme's date colour, so it sits
// apart from the times beside it.
func (renderer *Renderer) RenderClockDate(part string) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderClockDate(part))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	key := renderer.newKey(rgb(clockBackgroundPacked))
	face := renderer.faces.clockLabel
	inset := renderer.scaled(clockDateInsetHundredths)
	ink, _ := font.BoundString(face, part)

	drawer := font.Drawer{
		Dst: key, Src: image.NewUniform(rgb(clockDatePacked)), Face: face,
		Dot: fixed.Point26_6{X: fixed.I(renderer.keyPixels-inset) - ink.Max.X, Y: fixed.I(inset) - ink.Min.Y},
	}
	drawer.DrawString(part)

	return key
}

// RenderClockLabel draws a clock's short label, such as a time zone's.
func (renderer *Renderer) RenderClockLabel(label string) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderClockLabel(label))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	return renderer.clockText(label, renderer.faces.clockLabel, rgb(clockLabelPacked))
}

// RenderClockIcon draws a bundled icon in the clock's colours, or the plain
// key if there is no such icon.
func (renderer *Renderer) RenderClockIcon(iconName string) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderClockIcon(iconName))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	key := renderer.newKey(rgb(clockBackgroundPacked))
	size := renderer.scaled(clockIconHundredths)

	mask, found := renderer.icons.IconMask(iconName, size)
	if !found {
		return key
	}

	offset := (renderer.keyPixels - size) / centringDivisor
	draw.DrawMask(key, image.Rect(offset, offset, offset+size, offset+size),
		image.NewUniform(rgb(clockForegroundPacked)), image.Point{X: 0, Y: 0}, mask, image.Point{X: 0, Y: 0}, draw.Over)

	return key
}

// clockText draws text in the middle of a clock key, both ways, by its ink.
func (renderer *Renderer) clockText(text string, face font.Face, colour color.RGBA) *image.RGBA {
	key := renderer.newKey(rgb(clockBackgroundPacked))
	drawClockText(key, face, text, colour, 0, renderer.keyPixels, renderer.keyPixels)

	return key
}

// drawClockText draws text centred, by its ink, across the columns from left
// to right and down the key's height.
func drawClockText(key *image.RGBA, face font.Face, text string, colour color.RGBA, left int, right int, height int) {
	ink, _ := font.BoundString(face, text)
	baseline := (fixed.I(height) - ink.Min.Y - ink.Max.Y) / centringDivisor
	start := (fixed.I(left+right) - ink.Min.X - ink.Max.X) / centringDivisor

	drawer := font.Drawer{Dst: key, Src: image.NewUniform(colour), Face: face, Dot: fixed.Point26_6{X: start, Y: baseline}}
	drawer.DrawString(text)
}
