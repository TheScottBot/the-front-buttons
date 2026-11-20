package mural_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/mural"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
)

// notConnected is the reason a key shows when Mural's sign in is not good.
const notConnected = "not_connected"

// boardID is the mural boardLink names, as Mural's API knows it.
const boardID = "team.1712345678901"

// timerFake records what the timer keys ask of Mural.
type timerFake struct {
	mutex   sync.Mutex
	calls   []string
	failure error
	state   murals.TimerState
}

// newTimerFake is a fake that fails with failure, nil for none, and reads the
// timer as state.
func newTimerFake(failure error, state murals.TimerState) *timerFake {
	return &timerFake{mutex: sync.Mutex{}, calls: nil, failure: failure, state: state}
}

// noTimer is a board's timer when none runs.
func noTimer() murals.TimerState {
	return murals.TimerState{Running: false, Paused: false, Remaining: 0}
}

func (fake *timerFake) StopTimer(_ context.Context, muralID string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, "stop "+muralID)

	return fake.failure
}

func (fake *timerFake) PauseTimer(_ context.Context, muralID string, paused bool) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, fmt.Sprintf("pause %s %v", muralID, paused))

	return fake.failure
}

func (fake *timerFake) ReadTimer(_ context.Context, muralID string) (murals.TimerState, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, "read "+muralID)

	return fake.state, fake.failure
}

func (fake *timerFake) StartTimer(_ context.Context, muralID string, duration time.Duration, sound bool) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, fmt.Sprintf("start %s %s sound=%v", muralID, duration, sound))

	return fake.failure
}

func (fake *timerFake) AddTime(_ context.Context, muralID string, extra time.Duration) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.calls = append(fake.calls, fmt.Sprintf("add %s %s", muralID, extra))

	return fake.failure
}

// timerKey builds a timer kind with parameters over the fake.
func timerKey(t *testing.T, kindName string, parameters any, timer murals.Timer) any {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, found := registry.Kind(kindName)
	if !found {
		t.Fatalf("want the kind %s", kindName)
	}

	constructed, constructError := kind.Construct(parameters, brick.Ports{MuralTimer: timer})
	if constructError != nil {
		t.Fatal(constructError)
	}

	return constructed
}

// startParameters is a start key's parameters for the board, for a
// duration, silent or not.
func startParameters(duration time.Duration, silent bool) *mural.StartParameters {
	return &mural.StartParameters{Appearance: noAppearance(), Link: boardLink, Duration: duration, Silent: silent}
}

// boardParameters is a stop or pause key's parameters for the board.
func boardParameters() *mural.BoardParameters {
	return &mural.BoardParameters{Appearance: noAppearance(), Link: boardLink}
}

// addParameters is an add key's parameters for the board.
func addParameters() *mural.AddParameters {
	return &mural.AddParameters{Appearance: noAppearance(), Link: boardLink}
}

// press presses a key, failing on its error.
func press(t *testing.T, key any) {
	t.Helper()

	pressable, _ := key.(brick.Pressable)

	pressError := pressable.Press(t.Context())
	if pressError != nil {
		t.Fatal(pressError)
	}
}

// TestTimerKeys_ActOnTheirOwnBoard: each timer key acts on the board its
// link names, and no other: start for its duration, five minutes if left
// out, with the sound unless silent; and the add keys a minute and thirty
// seconds.
func TestTimerKeys_ActOnTheirOwnBoard(t *testing.T) {
	t.Parallel()

	fake := newTimerFake(nil, noTimer())

	press(t, timerKey(t, mural.StartKindName,
		startParameters(0, false), fake))
	press(t, timerKey(t, mural.StartKindName,
		startParameters(90*time.Second, true), fake))
	press(t, timerKey(t, mural.AddMinuteKindName,
		addParameters(), fake))
	press(t, timerKey(t, mural.AddHalfMinuteKindName,
		addParameters(), fake))

	want := []string{
		"start " + boardID + " 5m0s sound=true", "start " + boardID + " 1m30s sound=false",
		"add " + boardID + " 1m0s", "add " + boardID + " 30s",
	}

	if fmt.Sprint(fake.calls) != fmt.Sprint(want) {
		t.Errorf("want %v, got %v", want, fake.calls)
	}
}

// TestTimerKeys_StopTheirBoardsTimer: the stop key stops the timer of the
// board its link names.
func TestTimerKeys_StopTheirBoardsTimer(t *testing.T) {
	t.Parallel()

	fake := newTimerFake(nil, noTimer())

	press(t, timerKey(t, mural.StopKindName, boardParameters(), fake))

	if want := []string{"stop " + boardID}; fmt.Sprint(fake.calls) != fmt.Sprint(want) {
		t.Errorf("want %v, got %v", want, fake.calls)
	}
}

