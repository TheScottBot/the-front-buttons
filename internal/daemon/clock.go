package daemon

import (
	"context"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/dashboard"
)

// The clocks on the deck: the dashboard's top row and its world clocks tick
// on each second, and only the figures that change are drawn again.

// clockTickEvent says a second has passed on the dashboard's clock.
type clockTickEvent struct{}

// startClock has the loop told of each second, on the second, until stopped.
func (state *loopState) startClock(runContext context.Context) chan struct{} {
	stop := make(chan struct{})

	go func() {
		for {
			now := time.Now()

			select {
			case <-time.After(now.Truncate(time.Second).Add(time.Second).Sub(now)):
				state.send(runContext, clockTickEvent{})
			case <-stop:
				return
			case <-runContext.Done():
				return
			}
		}
	}()

	return stop
}

// tickClock draws again the clock keys whose figures changed.
func (state *loopState) tickClock(runContext context.Context) {
	if !state.dashboard.open || state.connected == nil {
		return
	}

	scene, now := state.scene(), time.Now()

	for keyIndex, shown := range scene.Tiles() {
		if !dashboard.Ticks(shown) {
			continue
		}

		figure := scene.Figure(shown, now)
		if state.dashboard.drawn[keyIndex] == figure {
			continue
		}

		writeError := state.connected.SetKeyImage(keyIndex, scene.ClockImage(state.options.Renderer, shown, now))
		if writeError != nil {
			state.disconnect(runContext, "write failed")

			return
		}

		state.dashboard.drawn[keyIndex] = figure
	}
}
