package render

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"
)

// percent turns the hundredths the layout is written in into fractions.
const percent = 100

// halfPixel moves a pixel's coordinate to its centre.
const halfPixel = 0.5

// centringDivisor halves the spare space either side of something centred.
const centringDivisor = 2

// The plausible key sizes. Every Stream Deck model's keys are within this range;
// a size outside it means the model table is wrong, not that a key is unusual.
const (
	minimumKeyPixels = 48
	maximumKeyPixels = 512
)

// ErrImplausibleKeySize reports a key size outside the plausible range.
var ErrImplausibleKeySize = errors.New("implausible key size")

// ErrFrameDoesNotFit reports a frame whose content square would not fit on the
// key, or would be too small to draw in.
var ErrFrameDoesNotFit = errors.New("key frame does not fit")

// The key's layout, in hundredths of its width, shared by every kind of key so
// they line up as a set. Provisional until the TE4 legibility gate.
const (
	marginHundredths         = 6
	glyphTopHundredths       = 6
	glyphSizeHundredths      = 46
	labelBaselineHundredths  = 75
	wordBaselineHundredths   = 92
	soloLabelBaseline        = 58
	pageNumberBaseline       = 70
	navigationNameBaseline   = 92
	navigationChevronSize    = 30
	questionBaselineFraction = 47
	// A key with an icon and no status word centres the icon and label as one
	// group, and an icon alone is centred, rather than leaving room for a word
	// line they never show (evaluation, deck testing).
	iconGroupTopHundredths = 15
	iconGroupBaseline      = 84
	iconAloneTopHundredths = (percent - glyphSizeHundredths) / centringDivisor
)

// Status is a live brick's state. Each has its own glyph shape and word.
type Status string

// The statuses a live brick reports (specification section 2).
const (
	StatusNone       Status = ""
	StatusSuccess    Status = "success"
	StatusFailure    Status = "failure"
	StatusInProgress Status = "in_progress"
	StatusQueued     Status = "queued"
	StatusCancelled  Status = "cancelled"
	StatusUnknown    Status = "unknown"
)

// StatusWord is the word drawn under a status glyph. Every status has a
// different word, so the state can be read as well as seen.
func StatusWord(status Status) string {
	switch status {
	case StatusSuccess:
		return "PASSED"
	case StatusFailure:
		return "FAILED"
	case StatusInProgress:
		return "RUNNING"
	case StatusQueued:
		return "QUEUED"
	case StatusCancelled:
		return "CANCELLED"
	case StatusUnknown:
		return "UNKNOWN"
	case StatusNone:
		return ""
	default:
		return ""
	}
}

// Appearance is everything a key shows. ErrorReason, when set, puts the key in
// its error state, which outranks a status (specification section 2: a brick in
// error shows that, never failing silently).
type Appearance struct {
	Label     string
	Icon      string
	IconImage image.Image
	// IconMissing marks a key whose own icon file could not be loaded; it is
	// drawn with the missing icon marker, as an unknown bundled icon is.
	IconMissing bool
	// BackgroundFromIcon colours the key to match the edge of IconImage, as an
	// application's own icon asks. An explicit Background still wins, because a
	// person who chose a colour meant it; an icon with no visible edge, or no
	// icon, leaves the default background.
	BackgroundFromIcon bool
	Background         color.RGBA
	Status             Status
	ErrorReason        string
	// Picture fills the whole key, such as a track's album art, in place of
	// the label and icon; an error still shows instead.
	Picture image.Image
}

// IconSet supplies bundled icons by name as masks, so the renderer can draw
// them in whatever colour contrasts with the key's background.
type IconSet interface {
	IconMask(iconName string, sizePixels int) (*image.Alpha, bool)
}

// Direction is which way a navigation key pages.
type Direction string

// The navigation directions.
const (
	DirectionBack    Direction = "back"
	DirectionForward Direction = "forward"
)

// Renderer draws keys. It is the only drawing code (specification section 2):
// the device and the editor preview both show the images it returns.
type Renderer struct {
	keyPixels int
	icons     IconSet
	faces     textFaces
	// Font faces cache glyphs and are not safe for concurrent use, so drawing
	// is serialised.
	mutex sync.Mutex
	// frame, when set, draws every key smaller, by an inner renderer, inside
	// a black frame.
	frame *keyFrame
}

