package systemadapter

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// pureWindowCycler moves between windows through purego. It is a stub until
// its adapter is written: every call reports the bridge unavailable, and
// nothing is asked for.
type pureWindowCycler struct{}

// newPureWindowCycler returns the WindowCycler through purego. It takes the
// cgo constructor's ask, which the adapter calls when refused Accessibility.
func newPureWindowCycler(_ func()) pureWindowCycler { return pureWindowCycler{} }

// CycleWindows reports the bridge unavailable.
func (pureWindowCycler) CycleWindows(context.Context, platform.WindowGroup, int) error {
	return ErrBridgeUnavailable
}
