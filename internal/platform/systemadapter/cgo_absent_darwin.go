//go:build darwin && !cgo

package systemadapter

import "context"

// In a build without cgo the cgo files are left out, and each cgo helper the
// constructors call gives the purego adapter in its place, whatever bridge was
// asked for. CgoAvailable is false in this build, so the daemon can log it.

func cgoApplicationIcons() pureApplicationIcons { return newPureApplicationIcons() }

func cgoApplicationActivator() pureApplicationActivator { return newPureApplicationActivator() }

func cgoApplicationBadges() pureApplicationBadges { return newPureApplicationBadges() }

func cgoWindowCycler(ask func()) pureWindowCycler { return newPureWindowCycler(ask) }

func cgoKeyPresser(watchContext context.Context, granted func()) pureKeyPresser {
	return newPureKeyPresser(watchContext, granted)
}

// cgoAccessibilityTrusted is the purego check itself, so the bridge's choice
// of check can be compared in tests.
//
//nolint:gochecknoglobals // a function chosen at build time
var cgoAccessibilityTrusted = pureAccessibilityTrusted

func cgoClipboardWriter() pureClipboardWriter { return newPureClipboardWriter() }

func cgoApplicationQuitter() pureApplicationQuitter { return newPureApplicationQuitter() }
