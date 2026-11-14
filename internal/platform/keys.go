package platform

import (
	"context"
	"errors"
	"strings"
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

// ShortcutPair is one action's shortcut on each platform, as an IDE's default
// keymaps give it: ⌘F9 on macOS, Ctrl+F9 on Windows.
type ShortcutPair struct {
	Mac     Shortcut
	Windows Shortcut
}

// ShortcutPresser sends shortcuts to whatever has the keyboard focus, writes
// them as this system does, for a key that shows its shortcut, and chooses
// this system's half of a pair.
type ShortcutPresser interface {
	PressShortcut(pressContext context.Context, shortcut Shortcut) error
	DescribeShortcut(shortcut Shortcut) string
	ForThisPlatform(pair ShortcutPair) Shortcut
}

// keyMarks are the keys written as the mark on them, or more plainly than
// their names, when a shortcut is written out.
//
//nolint:gochecknoglobals // a fixed table
var keyMarks = map[string]string{
	"LeftBracket": "[", "RightBracket": "]", "Backslash": "\\", "Semicolon": ";", "Quote": "'",
	"Comma": ",", "Period": ".", "Slash": "/", "Grave": "`", "Minus": "-", "Equals": "=",
	"Left": "Left arrow", "Right": "Right arrow", "Up": "Up arrow", "Down": "Down arrow",
	"ForwardDelete": "Forward Delete", "PageUp": "Page Up", "PageDown": "Page Down",
}

// ShortcutWords writes a shortcut as a person reads it on the Mac or on
// Windows: its modifiers in that platform's order and by its names, then its
// key, joined by plus signs, such as Command + Shift + ].
func ShortcutWords(shortcut Shortcut, mac bool) string {
	names := [4]string{"Ctrl", "Alt", "Shift", "Win"}
	if mac {
		names = [4]string{"Control", "Option", "Shift", "Command"}
	}

	var words []string

	for index, held := range []bool{shortcut.Control, shortcut.Alt, shortcut.Shift, shortcut.Command} {
		if held {
			words = append(words, names[index])
		}
	}

	key := shortcut.Key
	if mark, found := keyMarks[key]; found {
		key = mark
	}

	return strings.Join(append(words, key), " + ")
}
