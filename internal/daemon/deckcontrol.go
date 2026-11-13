package daemon

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// deckControls is what the deck control keys have set: whether a brightness
// key has dimmed the deck and to what, and whether an off key has turned every
// other key off, at what brightness, and which key it was.
type deckControls struct {
	dimmed     bool
	dimPercent int

	off        bool
	offPercent int
	offPage    int
	offBrick   int
}

// brightness is the percentage the deck should show now: the off key's while
// off, else a brightness key's while dimmed, else the usual.
func (state *loopState) brightness() int {
	switch {
	case state.controls.off:
		return state.controls.offPercent
	case state.controls.dimmed:
		return state.controls.dimPercent
	default:
		return state.options.Brightness
	}
}

// applyBrightness sets the connected deck to the brightness in force. A failed
// write means the deck has gone.
func (state *loopState) applyBrightness(runContext context.Context) {
	if state.connected == nil {
		return
	}

	brightnessError := state.connected.SetBrightness(state.brightness())
	if brightnessError != nil {
		state.disconnect(runContext, "write failed")
	}
}

// pressControl carries out a deck control key's request in the loop, which
// owns the deck. Pressing a brightness key at the level it set goes back to
// the usual; at another level, goes to that one.
func (state *loopState) pressControl(
	runContext context.Context,
	pageIndex int,
	brickIndex int,
	request brick.DeckControlRequest,
) {
	switch request.Action {
	case brick.DeckControlBrightness:
		if state.controls.dimmed && state.controls.dimPercent == request.Percent {
			state.controls.dimmed = false
		} else {
			state.controls.dimmed, state.controls.dimPercent = true, request.Percent
		}

		state.applyBrightness(runContext)
	case brick.DeckControlOff:
		state.controls.off, state.controls.offPercent = true, request.Percent
		state.controls.offPage, state.controls.offBrick = pageIndex, brickIndex

		state.applyBrightness(runContext)
		state.repaintAll(runContext)
	}
}

// wake turns the keys back on at the brightness in force before.
func (state *loopState) wake(runContext context.Context) {
	state.controls.off = false

	state.applyBrightness(runContext)
	state.repaintAll(runContext)
}

// hiddenByOff reports whether the key at a position on the showing page is
// turned off: everything is while off, but the off key itself.
func (state *loopState) hiddenByOff(position layout.Position) bool {
	if !state.controls.off {
		return false
	}

	if state.paging.Current() != state.controls.offPage {
		return true
	}

	return state.pages[state.controls.offPage][state.controls.offBrick].key.Position != position
}

// fallbackLabel is what a key shows when it has no label of its own: a page
// marker its page's number, a brick that names its own its name (a shortcut
// key its shortcut), a brightness key its percentage, an off key the word,
// and anything else nothing.
func fallbackLabel(configured configuredKey) string {
	if _, isMarker := configured.instance.(brick.PageMarker); isMarker {
		return fmt.Sprintf("Page %d", configured.pageIndex+1)
	}

	if labelled, isLabelled := configured.instance.(brick.DefaultLabelled); isLabelled {
		return labelled.DefaultLabel()
	}

	control, isControl := configured.instance.(brick.DeckControl)
	if !isControl {
		return ""
	}

	request := control.DeckControl()
	if request.Action == brick.DeckControlBrightness {
		return fmt.Sprintf("%d%%", request.Percent)
	}

	return "Off"
}
