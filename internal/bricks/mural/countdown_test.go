package mural_test

import (
	"context"
	"fmt"
	"image/color"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/mural"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// countdownParameters is a countdown key's parameters for the board, with
// the time a press starts.
func countdownParameters(duration time.Duration) *mural.CountdownParameters {
	return &mural.CountdownParameters{Appearance: noAppearance(), Link: boardLink, Duration: duration, Silent: false}
}

// countdownKey builds a countdown key for the board over the fake.
func countdownKey(t *testing.T, fake *timerFake) interface {
	brick.LiveSource
	brick.LivePressable
	brick.LiveDrawn
} {
	t.Helper()

	key, _ := timerKey(t, mural.CountdownKindName, countdownParameters(0), fake).(interface {
		brick.LiveSource
		brick.LivePressable
		brick.LiveDrawn
	})
	if key == nil {
		t.Fatal("want the countdown key live, pressed with its state and drawn from it")
	}

	return key
}

// The countdown test's texts: the key's own label, and a time it shows.
const (
	ownLabel  = "Timer"
	fourLeft  = "4:00"
	nothingIn = ""
)

// timerShown is the state of a countdown showing a time, running or not, and
// urgent or not; nothingShown is one with no timer.
func timerShown(text string, playing bool, urgent bool) brick.LiveState {
	return brick.LiveState{
		Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: nil, Playing: playing,
		Text: text, Urgent: urgent,
	}
}

func nothingShown() brick.LiveState { return timerShown(nothingIn, false, false) }

// noColour is a key's background when nothing colours it.
func noColour() color.RGBA { return color.RGBA{R: 0, G: 0, B: 0, A: 0} }

// published collects what a watch publishes.
type published struct {
	mutex  sync.Mutex
	states []brick.LiveState
}

func (collected *published) add(state brick.LiveState) {
	collected.mutex.Lock()
	defer collected.mutex.Unlock()

	collected.states = append(collected.states, state)
}

func (collected *published) last() brick.LiveState {
	collected.mutex.Lock()
	defer collected.mutex.Unlock()

	if len(collected.states) == 0 {
		return nothingShown()
	}

	return collected.states[len(collected.states)-1]
}

func (fake *timerFake) setState(state murals.TimerState) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.state = state
}

func (fake *timerFake) reads() int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	count := 0

	for _, call := range fake.calls {
		if call == "read "+boardID {
			count++
		}
	}

	return count
}

// TestCountdown_ShowsTheTimerTickingDown: a running timer is shown as minutes
// and seconds, counting down each second between reads, and marked urgent in
// its last minute.
func TestCountdown_ShowsTheTimerTickingDown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newTimerFake(nil, murals.TimerState{Running: true, Paused: false, Remaining: 65 * time.Second})
		key := countdownKey(t, fake)
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()

		if state := collected.last(); state.Text != "1:05" || !state.Playing || state.Urgent {
			t.Errorf("at first: want 1:05 running, not urgent, got %+v", state)
		}

		time.Sleep(time.Second)
		synctest.Wait()

		if state := collected.last(); state.Text != "1:04" {
			t.Errorf("a second later: want 1:04, got %+v", state)
		}

		time.Sleep(5 * time.Second)
		synctest.Wait()

		if state := collected.last(); state.Text != "0:59" || !state.Urgent {
			t.Errorf("in the last minute: want 0:59 urgent, got %+v", state)
		}

		if reads := fake.reads(); reads != 1 {
			t.Errorf("want the countdown kept between reads, got %d reads", reads)
		}
	})
}

// TestCountdown_ShowsPausedAndIdleTimers: a paused timer holds its time, not
// running; a board with no timer running shows nothing of its own; and the
// timer is read again in time to see changes made elsewhere.
func TestCountdown_ShowsPausedAndIdleTimers(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newTimerFake(nil, murals.TimerState{Running: true, Paused: true, Remaining: 2 * time.Minute})
		key := countdownKey(t, fake)
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()
		time.Sleep(3 * time.Second)
		synctest.Wait()

		if state := collected.last(); state.Text != "2:00" || state.Playing {
			t.Errorf("paused: want 2:00 held, not running, got %+v", state)
		}

		fake.setState(noTimer())
		time.Sleep(time.Minute)
		synctest.Wait()

		if state := collected.last(); state.Text != "" || state.Playing || state.ErrorReason != "" {
			t.Errorf("idle: want nothing shown, got %+v", state)
		}
	})
}

