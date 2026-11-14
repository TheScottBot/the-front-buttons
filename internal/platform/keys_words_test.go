package platform_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// TestShortcutWords_NamesKeysAsEachPlatformDoes: a shortcut is written as a
// person reads it on each platform, modifiers in that platform's order and
// punctuation keys as their marks.
func TestShortcutWords_NamesKeysAsEachPlatformDoes(t *testing.T) {
	t.Parallel()

	everything := platform.Shortcut{Key: "RightBracket", Control: true, Alt: true, Shift: true, Command: true}

	for _, want := range []struct {
		shortcut platform.Shortcut
		mac      bool
		words    string
	}{
		{everything, true, "Control + Option + Shift + Command + ]"},
		{everything, false, "Ctrl + Alt + Shift + Win + ]"},
		{platform.Shortcut{Key: "F9", Control: false, Alt: false, Shift: false, Command: true}, true, "Command + F9"},
		{platform.Shortcut{Key: "Grave", Control: true, Alt: false, Shift: false, Command: false}, false, "Ctrl + `"},
		{
			platform.Shortcut{Key: "Right", Control: false, Alt: false, Shift: false, Command: true},
			true,
			"Command + Right arrow",
		},
		{platform.Shortcut{Key: "Escape", Control: false, Alt: false, Shift: false, Command: false}, true, "Escape"},
	} {
		if words := platform.ShortcutWords(want.shortcut, want.mac); words != want.words {
			t.Errorf("%+v on the Mac %v: want %q, got %q", want.shortcut, want.mac, want.words, words)
		}
	}
}
