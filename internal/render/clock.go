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
// typeface, JetBrains Mono Regular. A time is drawn a figure to a key, large,
// so six keys in a row read HHMMSS; an icon takes the time's pink.
const (
	clockBackgroundPacked = 0x12131c
	clockForegroundPacked = 0xf894f8
	clockLabelPacked      = 0xd583d7
	// clockIconHundredths is a clock icon's size, as a share of the key.
	clockIconHundredths = 60
	// clockSecondWidthHundredths is the share of the key a pair's second
	// figure is centred in when a colon follows it in the rest.
	clockSecondWidthHundredths = 76
)

// RenderClockDigit draws one figure of a time.
func (renderer *Renderer) RenderClockDigit(figure string) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderClockDigit(figure))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	return renderer.clockText(figure, renderer.faces.clockDigit, rgb(clockForegroundPacked))
}

// RenderClockSecondDigit draws the second figure of a pair of a time, a
// little smaller than the first so HH MM SS read as pairs, and with colon a
// small colon beside it, at the key's right, as between hours and minutes.
func (renderer *Renderer) RenderClockSecondDigit(figure string, colon bool) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderClockSecondDigit(figure, colon))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	if !colon {
		return renderer.clockText(figure, renderer.faces.clockSecond, rgb(clockForegroundPacked))
	}

	key := renderer.newKey(rgb(clockBackgroundPacked))
	foreground := rgb(clockForegroundPacked)

	// The figure is centred in the key's left part, the colon in its right.
	figureWidth := renderer.scaled(clockSecondWidthHundredths)
	drawClockText(key, renderer.faces.clockSecond, figure, foreground, 0, figureWidth, renderer.keyPixels)
	drawClockText(key, renderer.faces.clockColon, ":", foreground, figureWidth, renderer.keyPixels, renderer.keyPixels)

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
