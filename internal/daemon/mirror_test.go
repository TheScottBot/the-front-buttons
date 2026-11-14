package daemon_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/deckmirror"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The live key on livePages' second page, shown on no deck while the first
// page is.
//
//nolint:gochecknoglobals // a fixed tile
var hiddenLiveTile = deckmirror.Tile{Page: 1, Position: layout.Position{Row: 0, Column: 0}}

// drain takes every event waiting.
func drain(events <-chan deckmirror.Event) []deckmirror.Event {
	var taken []deckmirror.Event

	for {
		select {
		case event := <-events:
			taken = append(taken, event)
		default:
			return taken
		}
	}
}

// TestDaemon_MirrorsLiveKeysOnEveryPage: the mirror has a picture of every
// live key on every page, drawn as the deck draws it, whether or not its page
// is showing, and none of a key that never changes; a source publishing
// redraws its keys there, and subscribers hear which tiles.
func TestDaemon_MirrorsLiveKeysOnEveryPage(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)

		events, cancel, _ := running.mirror.Subscribe()
		defer cancel()

		running.configurations <- livePages()

		time.Sleep(settle)
		synctest.Wait()

		if _, found := running.mirror.Frame(hiddenLiveTile, testsupport.LiveKindName); !found {
			t.Fatal("want the hidden page's live key mirrored")
		}

		static := deckmirror.Tile{Page: 0, Position: layout.Position{Row: 0, Column: 0}}
		if _, found := running.mirror.Frame(static, testsupport.ButtonKindName); found {
			t.Error("want no picture of a key that never changes")
		}

		drain(events)

		running.live.Set("shared", brick.LiveState{
			Status: render.StatusFailure, ErrorReason: "", TargetURL: liveRunURL, Picture: nil, Playing: false,
		})
		time.Sleep(settle)
		synctest.Wait()

		if heard := drain(events); !slices.Contains(heard, deckmirror.Event{Kind: deckmirror.Redrawn, Tile: hiddenLiveTile}) {
			t.Fatalf("want the hidden key's redraw heard, got %+v", heard)
		}

		drawn, _ := running.mirror.Frame(hiddenLiveTile, testsupport.LiveKindName)
		failure := running.renderer.Render(render.Appearance{Label: buildLabel, Status: render.StatusFailure})
		success := running.renderer.Render(render.Appearance{Label: buildLabel, Status: render.StatusSuccess})

		if distance(drawn, failure) >= distance(drawn, success) {
			t.Fatal("want the mirror to show the failure, as the deck would")
		}

		running.stop(t)
	})
}

// TestDaemon_TellsTheMirrorOfPressesAndReleases: a key pressed on the deck,
// a paging key among them, is heard by its tile on the page showing, and its
// release by the same tile, though paging has changed the page by then.
func TestDaemon_TellsTheMirrorOfPressesAndReleases(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- livePages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		events, cancel, _ := running.mirror.Subscribe()
		defer cancel()

		tap(device, 0)
		tap(device, forwardKey)

		grid, _ := layout.NewGrid(xlModel(t).Rows, xlModel(t).Columns)
		forward, _ := grid.PositionOfKey(forwardKey)
		first := deckmirror.Tile{Page: 0, Position: layout.Position{Row: 0, Column: 0}}
		paging := deckmirror.Tile{Page: 0, Position: forward}
		want := []deckmirror.Event{
			{Kind: deckmirror.Pressed, Tile: first},
			{Kind: deckmirror.Released, Tile: first},
			{Kind: deckmirror.Pressed, Tile: paging},
			{Kind: deckmirror.Released, Tile: paging},
		}

		var heard []deckmirror.Event

		for _, event := range drain(events) {
			if event.Kind == deckmirror.Pressed || event.Kind == deckmirror.Released {
				heard = append(heard, event)
			}
		}

		if !slices.Equal(heard, want) {
			t.Fatalf("want %+v, got %+v", want, heard)
		}

		running.stop(t)
	})
}

// TestDaemon_ReleasesHeldKeys_WhenTheDeckGoes: a key held down as the deck is
// unplugged is let go of in the mirror, as no release will come for it.
func TestDaemon_ReleasesHeldKeys_WhenTheDeckGoes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- livePages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		events, cancel, _ := running.mirror.Subscribe()
		defer cancel()

		device.PressKeys(0)
		time.Sleep(testReadTimeout)
		synctest.Wait()
		running.watcher.Detach("deck-a")
		time.Sleep(settle)
		synctest.Wait()

		held := deckmirror.Tile{Page: 0, Position: layout.Position{Row: 0, Column: 0}}
		if heard := drain(events); !slices.Contains(heard, deckmirror.Event{Kind: deckmirror.Released, Tile: held}) {
			t.Fatalf("want the held key let go of, got %+v", heard)
		}

		running.stop(t)
	})
}
