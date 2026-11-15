package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// A key held down is drawn pressed: its content a little smaller about its
// middle, as if pushed in, and dimmed, so a press is seen the moment it is
// made, whatever the key does. Sizes are hundredths of the key's content.
const (
	pressedScaleHundredths = 92
	pressedLightHundredths = 60
)

// A hold's bar runs along the bottom of a key held towards a hold, filled in
// the clock's pink from the left as far as the hold has gone, on a dim track.
const (
	holdTrackPacked         = 0x4a4a52
	holdBarSideHundredths   = 10
	holdBarBottomHundredths = 6
	holdBarHeightHundredths = 6
)

// A press count is a row of dots top left, one for each press an action needs,
// those made filled and the rest rings.
const (
	pressDotTopHundredths    = 5
	pressDotLeftHundredths   = 6
	pressDotSizeHundredths   = 8
	pressDotGapHundredths    = 4
	pressDotRingHundredths   = 30
	pressDotMinimumRingWidth = 1.5
)

// PressCount is how far a key counting quick presses towards an action has
// gone: Done of the Of presses it needs. No count is under way while Done is
// nothing.
type PressCount struct {
	Done int
	Of   int
}

// Pressed draws a key as it looks held down, from the picture it has when
// not. holding, from nothing to one, is how far a hold has gone, drawn as a
// bar along its bottom; at nothing there is no bar.
func (renderer *Renderer) Pressed(key image.Image, holding float64) image.Image {
	content := image.Rect(0, 0, renderer.keyPixels, renderer.keyPixels)
	if renderer.frame != nil {
		inset := renderer.frame.inset
		content = image.Rect(inset.Left, inset.Top, inset.Left+inset.ContentPixels, inset.Top+inset.ContentPixels)
	}

	bounds := key.Bounds()
	pressed := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(pressed, pressed.Bounds(), image.NewUniform(defaultBackground()), image.Point{X: 0, Y: 0}, draw.Src)

	shrunk := scaledAbout(pressed.Bounds(), content, pressedScaleHundredths)
	xdraw.ApproxBiLinear.Scale(pressed, shrunk, key, bounds, draw.Src, nil)
	dim(pressed, pressedLightHundredths)

	if holding > 0 {
		drawHoldBar(pressed, content, math.Min(holding, 1))
	}

	return pressed
}

// scaledAbout is the whole key's rectangle scaled about its content's middle,
// so the content shrinks in place.
func scaledAbout(whole image.Rectangle, content image.Rectangle, hundredths int) image.Rectangle {
	centreX := float64(content.Min.X+content.Max.X) / centringDivisor
	centreY := float64(content.Min.Y+content.Max.Y) / centringDivisor
	scale := float64(hundredths) / percent
	at := func(value float64, centre float64) int { return int(math.Round(centre + (value-centre)*scale)) }

	return image.Rect(
		at(float64(whole.Min.X), centreX), at(float64(whole.Min.Y), centreY),
		at(float64(whole.Max.X), centreX), at(float64(whole.Max.Y), centreY),
	)
}

// dim scales every pixel's colour towards black.
func dim(key *image.RGBA, hundredths int) {
	share := float64(hundredths) / percent

	for offset := 0; offset < len(key.Pix); offset += 4 {
		for channel := range 3 {
			key.Pix[offset+channel] = uint8(math.Round(float64(key.Pix[offset+channel]) * share))
		}
	}
}

// drawHoldBar draws a hold's bar along the bottom of a key's content, filled
// from the left as far as the hold has gone.
func drawHoldBar(key *image.RGBA, content image.Rectangle, holding float64) {
	size := float64(content.Dx())
	part := func(hundredths int) float64 { return size * float64(hundredths) / percent }
	left := float64(content.Min.X) + part(holdBarSideHundredths)
	width := size - centringDivisor*part(holdBarSideHundredths)
	height := part(holdBarHeightHundredths)
	top := float64(content.Max.Y) - part(holdBarBottomHundredths) - height
	box := glyphBox{destination: key, left: 0, top: 0, size: float32(key.Bounds().Dx())}

	for _, segment := range []struct {
		width  float64
		colour color.RGBA
	}{{width, rgb(holdTrackPacked)}, {math.Max(width*holding, height), rgb(clockForegroundPacked)}} {
		rasterizer := box.newRasterizer()
		box.windPolygon(rasterizer, true, stadium(left, top, segment.width, height, float64(key.Bounds().Dx())))
		box.fill(rasterizer, segment.colour)
	}
}

// drawPressCount draws a press count's dots top left: those made filled, the
// rest rings. While no count is under way it draws nothing.
func (renderer *Renderer) drawPressCount(key *image.RGBA, count PressCount, foreground color.RGBA) {
	if count.Done <= 0 || count.Of <= 0 {
		return
	}

	keySize := float64(renderer.keyPixels)
	diameter := keySize * pressDotSizeHundredths / percent
	radius := diameter / centringDivisor
	ring := math.Max(diameter*pressDotRingHundredths/percent, pressDotMinimumRingWidth)
	centreY := keySize*pressDotTopHundredths/percent + radius
	step := diameter + keySize*pressDotGapHundredths/percent
	box := glyphBox{destination: key, left: 0, top: 0, size: float32(keySize)}

	for dot := range count.Of {
		centreX := keySize*pressDotLeftHundredths/percent + radius + float64(dot)*step
		rasterizer := box.newRasterizer()
		box.circle(rasterizer, centreX/keySize, centreY/keySize, radius/keySize, true)

		if dot >= count.Done {
			box.circle(rasterizer, centreX/keySize, centreY/keySize, (radius-ring)/keySize, false)
		}

		box.fill(rasterizer, foreground)
	}
}
