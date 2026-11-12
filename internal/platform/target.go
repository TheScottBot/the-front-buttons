package platform

import (
	"errors"
	"fmt"
	"strings"
)

// MaximumLaunchTargetLength bounds a URL or application identifier. Browsers
// handle URLs of a few thousand characters, and Windows paths are far shorter; a
// longer target is a mistake or an attack, not a key a person set up.
const MaximumLaunchTargetLength = 4096

// The control character range, and the delete character, none of which belong in
// a URL or an identifier and some of which a shell or terminal would act on.
const (
	lastControlCharacter = 0x1F
	deleteCharacter      = 0x7F
)

// CheckLaunchTarget is the one rule every launch adapter applies before running
// anything (datum 0.7), so no adapter can be handed a target another would
// refuse. Adapters also pass the target as one argument with no shell, so this is
// the second guard, not the only one.
func CheckLaunchTarget(target string) error {
	switch {
	case target == "":
		return fmt.Errorf("%w: empty", ErrLaunchRefused)
	case len(target) > MaximumLaunchTargetLength:
		return fmt.Errorf("%w: %d bytes; the limit is %d", ErrLaunchRefused, len(target), MaximumLaunchTargetLength)
	case strings.HasPrefix(target, "-"):
		return fmt.Errorf("%w: starts with a hyphen, which a command would read as an option", ErrLaunchRefused)
	}

	for _, character := range target {
		if character <= lastControlCharacter || character == deleteCharacter {
			return fmt.Errorf("%w: holds control character U+%04X", ErrLaunchRefused, character)
		}
	}

	return nil
}

// The reason categories a launch failure shows on its key.
const (
	launchRefusedCategory = "refused"
	launchFailedCategory  = "open_failed"
)

// LaunchFailureCategory names a launch error's reason category: refused when the
// target was refused before anything ran, open failed otherwise.
func LaunchFailureCategory(launchError error) string {
	if errors.Is(launchError, ErrLaunchRefused) {
		return launchRefusedCategory
	}

	return launchFailedCategory
}
