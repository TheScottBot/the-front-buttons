package mural

import (
	"context"
	"fmt"
	"image/color"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The private mode key, mural_private_mode, shows whether private mode is on
// for a board, lit while it is, and a press turns it on or off. Private mode
// hides everyone's work from everyone else until it ends, for a silent
// brainstorm.

// PrivateModeKindName is the private mode key's name in configuration.
const PrivateModeKindName = "mural_private_mode"

// privateModeIdentity names the live source a board's private mode keys
// share.
const privateModeIdentity = "mural_private_mode:"

// privateModeRead is how often the key reads the board, to see private mode
// turned on or off on the board itself.
const privateModeRead = 10 * time.Second

// The private mode key's icons: a closed eye while it is on, an open one
// while it is off.
const (
	privateOnIcon  = "eye-off"
	privateOffIcon = "eye"
)

// privateOnBackground lights the key while private mode is on.
//
//nolint:gochecknoglobals // a colour, never changed
var privateOnBackground = color.RGBA{R: 0x5E, G: 0x35, B: 0xB1, A: 0xFF} //nolint:mnd // a colour's own channels

// PrivateModeParameters is the private mode key's one declaration of its
// parameters.
type PrivateModeParameters struct {
	brick.Appearance

	Link        string `brick:"link,required" bounds:"1,2048"                                             format:"mural_board" description:"The board's address, as its browser shows it; the key shows and runs that board's private mode."`
	HideAuthors bool   `brick:"hide_authors"  description:"Hide who wrote what while private mode is on."`
}

// registerPrivateMode adds the private mode kind to a registry.
func registerPrivateMode(registry *brick.Registry, shared *wakes) error {
	return brick.Register(registry, PrivateModeKindName, func( //nolint:wrapcheck // Register already names the kind
		parameters PrivateModeParameters, ports brick.Ports,
	) (any, error) {
		if ports.MuralBoard == nil {
			return nil, fmt.Errorf("%w: %s needs Mural's API", brick.ErrPortMissing, PrivateModeKindName)
		}

		// The link was checked as a board's address when the configuration
		// was read, so it always names one.
		muralID, _ := murals.BoardID(parameters.Link)

		return privateMode{
			muralID: muralID, board: ports.MuralBoard, hideAuthors: parameters.HideAuthors,
			wake: shared.forSource(privateModeIdentity + muralID),
		}, nil
	})
}

// privateMode is one private mode key.
type privateMode struct {
	muralID     string
	board       murals.Board
	hideAuthors bool
	wake        chan struct{}
}

// LiveIdentity is shared by every private mode key for the board.
func (key privateMode) LiveIdentity() string { return privateModeIdentity + key.muralID }

// DefaultLabel names the key when it has no label of its own.
func (privateMode) DefaultLabel() string { return "Private" }

// Watch reads whether private mode is on and publishes it when it changes,
// until the context ends. A press wakes it to read at once.
func (key privateMode) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	var last brick.LiveState

	for first := true; ; first = false {
		active, readError := key.board.ReadPrivateMode(watchContext, key.muralID)
		if watchContext.Err() != nil {
			return
		}

		state := brick.LiveState{
			Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: nil, Playing: active, Text: "",
			Urgent: false,
		}

		wait := privateModeRead

		if readError != nil {
			state.ErrorReason, state.Playing = muralFailureCategory(readError), false
			wait = countdownRetry
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

// DrawLive shows a closed eye on a lit key while private mode is on, and an
// open eye on the key's own background while it is off.
func (privateMode) DrawLive(state brick.LiveState, appearance *render.Appearance) {
	appearance.Icon = privateOffIcon

	if state.Playing {
		appearance.Icon = privateOnIcon
		appearance.Background = privateOnBackground
	}
}

// PressWithState starts private mode while it is off and ends it while it is
// on, then has the watch read the board at once.
func (key privateMode) PressWithState(pressContext context.Context, state brick.LiveState) error {
	var actError error

	if state.Playing {
		actError = key.board.EndPrivateMode(pressContext, key.muralID)
	} else {
		actError = key.board.StartPrivateMode(pressContext, key.muralID, key.hideAuthors)
	}

	if actError != nil {
		return fmt.Errorf("using the board's private mode: %w",
			brick.Categorised(muralFailureCategory(actError), actError))
	}

	wakeUp(key.wake)

	return nil
}
