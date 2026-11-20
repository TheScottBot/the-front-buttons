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

// votingKeys builds voting keys for the board over the fake, from one
// registry, as the daemon does, so they share what they know of the session.
func votingKeys(t *testing.T, fake murals.Board, parameters ...*mural.VotingParameters) []interface {
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

	kind, _ := registry.Kind(mural.VotingKindName)

	keys := make([]interface {
		brick.LiveSource
		brick.LivePressable
		brick.LiveDrawn
		brick.DefaultLabelled
	}, 0, len(parameters))

	for _, each := range parameters {
		constructed, constructError := kind.Construct(each, brick.Ports{MuralBoard: fake})
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
			t.Fatal("want the voting key live, pressed with its state, drawn from it and labelled")
		}

		keys = append(keys, key)
	}

	return keys
}

// votingParameters is a voting key's parameters for the board.
func votingParameters(title string, votes int, anyoneCanEnd bool) *mural.VotingParameters {
	return &mural.VotingParameters{
		Appearance: noAppearance(), Link: boardLink, Title: title, Votes: votes, AnyoneCanEnd: anyoneCanEnd,
	}
}

// TestVoting_PressStartsThenEnds: the first press starts a session with the
// key's title and votes, Vote and three if left out; the next ends it; and
// the one after starts another.
func TestVoting_PressStartsThenEnds(t *testing.T) {
	t.Parallel()

	fake := newBoardFake(nil, false)
	keys := votingKeys(t, fake, votingParameters("", 0, false), votingParameters("Ideas", 5, true))

	for _, key := range []int{0, 0, 1} {
		pressError := keys[key].PressWithState(t.Context(), nothingShown())
		if pressError != nil {
			t.Fatal(pressError)
		}
	}

	want := []string{
		"vote " + boardID + ` "Vote" votes=3 anyone=false`,
		"end vote " + boardID,
		"vote " + boardID + ` "Ideas" votes=5 anyone=true`,
	}

	if calls := fake.recorded(); fmt.Sprint(calls) != fmt.Sprint(want) {
		t.Errorf("want %v, got %v", want, calls)
	}
}

// TestVoting_ShowsTheVotesThenTheTopCount: while the session runs the key
// shows the votes cast, read again in time to see new ones; once it ends it
// shows the most any one widget has; and with no session it shows nothing.
func TestVoting_ShowsTheVotesThenTheTopCount(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newBoardFake(nil, false)
		key := votingKeys(t, fake, votingParameters("", 0, false))[0]
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()

		if state := collected.last(); state.Text != "" || state.Playing {
			t.Errorf("before a session: want nothing shown, got %+v", state)
		}

		fake.setTally(murals.VotingTally{Total: 1, Top: 1})

		pressError := key.PressWithState(t.Context(), collected.last())
		if pressError != nil {
			t.Fatal(pressError)
		}

		synctest.Wait()

		if state := collected.last(); state.Text != "1 vote" || !state.Playing {
			t.Errorf("open: want 1 vote, running, got %+v", state)
		}

		fake.setTally(murals.VotingTally{Total: 7, Top: 4})
		time.Sleep(time.Minute)
		synctest.Wait()

		if state := collected.last(); state.Text != "7 votes" {
			t.Errorf("open later: want 7 votes, got %+v", state)
		}

		pressError = key.PressWithState(t.Context(), collected.last())
		if pressError != nil {
			t.Fatal(pressError)
		}

		synctest.Wait()

		if state := collected.last(); state.Text != "Top 4" || state.Playing {
			t.Errorf("ended: want Top 4, not running, got %+v", state)
		}
	})
}

// TestVoting_ShowsWhyItCannotRead: a tally Mural refuses shows why.
func TestVoting_ShowsWhyItCannotRead(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newBoardFake(nil, false)
		key := votingKeys(t, fake, votingParameters("", 0, false))[0]
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()

		pressError := key.PressWithState(t.Context(), collected.last())
		if pressError != nil {
			t.Fatal(pressError)
		}

		fake.mutex.Lock()
		fake.failure = fmt.Errorf("wrapped: %w", murals.ErrNotConnected)
		fake.mutex.Unlock()

		time.Sleep(time.Minute)
		synctest.Wait()

		if state := collected.last(); state.ErrorReason != notConnected {
			t.Errorf("want not_connected, got %+v", state)
		}
	})
}

