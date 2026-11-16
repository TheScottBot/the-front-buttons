package systemadapter

import (
	"context"
	"image"
)

// pureApplicationIcons reads applications' icons through purego. It is a
// stub until its adapter is written: every call reports the bridge
// unavailable.
type pureApplicationIcons struct{}

// newPureApplicationIcons returns the ApplicationIcons through purego.
func newPureApplicationIcons() pureApplicationIcons { return pureApplicationIcons{} }

// Icon reports the bridge unavailable.
func (pureApplicationIcons) Icon(context.Context, string) (image.Image, error) {
	return nil, ErrBridgeUnavailable
}
