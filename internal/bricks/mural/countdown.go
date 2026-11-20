package mural

import (
	"context"
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The countdown key, mural_timer, shows a board's timer as it runs, in minutes
// and seconds, with a pause icon while it is paused and a red background in
// its last minute. A press starts the timer when none runs, pauses it while it
// runs, and resumes it while it is paused.

// CountdownKindName is the countdown key's name in configuration.
const CountdownKindName = "mural_timer"

// The countdown's timings: the timer is read every countdownRead, to see
// changes made on the board itself, and counted down each second between
// reads; a read that fails is tried again after countdownRetry. The last
// minute is urgent.
const (
	countdownRead  = 10 * time.Second
	countdownTick  = time.Second
	countdownRetry = 30 * time.Second
	urgentWithin   = time.Minute
)

// The countdown's icons: an hourglass while the timer runs, pause while it is
// paused.
const (
	runningIcon = "hourglass"
	pausedIcon  = "player-pause"
)

// urgentBackground is the countdown's background in the timer's last minute.
//
//nolint:gochecknoglobals // a colour, never changed
var urgentBackground = color.RGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xFF} //nolint:mnd // a colour's own channels

// countdownIdentity names the live source a board's countdown keys share.
const countdownIdentity = "mural_timer:"

// CountdownParameters is the countdown key's one declaration of its
// parameters.
type CountdownParameters struct {
	brick.Appearance

	Link     string        `brick:"link,required" bounds:"1,2048"                                             format:"mural_board"                                                                                description:"The board's address, as its browser shows it; the key shows that board's timer and runs it."`
	Duration time.Duration `brick:"duration"      bounds:"1s,90m"                                             description:"How long a press starts the timer for when none is running; five minutes if left out."`
	Silent   bool          `brick:"silent"        description:"Start the timer without its sound at the end."`
}

// wakes holds, for each board, the channel a press uses to have the board's
// countdown read the timer again at once. Keys made from one registry share
// them, so a press on any key for a board wakes the one watch the live hub
// runs for it.
type wakes struct {
	mutex    sync.Mutex
	channels map[string]chan struct{}
}

// forBoard is the board's wake channel, made on first use. It holds one wake,
// so a press never blocks and wakes made together read once.
func (board *wakes) forBoard(muralID string) chan struct{} {
	board.mutex.Lock()
	defer board.mutex.Unlock()

	channel, found := board.channels[muralID]
	if !found {
		channel = make(chan struct{}, 1)
		board.channels[muralID] = channel
	}

	return channel
}

// registerCountdown adds the countdown kind to a registry.
func registerCountdown(registry *brick.Registry) error {
	shared := &wakes{mutex: sync.Mutex{}, channels: map[string]chan struct{}{}}

	return brick.Register(registry, CountdownKindName, func( //nolint:wrapcheck // Register already names the kind
		parameters CountdownParameters, ports brick.Ports,
	) (any, error) {
		if ports.MuralTimer == nil {
			return nil, fmt.Errorf("%w: %s needs Mural's API", brick.ErrPortMissing, CountdownKindName)
		}

		duration := parameters.Duration
		if duration == 0 {
			duration = defaultDuration
		}

		// The link was checked as a board's address when the configuration
		// was read, so it always names one.
		muralID, _ := murals.BoardID(parameters.Link)

		return countdown{
			muralID: muralID, timer: ports.MuralTimer, duration: duration, sound: !parameters.Silent,
			wake: shared.forBoard(muralID),
		}, nil
	})
}

// countdown is one countdown key.
type countdown struct {
	muralID  string
	timer    murals.Timer
	duration time.Duration
	sound    bool
	wake     chan struct{}
}

// LiveIdentity is shared by every countdown key for the board.
func (key countdown) LiveIdentity() string { return countdownIdentity + key.muralID }

// DefaultLabel says what a press does when no timer runs.
func (key countdown) DefaultLabel() string { return "Start " + clockTime(key.duration) }

// Watch reads the board's timer and publishes it, counting down each second
// between reads, until the context ends. A press wakes it to read at once.
func (key countdown) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	var last brick.LiveState

	first := true

	show := func(state brick.LiveState) {
		if first || state != last {
			publish(state)

			last, first = state, false
		}
	}

	for {
		timer, readError := key.timer.ReadTimer(watchContext, key.muralID)
		if watchContext.Err() != nil {
			return
		}

		if readError != nil {
			show(brick.LiveState{
				Status: render.StatusNone, ErrorReason: timerFailureCategory(readError), TargetURL: "",
				Picture: nil, Playing: false, Text: "", Urgent: false,
			})

			if !key.pause(watchContext, countdownRetry) {
				return
			}

			continue
		}

		if !key.countDown(watchContext, timer, show) {
			return
		}
	}
}

// countdownState is what the key shows for a timer with some time left: its
// time, whether it runs, and whether it is in its last minute; nothing for a
// board with no timer running.
func countdownState(timer murals.TimerState, left time.Duration) brick.LiveState {
	state := brick.LiveState{
		Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: nil, Playing: false, Text: "", Urgent: false,
	}

	if !timer.Running {
		return state
	}

	state.Text = clockTime(left)
	state.Playing = !timer.Paused
	state.Urgent = left < urgentWithin

	return state
}

// DrawLive shows the time and whether it runs, or leaves the key's own label
// and icon when no timer runs.
func (countdown) DrawLive(state brick.LiveState, appearance *render.Appearance) {
	if state.Text == "" {
		return
	}

	appearance.Label = state.Text

	appearance.Icon = pausedIcon
	if state.Playing {
		appearance.Icon = runningIcon
	}

	if state.Urgent {
		appearance.Background = urgentBackground
	}
}

// PressWithState starts the timer when none runs, pauses it while it runs,
// and resumes it while it is paused, then has the watch read it at once.
func (key countdown) PressWithState(pressContext context.Context, state brick.LiveState) error {
	var actError error

	if state.Text == "" {
		actError = key.timer.StartTimer(pressContext, key.muralID, key.duration, key.sound)
	} else {
		actError = key.timer.PauseTimer(pressContext, key.muralID, state.Playing)
	}

	if actError != nil {
		return fmt.Errorf("using the board's timer: %w", brick.Categorised(timerFailureCategory(actError), actError))
	}

	select {
	case key.wake <- struct{}{}:
	default:
	}

	return nil
}

// countDown shows the timer just read, counting it down each second while it
// runs, until it is time to read again: after countdownRead, when it reaches
// nothing, or when a press wakes it. It reports false once the context ends.
func (key countdown) countDown(
	watchContext context.Context, timer murals.TimerState, show func(state brick.LiveState),
) bool {
	ticking := timer.Running && !timer.Paused

	for elapsed := time.Duration(0); elapsed < countdownRead; elapsed += countdownTick {
		left := timer.Remaining
		if ticking {
			left = max(timer.Remaining-elapsed, 0)
		}

		show(countdownState(timer, left))

		if ticking && left == 0 {
			return true
		}

		select {
		case <-watchContext.Done():
			return false
		case <-key.wake:
			return true
		case <-time.After(countdownTick):
		}
	}

	return true
}

// pause waits for a time, or until a press wakes the watch, and reports false
// once the context ends.
func (key countdown) pause(watchContext context.Context, length time.Duration) bool {
	select {
	case <-watchContext.Done():
		return false
	case <-key.wake:
		return true
	case <-time.After(length):
		return true
	}
}
