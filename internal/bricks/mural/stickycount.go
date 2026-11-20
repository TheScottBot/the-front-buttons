package mural

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The sticky count key, mural_sticky_count, shows how many sticky notes are
// on a board, or in one area of it named by its title, such as a parking
// lot. A press counts again at once.

// StickyCountKindName is the sticky count key's name in configuration.
const StickyCountKindName = "mural_sticky_count"

// stickyCountIdentity names the live source the keys counting one board and
// area share.
const stickyCountIdentity = "mural_sticky_count:"

// stickyCountRead is how often the key counts again. Counting reads every
// sticky note, a page at a time, so it is read less often than the timer.
const stickyCountRead = 30 * time.Second

// stickyIcon is the sticky count key's icon.
const stickyIcon = "note"

// noAreaCategory is what a key shows when the board has no area of the title
// it names.
const noAreaCategory = "no_area"

// StickyCountParameters is the sticky count key's one declaration of its
// parameters.
type StickyCountParameters struct {
	brick.Appearance

	Link string `brick:"link,required" bounds:"1,2048" format:"mural_board"                                                                                                description:"The board's address, as its browser shows it; the key counts that board's sticky notes."`
	Area string `brick:"area"          bounds:"0,200"  description:"The title of the area to count in, as the board shows it, ignoring case; the whole board if left out."`
}

// registerStickyCount adds the sticky count kind to a registry.
func registerStickyCount(registry *brick.Registry, shared *wakes) error {
	return brick.Register(registry, StickyCountKindName, func( //nolint:wrapcheck // Register already names the kind
		parameters StickyCountParameters, ports brick.Ports,
	) (any, error) {
		if ports.MuralBoard == nil {
			return nil, fmt.Errorf("%w: %s needs Mural's API", brick.ErrPortMissing, StickyCountKindName)
		}

		// The link was checked as a board's address when the configuration
		// was read, so it always names one.
		muralID, _ := murals.BoardID(parameters.Link)
		area := strings.TrimSpace(parameters.Area)
		identity := stickyCountIdentity + muralID + ":" + strings.ToLower(area)

		return stickyCount{
			muralID: muralID, area: area, identity: identity, board: ports.MuralBoard,
			wake: shared.forSource(identity),
		}, nil
	})
}

// stickyCount is one sticky count key.
type stickyCount struct {
	muralID  string
	area     string
	identity string
	board    murals.Board
	wake     chan struct{}
}

// LiveIdentity is shared by every key counting the same board and area.
func (key stickyCount) LiveIdentity() string { return key.identity }

// DefaultLabel names the key when it has no label of its own.
func (stickyCount) DefaultLabel() string { return "Stickies" }

// Watch counts the sticky notes and publishes the count when it changes,
// until the context ends. The area is found by its title once, and again
// after a read fails. A press wakes it to count at once.
func (key stickyCount) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	var last brick.LiveState

	areaID := ""

	for first := true; ; first = false {
		state := brick.LiveState{
			Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: nil, Playing: false, Text: "",
			Urgent: false,
		}

		count, countError := key.count(watchContext, &areaID)
		if watchContext.Err() != nil {
			return
		}

		wait := stickyCountRead

		if countError != nil {
			state.ErrorReason, areaID, wait = muralFailureCategory(countError), "", countdownRetry
		} else {
			state.Text = stickiesText(count)
		}

		if first || state != last {
			publish(state)

			last = state
		}

		select {
		case <-watchContext.Done():
			return
		case <-key.wake:
		case <-time.After(wait):
		}
	}
}

// DrawLive shows the count with a note icon.
func (stickyCount) DrawLive(state brick.LiveState, appearance *render.Appearance) {
	if state.Text == "" {
		return
	}

	appearance.Label = state.Text
	appearance.Icon = stickyIcon
}

// PressWithState has the watch count again at once.
func (key stickyCount) PressWithState(_ context.Context, _ brick.LiveState) error {
	wakeUp(key.wake)

	return nil
}

// count counts the sticky notes on the board, or in the key's area, finding
// the area first when it has not been found yet.
func (key stickyCount) count(countContext context.Context, areaID *string) (int, error) {
	if key.area != "" && *areaID == "" {
		found, findError := key.board.FindArea(countContext, key.muralID, key.area)
		if findError != nil {
			return 0, findError //nolint:wrapcheck // the watch shows its category
		}

		*areaID = found
	}

	return key.board.CountStickyNotes(countContext, key.muralID, *areaID) //nolint:wrapcheck // as above
}

// stickiesText writes a count of sticky notes.
func stickiesText(count int) string {
	if count == 1 {
		return "1 sticky"
	}

	return fmt.Sprintf("%d stickies", count)
}
