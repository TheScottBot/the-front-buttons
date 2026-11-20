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

// boardFake records what the private mode key asks of Mural.
type boardFake struct {
	mutex   sync.Mutex
	calls   []string
	failure error
	active  bool
}

func newBoardFake(failure error, active bool) *boardFake {
	return &boardFake{mutex: sync.Mutex{}, calls: nil, failure: failure, active: active}
}

func (fake *boardFake) StartPrivateMode(_ context.Context, muralID string, hideAuthors bool) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, fmt.Sprintf("start %s hide=%v", muralID, hideAuthors))

	return fake.failure
}

func (fake *boardFake) EndPrivateMode(_ context.Context, muralID string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, "end "+muralID)

	return fake.failure
}

func (fake *boardFake) ReadPrivateMode(_ context.Context, muralID string) (bool, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, "read "+muralID)

	return fake.active, fake.failure
}

func (fake *boardFake) setActive(active bool) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.active = active
}

func (fake *boardFake) recorded() []string {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return append([]string(nil), fake.calls...)
}

// privateModeKey builds a private mode key for the board over the fake,
// hiding authors or not.
func privateModeKey(t *testing.T, fake murals.Board, hideAuthors bool) interface {
	brick.LiveSource
	brick.LivePressable
	brick.LiveDrawn
	brick.DefaultLabelled
} {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.PrivateModeKindName)

	constructed, constructError := kind.Construct(
		&mural.PrivateModeParameters{Appearance: noAppearance(), Link: boardLink, HideAuthors: hideAuthors},
		brick.Ports{MuralBoard: fake})
	if constructError != nil {
		t.Fatal(constructError)
	}

	key, _ := constructed.(interface {
		brick.LiveSource
		brick.LivePressable
		brick.LiveDrawn
		brick.DefaultLabelled
	})
	if key == nil {
		t.Fatal("want the private mode key live, pressed with its state, drawn from it and labelled")
	}

	return key
}

// modeShown is the state of a private mode key, on or off.
func modeShown(on bool) brick.LiveState { return timerShown(nothingIn, on, false) }

// TestPrivateMode_ShowsWhetherItIsOn: the key publishes private mode on or
// off, reads the board again in time to see it changed elsewhere, and shows
// why when it cannot read.
func TestPrivateMode_ShowsWhetherItIsOn(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newBoardFake(nil, true)
		key := privateModeKey(t, fake, false)
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()

		if state := collected.last(); !state.Playing || state.ErrorReason != "" {
			t.Errorf("want private mode shown on, got %+v", state)
		}

		fake.setActive(false)
		time.Sleep(time.Minute)
		synctest.Wait()

		if state := collected.last(); state.Playing {
			t.Errorf("want private mode ended elsewhere shown off, got %+v", state)
		}
	})

	synctest.Test(t, func(t *testing.T) {
		key := privateModeKey(t, newBoardFake(fmt.Errorf("wrapped: %w", murals.ErrNotConnected), false), false)
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()

		if state := collected.last(); state.ErrorReason != notConnected {
			t.Errorf("want not_connected, got %+v", state)
		}
	})
}

// TestPrivateMode_PressTurnsItOnOrOff: a press starts private mode while it
// is off, hiding authors if the key says so, and ends it while it is on.
func TestPrivateMode_PressTurnsItOnOrOff(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		on    bool
		hide  bool
		wants string
	}{
		{"off", false, false, "start " + boardID + " hide=false"},
		{"off, hiding authors", false, true, "start " + boardID + " hide=true"},
		{"on", true, false, "end " + boardID},
	} {
		fake := newBoardFake(nil, test.on)

		pressError := privateModeKey(t, fake, test.hide).PressWithState(t.Context(), modeShown(test.on))
		if pressError != nil {
			t.Fatal(pressError)
		}

		if calls := fake.recorded(); fmt.Sprint(calls) != fmt.Sprint([]string{test.wants}) {
			t.Errorf("%s: want %v, got %v", test.name, test.wants, calls)
		}
	}

	failure := privateModeKey(t, newBoardFake(fmt.Errorf("wrapped: %w", murals.ErrNotFound), false), false).
		PressWithState(t.Context(), modeShown(false))
	if category := brick.ReasonCategoryOf(failure); category != "not_found" {
		t.Errorf("want a press Mural refuses to show why, got %q", category)
	}
}

