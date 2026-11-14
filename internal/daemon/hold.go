package daemon

import (
	"context"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// holdThreshold is how long an app key with its app screen ticked is held to
// show the screen as well as open the application: long enough that a tap is
// never taken for a hold, short enough not to feel like waiting.
const holdThreshold = 500 * time.Millisecond

// pendingHold is an app key pressed and not yet let go or held long enough:
// the key, under the configuration and deck it was pressed under, and the
// press it waits for.
type pendingHold struct {
	pageIndex               int
	brickIndex              int
	configurationGeneration int
	token                   int
}

// holdEvent says an app key has been held for holdThreshold.
type holdEvent struct {
	deckGeneration int
	keyIndex       int
	token          int
}

// holdState is the app keys held down now, by key index, and the last token
// given, so a hold's event is matched to the press that started it.
type holdState struct {
	pending map[int]pendingHold
	tokens  int
}

func newHoldState() holdState {
	return holdState{pending: map[int]pendingHold{}, tokens: 0}
}

// holdsForScreen reports whether a key on a page waits to see whether it is
// held: an app key whose application's screen it asks for and exists. Every
// other key acts the moment it is pressed.
func (state *loopState) holdsForScreen(pageIndex int, brickIndex int) bool {
	application, wanted := wantedScreenOf(state.pages[pageIndex][brickIndex])
	_, hasScreen := state.appScreens[application]

	return wanted && hasScreen && application != state.appScreen
}

// startHold notes an app key pressed down and has the loop told when it has
// been held long enough.
func (state *loopState) startHold(runContext context.Context, keyIndex int, pageIndex int, brickIndex int) {
	state.holds.tokens++
	token := state.holds.tokens
	state.holds.pending[keyIndex] = pendingHold{
		pageIndex: pageIndex, brickIndex: brickIndex,
		configurationGeneration: state.configurationGeneration, token: token,
	}
	deckGeneration := state.deckGeneration

	go func() {
		select {
		case <-time.After(holdThreshold):
			state.send(runContext, holdEvent{deckGeneration: deckGeneration, keyIndex: keyIndex, token: token})
		case <-runContext.Done():
		}
	}()
}

// releaseHold takes a held app key let go before the hold: a tap, which opens
// or raises the application alone. It reports whether the key was one.
func (state *loopState) releaseHold(runContext context.Context, change deck.KeyChange) bool {
	held, waiting := state.holds.pending[change.KeyIndex]
	if !waiting {
		return false
	}

	delete(state.holds.pending, change.KeyIndex)

	if held.configurationGeneration == state.configurationGeneration {
		state.press(runContext, held.pageIndex, held.brickIndex, false)
	}

	return true
}

// handleHold takes an app key held long enough: it opens or raises the
// application and shows its screen, and its release then does nothing.
func (state *loopState) handleHold(runContext context.Context, event holdEvent) {
	held, waiting := state.holds.pending[event.keyIndex]
	if !waiting || held.token != event.token || event.deckGeneration != state.deckGeneration {
		return
	}

	delete(state.holds.pending, event.keyIndex)

	if held.configurationGeneration == state.configurationGeneration {
		state.press(runContext, held.pageIndex, held.brickIndex, true)
	}
}

// pressDown takes a key going down: an app key that shows its screen when
// held waits to see which it is; any other key acts now.
func (state *loopState) pressDown(runContext context.Context, keyIndex int, slot layout.Slot) {
	pageIndex, showing := state.showingPage()
	if slot.Kind == layout.SlotBrick && showing && state.holdsForScreen(pageIndex, slot.BrickIndex) {
		state.startHold(runContext, keyIndex, pageIndex, slot.BrickIndex)

		return
	}

	state.pressSlot(runContext, slot)
}
