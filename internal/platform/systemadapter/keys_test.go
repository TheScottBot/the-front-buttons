package systemadapter_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// TestMacOSKeyCode_MapsF13ToF20_AsTheSDKDefinesThem: the codes are kVK_F13 to
// kVK_F20 in the macOS SDK's HIToolbox Events.h, which defines no F21 to F24.
func TestMacOSKeyCode_MapsF13ToF20_AsTheSDKDefinesThem(t *testing.T) {
	t.Parallel()

	want := map[string]uint16{
		"F13": 0x69, "F14": 0x6B, "F15": 0x71, "F16": 0x6A,
		"F17": 0x40, "F18": 0x4F, "F19": 0x50, "F20": 0x5A,
	}

	for key, code := range want {
		got, mapError := systemadapter.MacOSKeyCode(key)
		if mapError != nil || got != code {
			t.Errorf("%s: want %#x, got %#x (%v)", key, code, got, mapError)
		}
	}

	for _, key := range []string{"F21", "F24", "F12", "A"} {
		_, mapError := systemadapter.MacOSKeyCode(key)
		if !errors.Is(mapError, platform.ErrKeyUnsupported) {
			t.Errorf("%s: want ErrKeyUnsupported, got %v", key, mapError)
		}
	}
}

// TestWindowsVirtualKey_MapsF13ToF24: VK_F13 to VK_F24 as golang.org/x/sys
// v0.49.0 defines them.
func TestWindowsVirtualKey_MapsF13ToF24(t *testing.T) {
	t.Parallel()

	for offset, key := range []string{"F13", "F14", "F15", "F16", "F17", "F18", "F19", "F20", "F21", "F22", "F23", "F24"} {
		got, mapError := systemadapter.WindowsVirtualKey(key)
		if want := uint16(0x7C + offset); mapError != nil || got != want {
			t.Errorf("%s: want %#x, got %#x (%v)", key, want, got, mapError)
		}
	}

	_, mapError := systemadapter.WindowsVirtualKey("F12")
	if !errors.Is(mapError, platform.ErrKeyUnsupported) {
		t.Errorf("F12: want ErrKeyUnsupported, got %v", mapError)
	}
}

// TestShortcutKeys_EveryKeyMapsOnBothPlatforms: each key the shortcut brick
// offers has a code on macOS and on Windows, from the SDK's Events.h and
// golang.org/x/sys v0.49.0, read on 13 November 2026.
func TestShortcutKeys_EveryKeyMapsOnBothPlatforms(t *testing.T) {
	t.Parallel()

	for _, key := range brick.FormatChoices(brick.FormatShortcutKey) {
		_, macError := systemadapter.MacOSShortcutKeyCode(key)
		_, windowsError := systemadapter.WindowsShortcutVirtualKey(key)

		if macError != nil || windowsError != nil {
			t.Errorf("%s: want a code on both platforms, got %v and %v", key, macError, windowsError)
		}
	}

	for key, want := range map[string][2]uint16{
		"C": {0x08, 0x43}, "V": {0x09, 0x56}, "5": {0x17, 0x35}, "F1": {0x7A, 0x70}, "Return": {0x24, 0x0D},
		"Left": {0x7B, 0x25}, "Grave": {0x32, 0xC0}, "Delete": {0x33, 0x08}, "ForwardDelete": {0x75, 0x2E},
	} {
		macCode, _ := systemadapter.MacOSShortcutKeyCode(key)
		windowsCode, _ := systemadapter.WindowsShortcutVirtualKey(key)

		if macCode != want[0] || windowsCode != want[1] {
			t.Errorf("%s: want macOS %#x and Windows %#x, got %#x and %#x", key, want[0], want[1], macCode, windowsCode)
		}
	}
}

// TestWindowsShortcutStrokes_HoldTheModifiersAroundTheKey: on Windows the
// modifiers go down in order, then the key down and up, then the modifiers up
// in reverse, and a navigation key is marked extended so it is not read as
// the number pad's.
func TestWindowsShortcutStrokes_HoldTheModifiersAroundTheKey(t *testing.T) {
	t.Parallel()

	strokes, strokeError := systemadapter.WindowsShortcutStrokes(platform.Shortcut{
		Key: "C", Control: true, Alt: false, Shift: true, Command: false,
	})
	if strokeError != nil {
		t.Fatal(strokeError)
	}

	want := []systemadapter.WindowsStroke{
		{VirtualKey: 0x11, Up: false, Extended: false},
		{VirtualKey: 0x10, Up: false, Extended: false},
		{VirtualKey: 0x43, Up: false, Extended: false},
		{VirtualKey: 0x43, Up: true, Extended: false},
		{VirtualKey: 0x10, Up: true, Extended: false},
		{VirtualKey: 0x11, Up: true, Extended: false},
	}

	if !slices.Equal(strokes, want) {
		t.Errorf("want %+v, got %+v", want, strokes)
	}

	arrow, _ := systemadapter.WindowsShortcutStrokes(platform.Shortcut{
		Key: "Left", Control: false, Alt: false, Shift: false, Command: true,
	})
	if len(arrow) != 4 || arrow[0].VirtualKey != 0x5B || !arrow[1].Extended || !arrow[2].Extended {
		t.Errorf("want the Windows key around an extended Left, got %+v", arrow)
	}
}

// TestShortcutText_NamesAShortcutAsEachPlatformDoes: a key without a label
// shows its shortcut the way the platform writes it.
func TestShortcutText_NamesAShortcutAsEachPlatformDoes(t *testing.T) {
	t.Parallel()

	shortcut := platform.Shortcut{Key: "C", Control: true, Alt: true, Shift: true, Command: true}

	if got := systemadapter.MacOSShortcutText(shortcut); got != "⌃⌥⇧⌘C" {
		t.Errorf("want ⌃⌥⇧⌘C on macOS, got %q", got)
	}

	if got := systemadapter.WindowsShortcutText(shortcut); got != "Ctrl+Alt+Shift+Win+C" {
		t.Errorf("want Ctrl+Alt+Shift+Win+C on Windows, got %q", got)
	}
}