// TestTimerKeys_PauseOrResumeAsTheTimerIs: the pause key reads the board's
// timer and pauses it when it runs, or resumes it when it is paused.
func TestTimerKeys_PauseOrResumeAsTheTimerIs(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		state murals.TimerState
		want  string
	}{
		{murals.TimerState{Running: true, Paused: false, Remaining: time.Minute}, "pause " + boardID + " true"},
		{murals.TimerState{Running: true, Paused: true, Remaining: time.Minute}, "pause " + boardID + " false"},
	} {
		fake := newTimerFake(nil, test.state)

		press(t, timerKey(t, mural.PauseKindName, boardParameters(), fake))

		if want := []string{"read " + boardID, test.want}; fmt.Sprint(fake.calls) != fmt.Sprint(want) {
			t.Errorf("%+v: want %v, got %v", test.state, want, fake.calls)
		}
	}
}

// TestTimerKeys_PauseSaysWhenNoTimerRuns: pressed with no timer running, the
// pause key changes nothing and shows that there is no timer.
func TestTimerKeys_PauseSaysWhenNoTimerRuns(t *testing.T) {
	t.Parallel()

	fake := newTimerFake(nil, noTimer())
	key, _ := timerKey(t, mural.PauseKindName, boardParameters(), fake).(brick.Pressable)

	if category := brick.ReasonCategoryOf(key.Press(t.Context())); category != "no_timer" {
		t.Errorf("want no_timer, got %q", category)
	}

	if want := []string{"read " + boardID}; fmt.Sprint(fake.calls) != fmt.Sprint(want) {
		t.Errorf("want only a read, got %v", fake.calls)
	}
}

// TestTimerKeys_SayWhatTheyDo: with no label of their own, the keys say what
// a press does: start and its time, or the time added.
func TestTimerKeys_SayWhatTheyDo(t *testing.T) {
	t.Parallel()

	fake := newTimerFake(nil, noTimer())

	for want, key := range map[string]any{
		"Start 5:00": timerKey(t, mural.StartKindName,
			startParameters(0, false), fake),
		"Start 1:30": timerKey(t, mural.StartKindName,
			startParameters(90*time.Second, false), fake),
		"+1:00": timerKey(t, mural.AddMinuteKindName, addParameters(), fake),
		"+0:30": timerKey(t, mural.AddHalfMinuteKindName, addParameters(), fake),
		"Stop":  timerKey(t, mural.StopKindName, boardParameters(), fake),
		"Pause": timerKey(t, mural.PauseKindName, boardParameters(), fake),
	} {
		labelled, _ := key.(brick.DefaultLabelled)
		if labelled == nil || labelled.DefaultLabel() != want {
			t.Errorf("want %q", want)
		}
	}
}

// TestTimerKeys_ShowWhyMuralRefused: a press Mural refuses shows why on the
// key: not set up, not connected, no such mural, refused, or not reached.
func TestTimerKeys_ShowWhyMuralRefused(t *testing.T) {
	t.Parallel()

	for failure, want := range map[error]string{
		murals.ErrNotSetUp:     "not_set_up",
		murals.ErrNotConnected: notConnected,
		murals.ErrNotFound:     "not_found",
		murals.ErrRefused:      "refused",
		murals.ErrUnreachable:  "network",
	} {
		fake := newTimerFake(fmt.Errorf("wrapped: %w", failure), noTimer())
		for kindName, parameters := range map[string]any{
			mural.AddMinuteKindName: addParameters(),
			mural.StopKindName:      boardParameters(),
			mural.PauseKindName:     boardParameters(),
		} {
			key, _ := timerKey(t, kindName, parameters, fake).(brick.Pressable)

			if category := brick.ReasonCategoryOf(key.Press(t.Context())); category != want {
				t.Errorf("%s, %v: want %q, got %q", kindName, failure, want, category)
			}
		}
	}
}

// TestTimerKeys_NeedMuralsAPI: without the timer port, a timer key refuses to
// construct.
func TestTimerKeys_NeedMuralsAPI(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	for kindName, parameters := range map[string]any{
		mural.StartKindName:         startParameters(0, false),
		mural.AddMinuteKindName:     addParameters(),
		mural.AddHalfMinuteKindName: addParameters(),
		mural.StopKindName:          boardParameters(),
		mural.PauseKindName:         boardParameters(),
	} {
		kind, _ := registry.Kind(kindName)

		_, constructError := kind.Construct(parameters, brick.Ports{})
		if !errors.Is(constructError, brick.ErrPortMissing) {
			t.Errorf("%s: want the port missing, got %v", kindName, constructError)
		}
	}
}
