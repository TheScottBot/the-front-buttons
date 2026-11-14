// Package deckmirror is what the editor may see of the deck as it runs: the
// pictures of the keys whose look depends on live state, as the daemon draws
// them, and word of which tile was pressed, let go of or redrawn. It holds pixels the
// daemon drew and tile positions, never the state behind them, such as a
// track's name, a run's result or a badge's count, so the editor can show only
// what anyone looking at the deck could see.
//
// The daemon writes from its loop and must never wait, so a subscriber that
// does not keep up loses events rather than holding the loop.
package deckmirror

import (
	"bytes"
	"image"
	"maps"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// eventBuffer is how many events wait for a subscriber before more are lost.
const eventBuffer = 64

// EventKind is what happened on the deck.
type EventKind string

// The things a subscriber hears of.
const (
	// Pressed is a key pressed on the deck.
	Pressed EventKind = "press"
	// Released is a pressed key let go of.
	Released EventKind = "release"
	// Redrawn is a key's picture changed.
	Redrawn EventKind = "frame"
	// Reloaded is every picture replaced, for a new configuration.
	Reloaded EventKind = "reload"
)

// Tile is a key's place: its page's index in the configuration and its
// position on the page.
type Tile struct {
	Page     int
	Position layout.Position
}

// Frame is a key's picture and the kind of key drawn.
type Frame struct {
	Kind    string
	Picture image.Image
}

// Event is word of one thing on the deck, by tile alone; a reload names no
// tile.
type Event struct {
	Kind EventKind
	Tile Tile
}

// Mirror keeps the pictures and tells subscribers. Its zero value is not
// usable; New makes one. A nil Mirror does nothing.
type Mirror struct {
	mutex       sync.Mutex
	frames      map[Tile]Frame
	subscribers map[chan Event]struct{}
	maximum     int
}

// New makes a mirror that allows up to maximum subscribers at once.
func New(maximum int) *Mirror {
	return &Mirror{
		mutex: sync.Mutex{}, frames: map[Tile]Frame{}, subscribers: map[chan Event]struct{}{}, maximum: maximum,
	}
}

// Replace sets every picture at once, for a new configuration, and tells
// subscribers to fetch them all again.
func (mirror *Mirror) Replace(frames map[Tile]Frame) {
	if mirror == nil {
		return
	}

	mirror.mutex.Lock()
	defer mirror.mutex.Unlock()

	mirror.frames = make(map[Tile]Frame, len(frames))
	maps.Copy(mirror.frames, frames)

	mirror.publish(Event{Kind: Reloaded, Tile: Tile{Page: 0, Position: layout.Position{Row: 0, Column: 0}}})
}

// Set keeps a key's picture and, if it changed, tells subscribers.
func (mirror *Mirror) Set(tile Tile, kind string, picture image.Image) {
	if mirror == nil {
		return
	}

	mirror.mutex.Lock()
	defer mirror.mutex.Unlock()

	previous, had := mirror.frames[tile]
	mirror.frames[tile] = Frame{Kind: kind, Picture: picture}

	if !had || previous.Kind != kind || !samePicture(previous.Picture, picture) {
		mirror.publish(Event{Kind: Redrawn, Tile: tile})
	}
}

// Press tells subscribers a key was pressed.
func (mirror *Mirror) Press(tile Tile) {
	if mirror == nil {
		return
	}

	mirror.mutex.Lock()
	defer mirror.mutex.Unlock()

	mirror.publish(Event{Kind: Pressed, Tile: tile})
}

// Release tells subscribers a pressed key was let go of.
func (mirror *Mirror) Release(tile Tile) {
	if mirror == nil {
		return
	}

	mirror.mutex.Lock()
	defer mirror.mutex.Unlock()

	mirror.publish(Event{Kind: Released, Tile: tile})
}

// Frame is the picture kept for a tile, if one is kept for that kind of key.
func (mirror *Mirror) Frame(tile Tile, kind string) (image.Image, bool) {
	if mirror == nil {
		return nil, false
	}

	mirror.mutex.Lock()
	defer mirror.mutex.Unlock()

	frame, found := mirror.frames[tile]
	if !found || frame.Kind != kind {
		return nil, false
	}

	return frame.Picture, true
}

// Subscribe starts hearing of events, until cancel is called. It refuses, with
// false, once the most subscribers allowed are listening.
func (mirror *Mirror) Subscribe() (<-chan Event, func(), bool) {
	if mirror == nil {
		return nil, func() {}, false
	}

	mirror.mutex.Lock()
	defer mirror.mutex.Unlock()

	if len(mirror.subscribers) >= mirror.maximum {
		return nil, func() {}, false
	}

	events := make(chan Event, eventBuffer)
	mirror.subscribers[events] = struct{}{}

	var once sync.Once

	cancel := func() {
		once.Do(func() {
			mirror.mutex.Lock()
			defer mirror.mutex.Unlock()

			delete(mirror.subscribers, events)
		})
	}

	return events, cancel, true
}

// publish hands an event to every subscriber with room for it; the caller
// holds the lock.
func (mirror *Mirror) publish(event Event) {
	for subscriber := range mirror.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

// samePicture reports two pictures with the same pixels, where both are the
// renderer's own kind; any other is taken as changed.
func samePicture(first image.Image, second image.Image) bool {
	firstPixels, firstIsRGBA := first.(*image.RGBA)
	secondPixels, secondIsRGBA := second.(*image.RGBA)

	return firstIsRGBA && secondIsRGBA && firstPixels.Rect == secondPixels.Rect &&
		bytes.Equal(firstPixels.Pix, secondPixels.Pix)
}
