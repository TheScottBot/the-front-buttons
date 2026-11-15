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
	ink, _ := font.BoundString(face, text)
	baseline := (fixed.I(renderer.keyPixels) - ink.Min.Y - ink.Max.Y) / centringDivisor

	drawCentredText(key, face, text, colour, baseline.Round())

	return key
}
