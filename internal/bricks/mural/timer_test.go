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

// boardID is the mural boardLink names, as Mural's API knows it.
const boardID = "team.1712345678901"

// timerFake records what the timer keys ask of Mural.
type timerFake struct {
	mutex   sync.Mutex
	calls   []string
	failure error
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

	fake := &timerFake{mutex: sync.Mutex{}, calls: nil, failure: nil}

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

// TestTimerKeys_SayWhatTheyDo: with no label of their own, the keys say what
// a press does: start and its time, or the time added.
func TestTimerKeys_SayWhatTheyDo(t *testing.T) {
	t.Parallel()

	fake := &timerFake{mutex: sync.Mutex{}, calls: nil, failure: nil}

	for want, key := range map[string]any{
		"Start 5:00": timerKey(t, mural.StartKindName,
			startParameters(0, false), fake),
		"Start 1:30": timerKey(t, mural.StartKindName,
			startParameters(90*time.Second, false), fake),
		"+1:00": timerKey(t, mural.AddMinuteKindName, addParameters(), fake),
		"+0:30": timerKey(t, mural.AddHalfMinuteKindName, addParameters(), fake),
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
		murals.ErrNotConnected: "not_connected",
		murals.ErrNotFound:     "not_found",
		murals.ErrRefused:      "refused",
		murals.ErrUnreachable:  "network",
	} {
		fake := &timerFake{mutex: sync.Mutex{}, calls: nil, failure: fmt.Errorf("wrapped: %w", failure)}
		key, _ := timerKey(t, mural.AddMinuteKindName,
			addParameters(), fake).(brick.Pressable)

		if category := brick.ReasonCategoryOf(key.Press(t.Context())); category != want {
			t.Errorf("%v: want %q, got %q", failure, want, category)
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
	} {
		kind, _ := registry.Kind(kindName)

		_, constructError := kind.Construct(parameters, brick.Ports{})
		if !errors.Is(constructError, brick.ErrPortMissing) {
			t.Errorf("%s: want the port missing, got %v", kindName, constructError)
		}
	}
}
