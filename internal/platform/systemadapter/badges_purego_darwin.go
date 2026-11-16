package systemadapter

import "context"

// pureApplicationBadges reads the Dock's badges through purego. It is a stub
// until its adapter is written: every call reports the bridge unavailable.
type pureApplicationBadges struct{}

// newPureApplicationBadges returns the ApplicationBadges through purego.
func newPureApplicationBadges() pureApplicationBadges { return pureApplicationBadges{} }

// Badges reports the bridge unavailable.
func (pureApplicationBadges) Badges(context.Context) (map[string]string, error) {
	return nil, ErrBridgeUnavailable
}
