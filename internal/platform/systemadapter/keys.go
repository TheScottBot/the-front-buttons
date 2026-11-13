package systemadapter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// kVK_F13 to kVK_F20 from the macOS SDK's HIToolbox Events.h, read on 12
// November 2026. macOS defines no F21 to F24.
const (
	macOSKeyF13 = 0x69
	macOSKeyF14 = 0x6B
	macOSKeyF15 = 0x71
	macOSKeyF16 = 0x6A
	macOSKeyF17 = 0x40
	macOSKeyF18 = 0x4F
	macOSKeyF19 = 0x50
	macOSKeyF20 = 0x5A
)

// macOSKeyCodes maps each function key macOS has to its virtual key code.
//
//nolint:gochecknoglobals // a fixed table
var macOSKeyCodes = map[string]uint16{
	"F13": macOSKeyF13, "F14": macOSKeyF14, "F15": macOSKeyF15, "F16": macOSKeyF16,
	"F17": macOSKeyF17, "F18": macOSKeyF18, "F19": macOSKeyF19, "F20": macOSKeyF20,
}

// The Windows virtual key codes run VK_F13 to VK_F24 in order, 0x7C to 0x87,
// as golang.org/x/sys v0.49.0 defines them.
const (
	windowsVirtualKeyF13 = 0x7C
	firstExtendedKey     = 13
	lastExtendedKey      = 24
)

// MacOSKeyCode returns a function key's macOS virtual key code.
func MacOSKeyCode(key string) (uint16, error) {
	code, known := macOSKeyCodes[key]
	if !known {
		return 0, fmt.Errorf("%w: %s on macOS", platform.ErrKeyUnsupported, key)
	}

	return code, nil
}

// WindowsVirtualKey returns a function key's Windows virtual key code.
func WindowsVirtualKey(key string) (uint16, error) {
	var number int

	_, scanError := fmt.Sscanf(key, "F%d", &number)
	if scanError != nil || number < firstExtendedKey || number > lastExtendedKey || key != fmt.Sprintf("F%d", number) {
		return 0, fmt.Errorf("%w: %s on Windows", platform.ErrKeyUnsupported, key)
	}

	return uint16(windowsVirtualKeyF13 + number - firstExtendedKey), nil
}

// macOSShortcutKeyCodes are the macOS virtual key codes of every key a
// shortcut may name, from the SDK's HIToolbox Events.h (kVK_ANSI_A and the
// rest), read on 13 November 2026.
//
//nolint:mnd // the SDK's codes, copied as a table; naming each would only repeat it
func macOSShortcutKeyCodes() map[string]uint16 {
	return map[string]uint16{
		"A": 0x00, "S": 0x01, "D": 0x02, "F": 0x03, "H": 0x04, "G": 0x05, "Z": 0x06, "X": 0x07, "C": 0x08,
		"V": 0x09, "B": 0x0B, "Q": 0x0C, "W": 0x0D, "E": 0x0E, "R": 0x0F, "Y": 0x10, "T": 0x11, "O": 0x1F,
		"U": 0x20, "I": 0x22, "P": 0x23, "L": 0x25, "J": 0x26, "K": 0x28, "N": 0x2D, "M": 0x2E,
		"1": 0x12, "2": 0x13, "3": 0x14, "4": 0x15, "6": 0x16, "5": 0x17, "9": 0x19, "7": 0x1A, "8": 0x1C,
		"0":  0x1D,
		"F1": 0x7A, "F2": 0x78, "F3": 0x63, "F4": 0x76, "F5": 0x60, "F6": 0x61, "F7": 0x62, "F8": 0x64,
		"F9": 0x65, "F10": 0x6D, "F11": 0x67, "F12": 0x6F, "F13": macOSKeyF13, "F14": macOSKeyF14,
		"F15": macOSKeyF15, "F16": macOSKeyF16, "F17": macOSKeyF17, "F18": macOSKeyF18, "F19": macOSKeyF19,
		"F20":    macOSKeyF20,
		"Return": 0x24, "Tab": 0x30, "Space": 0x31, "Escape": 0x35, "Delete": 0x33, "ForwardDelete": 0x75,
		"Left": 0x7B, "Right": 0x7C, "Down": 0x7D, "Up": 0x7E, "Home": 0x73, "End": 0x77, "PageUp": 0x74,
		"PageDown": 0x79,
		"Equals":   0x18, "Minus": 0x1B, "RightBracket": 0x1E, "LeftBracket": 0x21, "Quote": 0x27,
		"Semicolon": 0x29, "Backslash": 0x2A, "Comma": 0x2B, "Slash": 0x2C, "Period": 0x2F, "Grave": 0x32,
	}
}

// The Windows virtual key codes a shortcut uses, from golang.org/x/sys
// v0.49.0: letters and digits are their ASCII codes, F1 to F20 run on from
// VK_F1, and the modifiers are VK_CONTROL, VK_MENU, VK_SHIFT and VK_LWIN.
const (
	windowsVirtualKeyF1      = 0x70
	windowsVirtualKeyControl = 0x11
	windowsVirtualKeyAlt     = 0x12
	windowsVirtualKeyShift   = 0x10
	windowsVirtualKeyWindows = 0x5B
)

