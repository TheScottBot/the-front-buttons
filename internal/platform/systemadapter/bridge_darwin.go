package systemadapter

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The macOS constructors give the purego adapter when asked for purego, and
// the cgo adapter otherwise. In a build without cgo, the cgo helpers give the
// purego adapters too (cgo_absent_darwin.go).

// NewApplicationIcons returns this machine's ApplicationIcons through the
// bridge.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewApplicationIcons(bridge Bridge) platform.ApplicationIcons {
	if bridge == BridgePurego {
		return newPureApplicationIcons()
	}

	return cgoApplicationIcons()
}

// NewApplicationActivator returns this machine's ApplicationActivator through
// the bridge.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewApplicationActivator(bridge Bridge) platform.ApplicationActivator {
	if bridge == BridgePurego {
		return newPureApplicationActivator()
	}

	return cgoApplicationActivator()
}

// NewApplicationBadges returns this machine's ApplicationBadges through the
// bridge.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewApplicationBadges(bridge Bridge) platform.ApplicationBadges {
	if bridge == BridgePurego {
		return newPureApplicationBadges()
	}

	return cgoApplicationBadges()
}

// NewWindowCycler returns this machine's WindowCycler through the bridge. A
// refusal calls ask, the key presser's RequestAccessibility, so both share one
// prompt and one restart.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewWindowCycler(ask func(), bridge Bridge) platform.WindowCycler {
	if bridge == BridgePurego {
		return newPureWindowCycler(ask)
	}

	return cgoWindowCycler(ask)
}

// NewKeyPresser returns this machine's key presser through the bridge. After
// a press is refused for want of Accessibility, it waits for the permission
// and calls granted once it arrives, so the daemon can restart itself to use
// it.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewKeyPresser(watchContext context.Context, granted func(), bridge Bridge) AskingKeyPresser {
	if bridge == BridgePurego {
		return newPureKeyPresser(watchContext, granted)
	}

	return cgoKeyPresser(watchContext, granted)
}

// AccessibilityTrusted reports, through the bridge, whether macOS trusts this
// process for Accessibility, as it decided when the process started.
func AccessibilityTrusted(bridge Bridge) bool {
	return accessibilityTrustedThrough(bridge)()
}

// accessibilityTrustedThrough is the bridge's Accessibility check.
func accessibilityTrustedThrough(bridge Bridge) func() bool {
	if bridge == BridgePurego {
		return pureAccessibilityTrusted
	}

	return cgoAccessibilityTrusted
}

// NewClipboardWriter returns this machine's ClipboardWriter through the
// bridge.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewClipboardWriter(bridge Bridge) platform.ClipboardWriter {
	if bridge == BridgePurego {
		return newPureClipboardWriter()
	}

	return cgoClipboardWriter()
}

// NewClipboardReader returns this machine's ClipboardReader through the
// bridge.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewClipboardReader(bridge Bridge) platform.ClipboardReader {
	if bridge == BridgePurego {
		return newPureClipboardWriter()
	}

	return cgoClipboardWriter()
}

// NewApplicationQuitter returns this machine's ApplicationQuitter through the
// bridge.
//
//nolint:ireturn // the bridge chooses the adapter behind the port
func NewApplicationQuitter(bridge Bridge) platform.ApplicationQuitter {
	if bridge == BridgePurego {
		return newPureApplicationQuitter()
	}

	return cgoApplicationQuitter()
}
