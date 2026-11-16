package systemadapter

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// UnavailableWindowCycler is the WindowCycler on Windows, which is parked
// (evaluation, deck testing): every cycle reports moving between windows
// unavailable, and the keys say so.
type UnavailableWindowCycler struct{}

// NewWindowCycler returns this machine's WindowCycler; there is no permission
// to ask for, and the bridge is ignored.
func NewWindowCycler(_ func(), _ Bridge) UnavailableWindowCycler { return UnavailableWindowCycler{} }

// CycleWindows reports moving between windows unavailable.
func (UnavailableWindowCycler) CycleWindows(context.Context, platform.WindowGroup, int) error {
	return platform.ErrWindowsUnavailable
}