// Inset places a key's content inside a black frame: a square of
// ContentPixels, its top left corner Left and Top pixels into the key. The XL
// draws each key's image a little up and left of its window, and its edges bow,
// so a light key showed a black strip at the bottom and right only; framing
// every key in black, thicker at the top and left that the deck hides, makes
// the black seen around each key even (the author's request, evaluation, deck
// testing).
type Inset struct {
	Left          int
	Top           int
	ContentPixels int
	// Bulge is how far each edge is drawn outward at its middle, falling to
	// nothing at the corners: the deck's domed key caps pull straight edges in
	// at the middle, so a bulge of the same size looks straight through them.
	Bulge float64
	// CornerRadius rounds the content's corners.
	CornerRadius int
}

// keyFrame draws a key's content with an inner renderer and places it in a
// black square.
type keyFrame struct {
	inner *Renderer
	inset Inset
	size  int
}

// edgeSamples is how many points along each side of a pixel are tested
// against the shape, so a pixel the edge crosses is blended with black in
// proportion: a hard one pixel staircase showed through the key cap
// (evaluation, deck testing).
const edgeSamples = 4

// place draws the content into a black key, shaped: its edges bulge outward at
// their middles and its corners are rounded, with the pixels the edge crosses
// blended with black. A pixel the bulge adds beyond the content takes the
// colour of the content's nearest edge pixel.
func (frame *keyFrame) place(content image.Image) image.Image {
	key := image.NewRGBA(image.Rect(0, 0, frame.size, frame.size))
	draw.Draw(key, key.Bounds(), image.NewUniform(defaultBackground()), image.Point{X: 0, Y: 0}, draw.Src)

	inset, origin := frame.inset, content.Bounds().Min

	for row := range frame.size {
		for column := range frame.size {
			coverage := frame.coverage(column, row)
			if coverage == 0 {
				continue
			}

			sourceColumn := min(max(column-inset.Left, 0), inset.ContentPixels-1)
			sourceRow := min(max(row-inset.Top, 0), inset.ContentPixels-1)
			source := color.RGBAModel.Convert(content.At(origin.X+sourceColumn, origin.Y+sourceRow))
			key.Set(column, row, scaleTowardsBlack(source, coverage))
		}
	}

	return key
}

// coverage is the share of a pixel inside the shape, from zero to one.
func (frame *keyFrame) coverage(column int, row int) float64 {
	inside := 0

	for sampleRow := range edgeSamples {
		for sampleColumn := range edgeSamples {
			pointX := float64(column) + (float64(sampleColumn)+halfPixel)/edgeSamples
			pointY := float64(row) + (float64(sampleRow)+halfPixel)/edgeSamples

			if frame.inside(pointX, pointY) {
				inside++
			}
		}
	}

	return float64(inside) / (edgeSamples * edgeSamples)
}

// scaleTowardsBlack blends a colour with the black around it by coverage.
func scaleTowardsBlack(source color.Color, coverage float64) color.RGBA {
	opaque, _ := source.(color.RGBA)
	scale := func(channel uint8) uint8 { return uint8(math.Round(float64(channel) * coverage)) }

	return color.RGBA{R: scale(opaque.R), G: scale(opaque.G), B: scale(opaque.B), A: opaque.A}
}

// inside reports whether a key pixel falls within the shaped content: between
// the bulging edges, and not in a rounded corner.
func (frame *keyFrame) inside(pointX float64, pointY float64) bool {
	inset := frame.inset
	size := float64(inset.ContentPixels)
	centreX := float64(inset.Left) + size/centringDivisor
	centreY := float64(inset.Top) + size/centringDivisor

	// The bulge at a point along an edge: full at the middle, nothing at the
	// ends, following a parabola.
	bulgeAt := func(along float64, centre float64) float64 {
		offset := math.Min(math.Abs(along-centre)/(size/centringDivisor), 1)

		return inset.Bulge * (1 - offset*offset)
	}

	left := float64(inset.Left) - bulgeAt(pointY, centreY)
	right := float64(inset.Left) + size + bulgeAt(pointY, centreY)
	top := float64(inset.Top) - bulgeAt(pointX, centreX)
	bottom := float64(inset.Top) + size + bulgeAt(pointX, centreX)

	if pointX < left || pointX >= right || pointY < top || pointY >= bottom {
		return false
	}

	return !frame.inRoundedCorner(pointX, pointY)
}

// inRoundedCorner reports whether a point lies in a corner cut away by the
// corner radius.
func (frame *keyFrame) inRoundedCorner(pointX float64, pointY float64) bool {
	inset := frame.inset
	radius := float64(inset.CornerRadius)
	low, high := float64(inset.Left)+radius, float64(inset.Left+inset.ContentPixels)-radius
	lowY, highY := float64(inset.Top)+radius, float64(inset.Top+inset.ContentPixels)-radius

	cornerX := math.Max(low-pointX, pointX-high)
	cornerY := math.Max(lowY-pointY, pointY-highY)

	return cornerX > 0 && cornerY > 0 && cornerX*cornerX+cornerY*cornerY > radius*radius
}

