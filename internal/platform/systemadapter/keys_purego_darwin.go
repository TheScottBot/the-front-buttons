package systemadapter

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// pureKeyPresser posts key presses through purego. It is a stub until its
// adapter is written: every press reports the bridge unavailable and nothing
// is asked for. Choosing and writing shortcuts reach no system call, so they
// answer as the cgo adapter does.
type pureKeyPresser struct{}

// newPureKeyPresser returns the key presser through purego. It takes the cgo
// constructor's context and granted, which the adapter's Accessibility
// watcher uses.
func newPureKeyPresser(_ context.Context, _ func()) pureKeyPresser { return pureKeyPresser{} }

// pureAccessibilityTrusted reports Accessibility not trusted.
func pureAccessibilityTrusted() bool { return false }

// RequestAccessibility does nothing.
func (pureKeyPresser) RequestAccessibility() {}

// PressKey reports the bridge unavailable.
func (pureKeyPresser) PressKey(context.Context, string) error { return ErrBridgeUnavailable }

// PressShortcut reports the bridge unavailable.
func (pureKeyPresser) PressShortcut(context.Context, platform.Shortcut) error {
	return ErrBridgeUnavailable
}

// ForThisPlatform chooses the macOS half of a pair.
func (pureKeyPresser) ForThisPlatform(pair platform.ShortcutPair) platform.Shortcut { return pair.Mac }

// DescribeShortcut writes a shortcut as macOS's menus do.
func (pureKeyPresser) DescribeShortcut(shortcut platform.Shortcut) string {
	return MacOSShortcutText(shortcut)
}