// TestPrivateMode_PressIsShownAtOnce: after a press the board is read again
// straight away.
func TestPrivateMode_PressIsShownAtOnce(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newBoardFake(nil, false)
		key := privateModeKey(t, fake, false)
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()
		fake.setActive(true)

		pressError := key.PressWithState(t.Context(), collected.last())
		if pressError != nil {
			t.Fatal(pressError)
		}

		synctest.Wait()

		if state := collected.last(); !state.Playing {
			t.Errorf("want private mode shown on at once, got %+v", state)
		}
	})
}

// TestPrivateMode_DrawsWhetherItIsOn: while on, the key shows a closed eye on
// a coloured background; while off it keeps its own look; and with no label
// of its own it says Private.
func TestPrivateMode_DrawsWhetherItIsOn(t *testing.T) {
	t.Parallel()

	key := privateModeKey(t, newBoardFake(nil, false), false)

	drawn := func(state brick.LiveState) render.Appearance {
		appearance := render.Appearance{
			Label: ownLabel, Icon: "", IconImage: nil, IconMissing: false, BackgroundFromIcon: false,
			Background: noColour(), Status: render.StatusNone, ErrorReason: "", Picture: nil, Badge: "",
			Presses: render.PressCount{Done: 0, Of: 0},
		}

		key.DrawLive(state, &appearance)

		return appearance
	}

	on := drawn(modeShown(true))
	off := drawn(modeShown(false))

	if on.Icon != "eye-off" || on.Background == noColour() {
		t.Errorf("on: want a closed eye on a coloured background, got %+v", on)
	}

	if off.Icon != "eye" || off.Background != noColour() || off.Label != ownLabel {
		t.Errorf("off: want an open eye and the key's own look, got %+v", off)
	}

	if key.DefaultLabel() != "Private" {
		t.Errorf("want Private as the default label, got %q", key.DefaultLabel())
	}
}

// TestPrivateMode_SharesOneWatchPerBoard: private mode keys for one board
// watch one source, and keys for another board another.
func TestPrivateMode_SharesOneWatchPerBoard(t *testing.T) {
	t.Parallel()

	first := privateModeKey(t, newBoardFake(nil, false), false)
	second := privateModeKey(t, newBoardFake(nil, false), true)

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.PrivateModeKindName)

	constructed, _ := kind.Construct(&mural.PrivateModeParameters{
		Appearance: noAppearance(), Link: "https://app.mural.co/t/team/m/team/1799999999999/0a1b2c3d", HideAuthors: false,
	}, brick.Ports{MuralBoard: newBoardFake(nil, false)})

	other, _ := constructed.(brick.LiveSource)

	if first.LiveIdentity() != second.LiveIdentity() || other == nil ||
		other.LiveIdentity() == first.LiveIdentity() {
		t.Errorf("want one identity per board, got %q, %q and another board's %v",
			first.LiveIdentity(), second.LiveIdentity(), other)
	}
}

// TestPrivateMode_NeedsMuralsAPI: without the board port the key refuses to
// construct.
func TestPrivateMode_NeedsMuralsAPI(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.PrivateModeKindName)

	_, constructError := kind.Construct(
		&mural.PrivateModeParameters{Appearance: noAppearance(), Link: boardLink, HideAuthors: false}, brick.Ports{})
	if !errors.Is(constructError, brick.ErrPortMissing) {
		t.Errorf("want the port missing, got %v", constructError)
	}
}
