package systemadapter

import "context"

// pureApplicationActivator brings applications to the front through purego.
// It is a stub until its adapter is written: every call reports the bridge
// unavailable.
type pureApplicationActivator struct{}

// newPureApplicationActivator returns the ApplicationActivator through purego.
func newPureApplicationActivator() pureApplicationActivator { return pureApplicationActivator{} }

// ActivateApplication reports the bridge unavailable.
func (pureApplicationActivator) ActivateApplication(context.Context, string) error {
	return ErrBridgeUnavailable
}
