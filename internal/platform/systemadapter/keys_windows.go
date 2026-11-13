package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The SendInput constants from WinUser.h: a keyboard input, and the flag that
// makes it a key up.
const (
	inputKeyboard = 1
	keyEventKeyUp = 0x0002
	// keyEventExtendedKey is KEYEVENTF_EXTENDEDKEY: a navigation key, kept
	// apart from the number pad's.
	keyEventExtendedKey = 0x0001
	inputStructSize     = 40
)

// keyboardInput is Win32's INPUT holding a KEYBDINPUT, laid out for 64 bit
// Windows: a type, padding, the KEYBDINPUT, and padding to the size of the
// union's largest member, MOUSEINPUT. 40 bytes in all, checked below.
type keyboardInput struct {
	inputType  uint32
	_          uint32
	virtualKey uint16
	scanCode   uint16
	flags      uint32
	time       uint32
	_          uint32
	extraInfo  uintptr
	_          [8]byte
}

//nolint:gochecknoglobals // the one lazily loaded system call
var procSendInput = windows.NewLazySystemDLL("user32.dll").NewProc("SendInput")

// errKeyNotSent reports that Windows inserted fewer key events than given,
// which it does when the focused window runs with higher integrity.
var errKeyNotSent = errors.New("key press not sent by Windows")

// WindowsKeyPresser is the KeyPresser adapter on Windows: SendInput with a key
// down and a key up. Not yet run on a Windows machine: the author is testing on
// macOS only for now.
type WindowsKeyPresser struct{}

// AccessibilityTrusted is always true on Windows, which asks no permission to
// send key presses.
func AccessibilityTrusted() bool { return true }

// NewKeyPresser returns this machine's KeyPresser. Windows needs no permission,
// so nothing is ever waited for.
func NewKeyPresser(_ context.Context, _ func()) WindowsKeyPresser {
	// The build fails unless keyboardInput is exactly the 40 bytes SendInput
	// expects for each input: the index is a constant, out of range otherwise.
	var probe keyboardInput

	_ = [1]struct{}{}[unsafe.Sizeof(probe)-inputStructSize]

	return WindowsKeyPresser{}
}

// PressKey sends one momentary press of a function key.
func (WindowsKeyPresser) PressKey(_ context.Context, key string) error {
	virtualKey, mapError := WindowsVirtualKey(key)
	if mapError != nil {
		return mapError
	}

	inputs := [2]keyboardInput{
		{inputType: inputKeyboard, virtualKey: virtualKey, scanCode: 0, flags: 0, time: 0, extraInfo: 0},
		{inputType: inputKeyboard, virtualKey: virtualKey, scanCode: 0, flags: keyEventKeyUp, time: 0, extraInfo: 0},
	}

	// SendInput takes a pointer to the array; unsafe is the only way to pass
	// one to a system call, and the array outlives the call.
	sent, _, callError := procSendInput.Call(uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])), inputStructSize) //nolint:gosec // see above
	if sent != uintptr(len(inputs)) {
		return fmt.Errorf("%w: %w", errKeyNotSent, callError)
	}

	return nil
}

// PressShortcut sends one momentary press of a key with its modifiers held:
// each modifier down, the key down and up, then the modifiers up.
func (WindowsKeyPresser) PressShortcut(_ context.Context, shortcut platform.Shortcut) error {
	strokes, strokeError := WindowsShortcutStrokes(shortcut)
	if strokeError != nil {
		return strokeError
	}

	inputs := make([]keyboardInput, 0, len(strokes))

	for _, stroke := range strokes {
		var flags uint32
		if stroke.Up {
			flags |= keyEventKeyUp
		}

		if stroke.Extended {
			flags |= keyEventExtendedKey
		}

		inputs = append(inputs, keyboardInput{
			inputType: inputKeyboard, virtualKey: stroke.VirtualKey, scanCode: 0, flags: flags, time: 0, extraInfo: 0,
		})
	}

	// SendInput takes a pointer to the slice's array; unsafe is the only way
	// to pass one to a system call, and the slice outlives the call.
	sent, _, callError := procSendInput.Call(uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])), inputStructSize) //nolint:gosec // see above
	if sent != uintptr(len(inputs)) {
		return fmt.Errorf("%w: %w", errKeyNotSent, callError)
	}

	return nil
}

// DescribeShortcut writes a shortcut as Windows does.
func (WindowsKeyPresser) DescribeShortcut(shortcut platform.Shortcut) string {
	return WindowsShortcutText(shortcut)
}

// RequestAccessibility does nothing: Windows needs no permission to send keys.
func (WindowsKeyPresser) RequestAccessibility() {}