// TestVoting_AFailedStartLeavesNoSession: a start Mural refuses shows why and
// leaves the key ready to start again.
func TestVoting_AFailedStartLeavesNoSession(t *testing.T) {
	t.Parallel()

	fake := newBoardFake(fmt.Errorf("wrapped: %w", murals.ErrRefused), false)
	key := votingKeys(t, fake, votingParameters("", 0, false))[0]

	failure := key.PressWithState(t.Context(), nothingShown())
	if category := brick.ReasonCategoryOf(failure); category != refusedReason {
		t.Errorf("want refused, got %q", category)
	}

	fake.mutex.Lock()
	fake.failure = nil
	fake.mutex.Unlock()

	pressError := key.PressWithState(t.Context(), nothingShown())
	if pressError != nil {
		t.Fatal(pressError)
	}

	if calls := fake.recorded(); len(calls) != 2 || calls[1] != "vote "+boardID+` "Vote" votes=3 anyone=false` {
		t.Errorf("want a second start, not an end, got %v", calls)
	}
}

// TestVoting_DrawsTheSession: while open the key shows the votes with a check
// box on a coloured background; once ended the top count with a trophy; and
// with no session its own look, labelled Start vote if it has no label.
func TestVoting_DrawsTheSession(t *testing.T) {
	t.Parallel()

	key := votingKeys(t, newBoardFake(nil, false), votingParameters("", 0, false))[0]

	drawn := func(state brick.LiveState) render.Appearance {
		appearance := render.Appearance{
			Label: ownLabel, Icon: "", IconImage: nil, IconMissing: false, BackgroundFromIcon: false,
			Background: noColour(), Status: render.StatusNone, ErrorReason: "", Picture: nil, Badge: "",
			Presses: render.PressCount{Done: 0, Of: 0},
		}

		key.DrawLive(state, &appearance)

		return appearance
	}

	open := drawn(timerShown("3 votes", true, false))
	ended := drawn(timerShown("Top 2", false, false))
	idle := drawn(nothingShown())

	if open.Label != "3 votes" || open.Icon != "checkbox" || open.Background == noColour() {
		t.Errorf("open: want 3 votes, a check box, a coloured key, got %+v", open)
	}

	if ended.Label != "Top 2" || ended.Icon != "trophy" || ended.Background != noColour() {
		t.Errorf("ended: want Top 2 and a trophy, got %+v", ended)
	}

	if idle.Label != ownLabel || idle.Icon != "" {
		t.Errorf("idle: want the key's own look, got %+v", idle)
	}

	if key.DefaultLabel() != "Start vote" {
		t.Errorf("want Start vote as the default label, got %q", key.DefaultLabel())
	}
}

// TestVoting_SharesOneWatchPerBoardAndNeedsMuralsAPI: voting keys for one
// board watch one source, and without the board port a key refuses to
// construct.
func TestVoting_SharesOneWatchPerBoardAndNeedsMuralsAPI(t *testing.T) {
	t.Parallel()

	other := votingParameters("", 0, false)
	other.Link = otherBoardLink

	keys := votingKeys(t, newBoardFake(nil, false), votingParameters("", 0, false), votingParameters("x", 2, true),
		other)

	if keys[0].LiveIdentity() != keys[1].LiveIdentity() || keys[2].LiveIdentity() == keys[0].LiveIdentity() {
		t.Errorf("want one identity per board, got %q, %q and %q",
			keys[0].LiveIdentity(), keys[1].LiveIdentity(), keys[2].LiveIdentity())
	}

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.VotingKindName)

	_, constructError := kind.Construct(votingParameters("", 0, false), brick.Ports{})
	if !errors.Is(constructError, brick.ErrPortMissing) {
		t.Errorf("want the port missing, got %v", constructError)
	}
}
