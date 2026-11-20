package mural_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/mural"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// stickyCountKey builds a sticky count key for the board, counting in the
// area titled area, or the whole board when it is empty.
//
//nolint:ireturn // keys are built as the interfaces the daemon uses them through
func stickyCountKey(t *testing.T, fake murals.Board, area string) liveBoardKey {
	t.Helper()

	return boardKey(t, mural.StickyCountKindName,
		&mural.StickyCountParameters{Appearance: noAppearance(), Link: boardLink, Area: area}, fake)
}

// parkingAreaID is the ID of the board's parking lot area.
const parkingAreaID = "0-2"

// watchKey watches a key into a collection until the test ends.
func watchKey(t *testing.T, key brick.LiveSource) *published {
	t.Helper()

	collected := &published{mutex: sync.Mutex{}, states: nil}
	watchContext, cancel := context.WithCancel(t.Context())

	t.Cleanup(cancel)

	go key.Watch(watchContext, collected.add)

	return collected
}

// TestStickyCount_CountsTheBoardOrAnArea: the key counts the sticky notes on
// the whole board, or in the area its title names, found once, and counts
// again in time to see new ones.
func TestStickyCount_CountsTheBoardOrAnArea(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newBoardFake(nil, false)
		fake.setStickies(1)

		collected := watchKey(t, stickyCountKey(t, fake, ""))

		synctest.Wait()

		if state := collected.last(); state.Text != "1 sticky" {
			t.Errorf("want 1 sticky, got %+v", state)
		}

		fake.setStickies(12)
		time.Sleep(time.Minute)
		synctest.Wait()

		if state := collected.last(); state.Text != "12 stickies" {
			t.Errorf("want 12 stickies, got %+v", state)
		}

		if calls := fake.recorded(); calls[0] != "count "+boardID+` ""` {
			t.Errorf("want the whole board counted, got %v", calls)
		}
	})

	synctest.Test(t, func(t *testing.T) {
		fake := newBoardFake(nil, false)
		fake.area = parkingAreaID
		fake.setStickies(4)

		collected := watchKey(t, stickyCountKey(t, fake, "Parking lot"))

		synctest.Wait()
		time.Sleep(time.Minute)
		synctest.Wait()

		if state := collected.last(); state.Text != "4 stickies" {
			t.Errorf("want 4 stickies in the area, got %+v", state)
		}

		calls := fake.recorded()
		finds := 0

		for _, call := range calls {
			if call == "find "+boardID+` "Parking lot"` {
				finds++
			}
		}

		if finds != 1 || calls[1] != "count "+boardID+` "0-2"` {
			t.Errorf("want the area found once and counted in, got %v", calls)
		}
	})
}

// TestStickyCount_SaysWhenTheAreaIsMissing: an area the board does not have
// shows no_area, and other refusals show why, as the other keys do.
func TestStickyCount_SaysWhenTheAreaIsMissing(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		collected := watchKey(t, stickyCountKey(t, newBoardFake(nil, false), "Nowhere"))

		synctest.Wait()

		if state := collected.last(); state.ErrorReason != "no_area" {
			t.Errorf("want no_area, got %+v", state)
		}
	})

	synctest.Test(t, func(t *testing.T) {
		refusing := newBoardFake(fmt.Errorf("wrapped: %w", murals.ErrNotConnected), false)
		collected := watchKey(t, stickyCountKey(t, refusing, ""))

		synctest.Wait()

		if state := collected.last(); state.ErrorReason != notConnected {
			t.Errorf("want not_connected, got %+v", state)
		}
	})
}

// TestStickyCount_PressCountsAgainAtOnce: a press has the key count again
// straight away.
func TestStickyCount_PressCountsAgainAtOnce(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newBoardFake(nil, false)
		key := stickyCountKey(t, fake, "")
		collected := watchKey(t, key)

		synctest.Wait()
		fake.setStickies(5)

		pressError := key.PressWithState(t.Context(), collected.last())
		if pressError != nil {
			t.Fatal(pressError)
		}

		synctest.Wait()

		if state := collected.last(); state.Text != "5 stickies" {
			t.Errorf("want 5 stickies at once, got %+v", state)
		}
	})
}

// TestStickyCount_DrawsTheCount: the key shows the count with a note icon,
// and with no label of its own says Stickies.
func TestStickyCount_DrawsTheCount(t *testing.T) {
	t.Parallel()

	key := stickyCountKey(t, newBoardFake(nil, false), "")

	appearance := render.Appearance{
		Label: ownLabel, Icon: "", IconImage: nil, IconMissing: false, BackgroundFromIcon: false,
		Background: noColour(), Status: render.StatusNone, ErrorReason: "", Picture: nil, Badge: "",
		Presses: render.PressCount{Done: 0, Of: 0},
	}

	key.DrawLive(timerShown("3 stickies", false, false), &appearance)

	if appearance.Label != "3 stickies" || appearance.Icon != "note" {
		t.Errorf("want 3 stickies with a note icon, got %+v", appearance)
	}

	if key.DefaultLabel() != "Stickies" {
		t.Errorf("want Stickies as the default label, got %q", key.DefaultLabel())
	}
}

// TestStickyCount_SharesOneWatchPerBoardAndAreaAndNeedsMuralsAPI: keys
// counting the same board and area watch one source, keys counting another
// area another, and without the board port a key refuses to construct.
func TestStickyCount_SharesOneWatchPerBoardAndAreaAndNeedsMuralsAPI(t *testing.T) {
	t.Parallel()

	first := stickyCountKey(t, newBoardFake(nil, false), "Ideas")
	second := stickyCountKey(t, newBoardFake(nil, false), "Ideas")
	other := stickyCountKey(t, newBoardFake(nil, false), "Parking lot")

	if first.LiveIdentity() != second.LiveIdentity() || other.LiveIdentity() == first.LiveIdentity() {
		t.Errorf("want one identity per board and area, got %q, %q and %q",
			first.LiveIdentity(), second.LiveIdentity(), other.LiveIdentity())
	}

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.StickyCountKindName)

	_, constructError := kind.Construct(
		&mural.StickyCountParameters{Appearance: noAppearance(), Link: boardLink, Area: ""}, brick.Ports{})
	if !errors.Is(constructError, brick.ErrPortMissing) {
		t.Errorf("want the port missing, got %v", constructError)
	}
}