// NewFramedRenderer returns a renderer whose keys are keyPixels square, with
// their content drawn at the inset's size and placed in a black frame.
func NewFramedRenderer(keyPixels int, icons IconSet, inset Inset) (*Renderer, error) {
	bulge := int(math.Ceil(inset.Bulge))
	if inset.Left < bulge || inset.Top < bulge || inset.Bulge < 0 ||
		inset.ContentPixels < minimumKeyPixels || inset.CornerRadius < 0 ||
		inset.CornerRadius > inset.ContentPixels/centringDivisor ||
		inset.Left+inset.ContentPixels+bulge > keyPixels ||
		inset.Top+inset.ContentPixels+bulge > keyPixels {
		return nil, fmt.Errorf("%w: %d pixels at %d, %d on a %d pixel key",
			ErrFrameDoesNotFit, inset.ContentPixels, inset.Left, inset.Top, keyPixels)
	}

	inner, innerError := NewRenderer(inset.ContentPixels, icons)
	if innerError != nil {
		return nil, innerError
	}

	return &Renderer{
		keyPixels: keyPixels, icons: icons, faces: inner.faces, mutex: sync.Mutex{},
		frame: &keyFrame{inner: inner, inset: inset, size: keyPixels},
	}, nil
}

// NewRenderer returns a renderer for square keys of keyPixels, taken from the
// deck model table.
func NewRenderer(keyPixels int, icons IconSet) (*Renderer, error) {
	if keyPixels < minimumKeyPixels || keyPixels > maximumKeyPixels {
		return nil, fmt.Errorf("%w: %d pixels, expected %d to %d",
			ErrImplausibleKeySize, keyPixels, minimumKeyPixels, maximumKeyPixels)
	}

	faces, facesError := loadFaces(keyPixels)
	if facesError != nil {
		return nil, facesError
	}

	return &Renderer{keyPixels: keyPixels, icons: icons, faces: faces, mutex: sync.Mutex{}, frame: nil}, nil
}

// Render draws a brick's key.
func (renderer *Renderer) Render(appearance Appearance) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.Render(appearance))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	background := backgroundOf(appearance)
	key := renderer.newKey(background)
	foreground := ForegroundFor(background)
	box := renderer.glyphBox(key, glyphSizeHundredths)

	var word string

	switch {
	case appearance.ErrorReason != "":
		drawErrorGlyph(box, glyphColourFor(rgb(errorPacked), background), background)

		word = appearance.ErrorReason
	case appearance.Status != StatusNone:
		renderer.drawStatusGlyph(box, appearance.Status, background)

		word = StatusWord(appearance.Status)
	case appearance.Picture != nil:
		fillWithPicture(key, appearance.Picture)

		return key
	default:
		return renderer.renderPlain(key, appearance, foreground)
	}

	renderer.drawLabel(key, appearance.Label, foreground, renderer.scaled(labelBaselineHundredths))

	if word != "" {
		fittedWord, wordFace, _ := fitText(word, renderer.faces.wordSizes, renderer.labelWidth())
		drawCentredText(key, wordFace, fittedWord, foreground, renderer.scaled(wordBaselineHundredths))
	}

	return key
}

// backgroundOf chooses a key's background: an explicit one first, then the
// icon's edge colour when asked for, then the default.
func backgroundOf(appearance Appearance) color.RGBA {
	if appearance.Background.A != 0 {
		return appearance.Background
	}

	if appearance.BackgroundFromIcon && appearance.IconImage != nil {
		edge, found := EdgeColour(appearance.IconImage)
		if found {
			return edge
		}
	}

	return defaultBackground()
}

// RenderBlank draws an empty position: plain background, not an error
// (specification section 2).
func (renderer *Renderer) RenderBlank() image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(renderer.frame.inner.RenderBlank())
	}

	return renderer.newKey(defaultBackground())
}

// renderPlain draws a key with no status and no error: its icon and label
// centred together, an icon alone centred, or a label alone.
func (renderer *Renderer) renderPlain(key *image.RGBA, appearance Appearance, foreground color.RGBA) *image.RGBA {
	top := iconGroupTopHundredths
	if appearance.Label == "" {
		top = iconAloneTopHundredths
	}

	if !renderer.drawIcon(key, renderer.glyphBoxAt(key, glyphSizeHundredths, top), appearance, foreground) {
		renderer.drawLabel(key, appearance.Label, foreground, renderer.scaled(soloLabelBaseline))

		return key
	}

	renderer.drawLabel(key, appearance.Label, foreground, renderer.scaled(iconGroupBaseline))

	return key
}
