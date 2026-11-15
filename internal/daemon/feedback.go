package daemon

import (
	"context"
	"image"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// What a key shows of being pressed: while it is down it is drawn pressed,
// the moment it goes down and whatever it does, so a press that changes
// nothing else is still seen; a key held towards a hold, an app key's screen
// or a page marker's dashboard, shows a bar filling towards it; and a key
// counting quick presses, as Triple press to close does, shows how far the
// count has gone until its next press is no longer awaited.

// The hold's bar is drawn from holdBarDelay after the key went down, so a
// quick tap never flashes one, and redrawn every holdBarStep until the hold.
const (
	holdBarDelay = 150 * time.Millisecond
	holdBarStep  = 50 * time.Millisecond
)

// holdTickEvent says it is time to redraw a held key's bar.
type holdTickEvent struct {
	deckGeneration int
	keyIndex       int
	token          int
}

// pressCountEvent says a key's press count may have lapsed, so it is redrawn.
type pressCountEvent struct {
	configurationGeneration int
	pageIndex               int
	brickIndex              int
}

// handleFeedback takes the events of holds and press counts.
func (state *loopState) handleFeedback(runContext context.Context, received any) {
	switch typed := received.(type) {
	case holdEvent:
		state.handleHold(runContext, typed)
	case holdTickEvent:
		state.handleHoldTick(runContext, typed)
	case pressCountEvent:
		state.handlePressCount(runContext, typed)
	}
}

// keyWentDown notes a key on the page going down and draws it pressed.
func (state *loopState) keyWentDown(runContext context.Context, keyIndex int) {
	state.held[keyIndex] = true
	state.repaintSlot(runContext, keyIndex)
}

// keyWentUp notes a key let go and draws it as it is, if it was drawn
// pressed.
func (state *loopState) keyWentUp(runContext context.Context, keyIndex int) {
	if !state.held[keyIndex] {
		return
	}

	delete(state.held, keyIndex)
	state.repaintSlot(runContext, keyIndex)
}

// repaintSlot draws one key of the page showing as it is now.
func (state *loopState) repaintSlot(runContext context.Context, keyIndex int) {
	if state.connected == nil || state.dashboard.open {
		return
	}

	slot, found := layout.SlotAt(state.slots, keyIndex)
	if !found {
		return
	}

	state.writeKey(runContext, keyIndex, state.imageFor(runContext, slot))
}

// writeKey sends a key of the page its picture, drawn pressed while the key
// is down. A failed write means the deck has gone, so the deck is dropped; it
// reports whether the write went.
func (state *loopState) writeKey(runContext context.Context, keyIndex int, keyImage image.Image) bool {
	if state.held[keyIndex] {
		keyImage = state.options.Renderer.Pressed(keyImage, state.holdFraction(keyIndex))
	}

	writeError := state.connected.SetKeyImage(keyIndex, keyImage)
	if writeError != nil {
		state.disconnect(runContext, "write failed")

		return false
	}

	return true
}

// holdFraction is how far a key held towards a hold has gone, from nothing to
// one, or nothing for a key not held towards one or not yet held long enough
// to show it.
func (state *loopState) holdFraction(keyIndex int) float64 {
	held, waiting := state.holds.pending[keyIndex]
	if !waiting {
		return 0
	}

	elapsed := time.Since(held.started)
	if elapsed < holdBarDelay {
		return 0
	}

	return min(float64(elapsed)/float64(holdThreshold), 1)
}

// handleHoldTick redraws a key still held towards its hold, once its bar
// shows.
func (state *loopState) handleHoldTick(runContext context.Context, event holdTickEvent) {
	held, waiting := state.holds.pending[event.keyIndex]
	if !waiting || held.token != event.token || event.deckGeneration != state.deckGeneration ||
		state.holdFraction(event.keyIndex) == 0 {
		return
	}

	state.repaintSlot(runContext, event.keyIndex)
}

// awaitPressCount has the loop told when a key's press count lapses, so the
// key is redrawn without it.
func (state *loopState) awaitPressCount(
	runContext context.Context, result pressResultEvent, counter brick.PressCounter,
) {
	_, until := counter.PressCount(time.Now())
	if until.IsZero() {
		return
	}

	go func() {
		select {
		case <-time.After(time.Until(until)):
			state.send(runContext, pressCountEvent{
				configurationGeneration: result.configurationGeneration,
				pageIndex:               result.pageIndex,
				brickIndex:              result.brickIndex,
			})
		case <-runContext.Done():
		}
	}()
}

// handlePressCount redraws a key whose press count may have lapsed, if it is
// showing.
func (state *loopState) handlePressCount(runContext context.Context, event pressCountEvent) {
	showingIndex, showing := state.showingPage()
	if event.configurationGeneration != state.configurationGeneration || !showing || event.pageIndex != showingIndex {
		return
	}

	configured := state.pages[event.pageIndex][event.brickIndex]
	state.repaintKey(runContext, configured.key.Position, configured)
}
