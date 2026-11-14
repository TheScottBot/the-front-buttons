package deckmirror_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deckmirror"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The kinds and tiles of the mirror tests.
const (
	albumKind = "spotify_album"
	otherKind = "open_url"
	maximum   = 2
)

//nolint:gochecknoglobals // fixed tiles
var (
	albumTile = deckmirror.Tile{Page: 1, Position: layout.Position{Row: 0, Column: 2}}
	otherTile = deckmirror.Tile{Page: 0, Position: layout.Position{Row: 3, Column: 4}}
)

func picture(shade uint8) image.Image {
	drawn := image.NewRGBA(image.Rect(0, 0, 4, 4))

	for y := range 4 {
		for x := range 4 {
			drawn.Set(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
		}
	}

	return drawn
}

// next is the next event waiting, or false if there is none.
func next(events <-chan deckmirror.Event) (deckmirror.Event, bool) {
	select {
	case event := <-events:
		return event, true
	default:
		return deckmirror.Event{Kind: "", Tile: deckmirror.Tile{Page: 0, Position: layout.Position{Row: 0, Column: 0}}},
			false
	}
}

// TestMirror_KeepsAKeysPicture_ForItsKindOnly: a picture is kept for its tile
// and the kind of key drawn there, so a different key on the tile, as after a
// move the deck has not yet loaded, is never shown another's picture.
func TestMirror_KeepsAKeysPicture_ForItsKindOnly(t *testing.T) {
	t.Parallel()

	mirror := deckmirror.New(maximum)
	mirror.Set(albumTile, albumKind, picture(10))

	if drawn, found := mirror.Frame(albumTile, albumKind); !found || drawn.At(0, 0) != picture(10).At(0, 0) {
		t.Error("want the album's picture kept")
	}

	if _, found := mirror.Frame(albumTile, otherKind); found {
		t.Error("want no picture for another kind on the tile")
	}

	if _, found := mirror.Frame(otherTile, albumKind); found {
		t.Error("want no picture for a tile never drawn")
	}
}

// TestMirror_TellsSubscribersOfPressesAndChanges: a subscriber hears of a
// press and of a picture that changed, by tile alone, and of a reload; a
// picture drawn again unchanged is not news.
func TestMirror_TellsSubscribersOfPressesAndChanges(t *testing.T) {
	t.Parallel()

	mirror := deckmirror.New(maximum)

	events, cancel, subscribed := mirror.Subscribe()
	if !subscribed {
		t.Fatal("want a subscription")
	}

	defer cancel()

	mirror.Set(albumTile, albumKind, picture(10))
	mirror.Set(albumTile, albumKind, picture(10))
	mirror.Press(otherTile)
	mirror.Replace(map[deckmirror.Tile]deckmirror.Frame{otherTile: {Kind: otherKind, Picture: picture(20)}})

	want := []deckmirror.Event{
		{Kind: deckmirror.Redrawn, Tile: albumTile},
		{Kind: deckmirror.Pressed, Tile: otherTile},
		{Kind: deckmirror.Reloaded, Tile: deckmirror.Tile{Page: 0, Position: layout.Position{Row: 0, Column: 0}}},
	}

	for _, wanted := range want {
		if event, found := next(events); !found || event != wanted {
			t.Errorf("want %+v, got %+v (%v)", wanted, event, found)
		}
	}

	if event, found := next(events); found {
		t.Errorf("want no more events, got %+v", event)
	}

	// A reload replaces every picture.
	if _, found := mirror.Frame(albumTile, albumKind); found {
		t.Error("want the album's picture gone after a reload")
	}

	if _, found := mirror.Frame(otherTile, otherKind); !found {
		t.Error("want the reload's picture kept")
	}
}

// TestMirror_LimitsSubscribers: only so many may subscribe at once, and a
// cancelled subscription frees its place.
func TestMirror_LimitsSubscribers(t *testing.T) {
	t.Parallel()

	mirror := deckmirror.New(maximum)

	_, first, _ := mirror.Subscribe()
	_, second, _ := mirror.Subscribe()

	defer second()

	if _, _, subscribed := mirror.Subscribe(); subscribed {
		t.Fatal("want a third subscription refused")
	}

	first()

	_, third, subscribed := mirror.Subscribe()
	if !subscribed {
		t.Fatal("want a place freed by cancelling")
	}

	third()
}

// TestMirror_NeverWaitsOnASlowSubscriber: the daemon's loop publishes, so a
// subscriber that does not read loses events rather than holding it up.
func TestMirror_NeverWaitsOnASlowSubscriber(t *testing.T) {
	t.Parallel()

	mirror := deckmirror.New(maximum)

	_, cancel, _ := mirror.Subscribe()
	defer cancel()

	for range 10000 {
		mirror.Press(otherTile)
	}
}

// TestMirror_NilDoesNothing: a daemon or editor given no mirror works as
// before.
func TestMirror_NilDoesNothing(t *testing.T) {
	t.Parallel()

	var mirror *deckmirror.Mirror

	mirror.Set(albumTile, albumKind, picture(10))
	mirror.Press(albumTile)
	mirror.Replace(nil)

	if _, found := mirror.Frame(albumTile, albumKind); found {
		t.Error("want no picture from no mirror")
	}

	if _, _, subscribed := mirror.Subscribe(); subscribed {
		t.Error("want no subscription to no mirror")
	}
}