// windowsNamedKeys are the Windows virtual key codes of the named keys, and
// whether each is an extended key, one of the navigation keys a keyboard
// keeps apart from the number pad.
//
//nolint:mnd // x/sys's codes, copied as a table; naming each would only repeat it
func windowsNamedKeys() map[string]WindowsStroke {
	plain := func(code uint16) WindowsStroke { return WindowsStroke{VirtualKey: code, Up: false, Extended: false} }
	extended := func(code uint16) WindowsStroke { return WindowsStroke{VirtualKey: code, Up: false, Extended: true} }

	return map[string]WindowsStroke{
		"Return": plain(0x0D), "Tab": plain(0x09), "Space": plain(0x20), "Escape": plain(0x1B),
		"Delete": plain(0x08), "ForwardDelete": extended(0x2E),
		"Left": extended(0x25), "Up": extended(0x26), "Right": extended(0x27), "Down": extended(0x28),
		"Home": extended(0x24), "End": extended(0x23), "PageUp": extended(0x21), "PageDown": extended(0x22),
		"Minus": plain(0xBD), "Equals": plain(0xBB), "LeftBracket": plain(0xDB), "RightBracket": plain(0xDD),
		"Backslash": plain(0xDC), "Semicolon": plain(0xBA), "Quote": plain(0xDE), "Comma": plain(0xBC),
		"Period": plain(0xBE), "Slash": plain(0xBF), "Grave": plain(0xC0),
	}
}

// WindowsStroke is one key going down or up in a Windows shortcut.
type WindowsStroke struct {
	VirtualKey uint16
	Up         bool
	Extended   bool
}

// MacOSShortcutKeyCode returns a shortcut key's macOS virtual key code.
func MacOSShortcutKeyCode(key string) (uint16, error) {
	code, known := macOSShortcutKeyCodes()[key]
	if !known {
		return 0, fmt.Errorf("%w: %s on macOS", platform.ErrKeyUnsupported, key)
	}

	return code, nil
}

// WindowsShortcutVirtualKey returns a shortcut key's Windows virtual key code.
func WindowsShortcutVirtualKey(key string) (uint16, error) {
	stroke, strokeError := windowsKeyStroke(key)

	return stroke.VirtualKey, strokeError
}

func windowsKeyStroke(key string) (WindowsStroke, error) {
	if named, isNamed := windowsNamedKeys()[key]; isNamed {
		return named, nil
	}

	if isLetterOrDigit(key) {
		return WindowsStroke{VirtualKey: uint16(key[0]), Up: false, Extended: false}, nil
	}

	var number int

	_, scanError := fmt.Sscanf(key, "F%d", &number)
	if scanError == nil && number >= 1 && number <= lastExtendedKey && key == fmt.Sprintf("F%d", number) {
		return WindowsStroke{VirtualKey: uint16(windowsVirtualKeyF1 + number - 1), Up: false, Extended: false}, nil
	}

	return WindowsStroke{VirtualKey: 0, Up: false, Extended: false},
		fmt.Errorf("%w: %s on Windows", platform.ErrKeyUnsupported, key)
}

// isLetterOrDigit reports one capital letter or digit, whose Windows virtual
// key code is its ASCII code.
func isLetterOrDigit(key string) bool {
	return len(key) == 1 && ((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= '0' && key[0] <= '9'))
}

// strokesPerKey is a key's two strokes, down and up.
const strokesPerKey = 2

// WindowsShortcutStrokes is a shortcut as Windows input: each modifier down
// in order, the key down and up, then the modifiers up in reverse.
func WindowsShortcutStrokes(shortcut platform.Shortcut) ([]WindowsStroke, error) {
	key, keyError := windowsKeyStroke(shortcut.Key)
	if keyError != nil {
		return nil, keyError
	}

	var modifiers []uint16

	for _, held := range []struct {
		on   bool
		code uint16
	}{
		{shortcut.Control, windowsVirtualKeyControl},
		{shortcut.Alt, windowsVirtualKeyAlt},
		{shortcut.Shift, windowsVirtualKeyShift},
		{shortcut.Command, windowsVirtualKeyWindows},
	} {
		if held.on {
			modifiers = append(modifiers, held.code)
		}
	}

	strokes := make([]WindowsStroke, 0, strokesPerKey*(len(modifiers)+1))
	for _, code := range modifiers {
		strokes = append(strokes, WindowsStroke{VirtualKey: code, Up: false, Extended: false})
	}

	keyUp := key
	keyUp.Up = true
	strokes = append(strokes, key, keyUp)

	for _, code := range slices.Backward(modifiers) {
		strokes = append(strokes, WindowsStroke{VirtualKey: code, Up: true, Extended: false})
	}

	return strokes, nil
}

// MacOSShortcutText writes a shortcut as macOS's menus do: ⌃⌥⇧⌘ then the key.
func MacOSShortcutText(shortcut platform.Shortcut) string {
	return shortcutText(shortcut, [4]string{"⌃", "⌥", "⇧", "⌘"}, "")
}

// WindowsShortcutText writes a shortcut as Windows does: Ctrl+Alt+Shift+Win
// then the key.
func WindowsShortcutText(shortcut platform.Shortcut) string {
	return shortcutText(shortcut, [4]string{"Ctrl", "Alt", "Shift", "Win"}, "+")
}

func shortcutText(shortcut platform.Shortcut, names [4]string, joiner string) string {
	var parts []string

	for index, held := range []bool{shortcut.Control, shortcut.Alt, shortcut.Shift, shortcut.Command} {
		if held {
			parts = append(parts, names[index])
		}
	}

	return strings.Join(append(parts, shortcut.Key), joiner)
}
