package systemadapter

import "strings"

// QuitTarget is how the macOS quitter finds an application among those
// running: by bundle identifier, or else by name.
type QuitTarget struct {
	Name     string
	ByBundle bool
}

// MacOSQuitTarget reads an identifier as the launcher reads it: a bundle
// identifier by bundle, or else a name, Signal and Signal.app alike, as the
// open command takes it.
func MacOSQuitTarget(identifier string) QuitTarget {
	if isMacOSBundleIdentifier(identifier) {
		return QuitTarget{Name: identifier, ByBundle: true}
	}

	return QuitTarget{Name: strings.TrimSuffix(identifier, macOSApplicationSuffix), ByBundle: false}
}
