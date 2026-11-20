package mural

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
)

// The timer keys: each acts on the one board its link names, through Mural's
// API, signed in from the editor's credentials page. mural_start_timer starts
// the board's timer, five minutes unless given another time, with its sound
// at the end unless silent; mural_add_minute and mural_add_half_minute add a
// minute and thirty seconds to it; mural_stop_timer stops it; and
// mural_pause_timer pauses it while it runs, or resumes it while it is paused.

// The timer kinds' names in configuration.
const (
	StartKindName         = "mural_start_timer"
	AddMinuteKindName     = "mural_add_minute"
	AddHalfMinuteKindName = "mural_add_half_minute"
	StopKindName          = "mural_stop_timer"
	PauseKindName         = "mural_pause_timer"
)

// noTimerCategory is what the pause key shows when the board has no timer
// running to pause or resume.
const noTimerCategory = "no_timer"

// errNoTimer reports a board with no timer running.
var errNoTimer = errors.New("the board has no timer running")

// The timer's times: Mural's own starting five minutes, and what each add key
// adds.
const (
	defaultDuration   = 5 * time.Minute
	minuteKeyAdds     = time.Minute
	halfMinuteKeyAdds = 30 * time.Second
)

// StartParameters is the start key's one declaration of its parameters.
type StartParameters struct {
	brick.Appearance

	Link     string        `brick:"link,required" bounds:"1,2048"                                             format:"mural_board"                                             description:"The board's address, as its browser shows it; the timer started is that board's."`
	Duration time.Duration `brick:"duration"      bounds:"1s,90m"                                             description:"How long the timer runs; five minutes if left out."`
	Silent   bool          `brick:"silent"        description:"Start the timer without its sound at the end."`
}

// AddParameters is each add key's one declaration of its parameters.
type AddParameters struct {
	brick.Appearance

	Link string `brick:"link,required" bounds:"1,2048" format:"mural_board" description:"The board's address, as its browser shows it; the time is added to that board's timer."`
}

// BoardParameters is the stop and pause keys' one declaration of their
// parameters.
type BoardParameters struct {
	brick.Appearance

	Link string `brick:"link,required" bounds:"1,2048" format:"mural_board" description:"The board's address, as its browser shows it; the key acts on that board's timer."`
}

// registerTimers adds the timer kinds to a registry.
func registerTimers(registry *brick.Registry) error {
	startError := brick.Register(registry, StartKindName, func(
		parameters StartParameters, ports brick.Ports,
	) (any, error) {
		duration := parameters.Duration
		if duration == 0 {
			duration = defaultDuration
		}

		return newTimerKey(StartKindName, parameters.Link, ports, func(timerContext context.Context, timer murals.Timer,
			muralID string,
		) error {
			return timer.StartTimer(timerContext, muralID, duration, !parameters.Silent)
		}, "Start "+clockTime(duration))
	})
	if startError != nil {
		return startError //nolint:wrapcheck // Register already names the kind
	}

	for kindName, extra := range map[string]time.Duration{
		AddMinuteKindName: minuteKeyAdds, AddHalfMinuteKindName: halfMinuteKeyAdds,
	} {
		addError := brick.Register(registry, kindName, func(parameters AddParameters, ports brick.Ports) (any, error) {
			return newTimerKey(kindName, parameters.Link, ports, func(timerContext context.Context, timer murals.Timer,
				muralID string,
			) error {
				return timer.AddTime(timerContext, muralID, extra)
			}, "+"+clockTime(extra))
		})
		if addError != nil {
			return addError //nolint:wrapcheck // Register already names the kind
		}
	}

	stopError := brick.Register(registry, StopKindName, constructStop)
	if stopError != nil {
		return stopError //nolint:wrapcheck // Register already names the kind
	}

	pauseError := brick.Register(registry, PauseKindName, constructPause)
	if pauseError != nil {
		return pauseError //nolint:wrapcheck // Register already names the kind
	}

	return registerCountdown(registry)
}

// constructStop builds a stop key.
func constructStop(parameters BoardParameters, ports brick.Ports) (any, error) {
	return newTimerKey(StopKindName, parameters.Link, ports, func(timerContext context.Context, timer murals.Timer,
		muralID string,
	) error {
		return timer.StopTimer(timerContext, muralID)
	}, "Stop")
}

// constructPause builds a pause key.
func constructPause(parameters BoardParameters, ports brick.Ports) (any, error) {
	return newTimerKey(PauseKindName, parameters.Link, ports, pauseOrResume, "Pause")
}

// pauseOrResume reads the board's timer, then pauses it if it runs or resumes
// it if it is paused.
func pauseOrResume(timerContext context.Context, timer murals.Timer, muralID string) error {
	state, readError := timer.ReadTimer(timerContext, muralID)
	if readError != nil {
		return readError //nolint:wrapcheck // the key's press names what it was doing
	}

	if !state.Running {
		return errNoTimer
	}

	return timer.PauseTimer(timerContext, muralID, !state.Paused) //nolint:wrapcheck // as above
}

// clockTime writes a time as a clock does, minutes and seconds.
func clockTime(length time.Duration) string {
	seconds := int(length / time.Second)

	return fmt.Sprintf("%d:%02d", seconds/int(time.Minute/time.Second), seconds%int(time.Minute/time.Second))
}

// timerKey is one timer key: what it does to its board's timer, and what it
// says it does.
type timerKey struct {
	muralID string
	timer   murals.Timer
	act     func(timerContext context.Context, timer murals.Timer, muralID string) error
	label   string
}

// newTimerKey builds a timer key for a board, refusing without Mural's API.
func newTimerKey(
	kindName string, link string, ports brick.Ports,
	act func(timerContext context.Context, timer murals.Timer, muralID string) error, label string,
) (timerKey, error) {
	if ports.MuralTimer == nil {
		return timerKey{}, fmt.Errorf("%w: %s needs Mural's API", brick.ErrPortMissing, kindName)
	}

	// The link was checked as a board's address when the configuration was
	// read, so it always names one.
	muralID, _ := murals.BoardID(link)

	return timerKey{muralID: muralID, timer: ports.MuralTimer, act: act, label: label}, nil
}

// Press does what the key does to its board's timer, and shows why on the key
// when Mural would not.
func (key timerKey) Press(pressContext context.Context) error {
	actError := key.act(pressContext, key.timer, key.muralID)
	if actError != nil {
		return fmt.Errorf("using the board's timer: %w", brick.Categorised(timerFailureCategory(actError), actError))
	}

	return nil
}

// DefaultLabel says what a press does when the key has no label of its own.
func (key timerKey) DefaultLabel() string { return key.label }

// timerFailureCategory is the reason a timer key shows for each way Mural
// fails.
func timerFailureCategory(actError error) string {
	for failure, category := range map[error]string{
		murals.ErrNotSetUp:     "not_set_up",
		murals.ErrNotConnected: "not_connected",
		murals.ErrNotFound:     "not_found",
		murals.ErrRefused:      "refused",
		errNoTimer:             noTimerCategory,
	} {
		if errors.Is(actError, failure) {
			return category
		}
	}

	return "network"
}
