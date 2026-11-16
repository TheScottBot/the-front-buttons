package systemadapter

import "context"

// pureApplicationQuitter quits applications through purego. It is a stub
// until its adapter is written: every call reports the bridge unavailable.
type pureApplicationQuitter struct{}

// newPureApplicationQuitter returns the ApplicationQuitter through purego.
func newPureApplicationQuitter() pureApplicationQuitter { return pureApplicationQuitter{} }

// QuitApplication reports the bridge unavailable.
func (pureApplicationQuitter) QuitApplication(context.Context, string) error {
	return ErrBridgeUnavailable
}
