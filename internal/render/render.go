package render

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"
)

// percent turns the hundredths the layout is written in into fractions.
const percent = 100

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
}

// keyFrame draws a key's content with an inner renderer and places it in a
// black square.
type keyFrame struct {
	inner *Renderer
	inset Inset
	size  int
}

func (frame *keyFrame) place(content image.Image) image.Image {
	key := image.NewRGBA(image.Rect(0, 0, frame.size, frame.size))
	draw.Draw(key, key.Bounds(), image.NewUniform(defaultBackground()), image.Point{X: 0, Y: 0}, draw.Src)

	destination := image.Rect(frame.inset.Left, frame.inset.Top,
		frame.inset.Left+frame.inset.ContentPixels, frame.inset.Top+frame.inset.ContentPixels)
	draw.Draw(key, destination, content, content.Bounds().Min, draw.Src)

	return key
}

// NewFramedRenderer returns a renderer whose keys are keyPixels square, with
// their content drawn at the inset's size and placed in a black frame.
func NewFramedRenderer(keyPixels int, icons IconSet, inset Inset) (*Renderer, error) {
	if inset.Left < 0 || inset.Top < 0 || inset.ContentPixels < minimumKeyPixels ||
		inset.Left+inset.ContentPixels > keyPixels || inset.Top+inset.ContentPixels > keyPixels {
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