// TestCountdown_ShowsWhyItCannotRead: a read Mural refuses shows why on the
// key, as the other timer keys do.
func TestCountdown_ShowsWhyItCannotRead(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newTimerFake(fmt.Errorf("wrapped: %w", murals.ErrNotConnected), noTimer())
		key := countdownKey(t, fake)
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

// TestCountdown_PressStartsPausesOrResumes: a press starts the timer when none
// runs, for five minutes unless given another time, pauses it while it runs,
// and resumes it while it is paused, each from the state shown.
func TestCountdown_PressStartsPausesOrResumes(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		state brick.LiveState
		want  string
	}{
		{"idle", nothingShown(), "start " + boardID + " 5m0s sound=true"},
		{"running", timerShown("1:00", true, false), "pause " + boardID + " true"},
		{"paused", timerShown("1:00", false, false), "pause " + boardID + " false"},
	} {
		fake := newTimerFake(nil, noTimer())

		pressError := countdownKey(t, fake).PressWithState(t.Context(), test.state)
		if pressError != nil {
			t.Fatal(pressError)
		}

		if fmt.Sprint(fake.calls) != fmt.Sprint([]string{test.want}) {
			t.Errorf("%s: want %v, got %v", test.name, test.want, fake.calls)
		}
	}

	failing := newTimerFake(fmt.Errorf("wrapped: %w", murals.ErrRefused), noTimer())

	failure := countdownKey(t, failing).PressWithState(t.Context(), nothingShown())
	if category := brick.ReasonCategoryOf(failure); category != refusedReason {
		t.Errorf("want a refused press to show refused, got %q", category)
	}
}

// TestCountdown_PressIsShownAtOnce: after a press the timer is read again
// straight away, so the key does not wait for its next read to show it.
func TestCountdown_PressIsShownAtOnce(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fake := newTimerFake(nil, noTimer())
		key := countdownKey(t, fake)
		collected := &published{mutex: sync.Mutex{}, states: nil}

		watchContext, cancel := context.WithCancel(t.Context())
		defer cancel()

		go key.Watch(watchContext, collected.add)

		synctest.Wait()

		fake.setState(murals.TimerState{Running: true, Paused: false, Remaining: 5 * time.Minute})

		pressError := key.PressWithState(t.Context(), collected.last())
		if pressError != nil {
			t.Fatal(pressError)
		}

		synctest.Wait()

		if state := collected.last(); state.Text != "5:00" || !state.Playing {
			t.Errorf("want the started timer shown at once, got %+v", state)
		}
	})
}

// TestCountdown_DrawsTheTime: the key shows the time as its label, a pause
// icon while paused, and a red background in the last minute; with no timer
// it keeps its own label.
func TestCountdown_DrawsTheTime(t *testing.T) {
	t.Parallel()

	key := countdownKey(t, newTimerFake(nil, noTimer()))

	drawn := func(state brick.LiveState) render.Appearance {
		appearance := render.Appearance{
			Label: ownLabel, Icon: "", IconImage: nil, IconMissing: false, BackgroundFromIcon: false,
			Background: noColour(), Status: render.StatusNone, ErrorReason: "", Picture: nil, Badge: "",
			Presses: render.PressCount{Done: 0, Of: 0},
		}

		key.DrawLive(state, &appearance)

		return appearance
	}

	running := drawn(timerShown(fourLeft, true, false))
	paused := drawn(timerShown(fourLeft, false, false))
	urgent := drawn(timerShown("0:30", true, true))
	idle := drawn(nothingShown())

	if running.Label != fourLeft || running.Icon != "hourglass" || running.Background != noColour() {
		t.Errorf("running: want 4:00 with an hourglass, got %+v", running)
	}

	if paused.Label != fourLeft || paused.Icon != "player-pause" {
		t.Errorf("paused: want 4:00 with a pause icon, got %+v", paused)
	}

	if urgent.Label != "0:30" || urgent.Background == noColour() {
		t.Errorf("urgent: want 0:30 on a coloured background, got %+v", urgent)
	}

	if idle.Label != ownLabel || idle.Icon != "" {
		t.Errorf("idle: want the key's own label, got %+v", idle)
	}
}

// TestCountdown_SharesOneWatchPerBoard: countdown keys for one board watch
// one source, and keys for another board another.
func TestCountdown_SharesOneWatchPerBoard(t *testing.T) {
	t.Parallel()

	first := countdownKey(t, newTimerFake(nil, noTimer()))
	second := countdownKey(t, newTimerFake(nil, noTimer()))

	parameters := countdownParameters(0)
	parameters.Link = otherBoardLink

	other, _ := timerKey(t, mural.CountdownKindName, parameters, newTimerFake(nil, noTimer())).(brick.LiveSource)

	if first.LiveIdentity() != second.LiveIdentity() || other == nil ||
		other.LiveIdentity() == first.LiveIdentity() {
		t.Errorf("want one identity per board, got %q, %q and another board's %v",
			first.LiveIdentity(), second.LiveIdentity(), other)
	}
}
