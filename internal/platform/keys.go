package platform

import (
	"context"
	"errors"
)

// The ways a key press can fail that the person can act on, each shown on the
// key by its reason category.
var (
	// ErrKeyNotPermitted reports that this system will not let the daemon send
	// key presses: on macOS, Accessibility permission has not been granted.
	ErrKeyNotPermitted = errors.New("not permitted to send key presses")
	// ErrKeyUnsupported reports a key this system has no code for: macOS
	// defines no F21 to F24.
	ErrKeyUnsupported = errors.New("key not available on this system")
)

// The reason categories a key press failure is shown with.
const (
	keyNotPermittedCategory = "permission"
	keyUnsupportedCategory  = "unsupported_key"
	keyFailedCategory       = "key_failed"
)

// KeyFailureCategory names a key press error's reason category.
func KeyFailureCategory(pressError error) string {
	switch {
	case errors.Is(pressError, ErrKeyNotPermitted):
		return keyNotPermittedCategory
	case errors.Is(pressError, ErrKeyUnsupported):
		return keyUnsupportedCategory
	default:
		return keyFailedCategory
	}
}

// Shortcut is a key pressed with modifiers held, as ⌘C or Ctrl+C. Alt is
// Option on a Mac; Command is the Windows key on Windows.
type Shortcut struct {
	Key     string
	Control bool
	Alt     bool
	Shift   bool
	Command bool
}

// ShortcutPresser sends shortcuts to whatever has the keyboard focus, and
// writes them as this system does, for a key that shows its shortcut.
type ShortcutPresser interface {
	PressShortcut(pressContext context.Context, shortcut Shortcut) error
	DescribeShortcut(shortcut Shortcut) string
}
