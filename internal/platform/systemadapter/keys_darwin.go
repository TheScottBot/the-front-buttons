package systemadapter

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>

// frontButtonsTrusted reports whether this process may send input events, and,
// when asked, has macOS show its Accessibility permission prompt.
static int frontButtonsTrusted(int prompt) {
	if (!prompt) {
		return AXIsProcessTrusted();
	}

	const void *keys[] = { kAXTrustedCheckOptionPrompt };
	const void *values[] = { kCFBooleanTrue };
	CFDictionaryRef options = CFDictionaryCreate(NULL, keys, values, 1,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	Boolean trusted = AXIsProcessTrustedWithOptions(options);
	CFRelease(options);

	return trusted;
}

// frontButtonsPressKey posts one key down and one key up, as a keyboard would,
// with the modifiers asked for held: each is 1 to hold it.
static int frontButtonsPressKey(CGKeyCode code, int control, int option, int shift, int command) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, code, true);
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, code, false);

	if (down == NULL || up == NULL) {
		if (down != NULL) {
			CFRelease(down);
		}
		if (up != NULL) {
			CFRelease(up);
		}

		return 0;
	}

	CGEventFlags flags = 0;
	if (control) {
		flags |= kCGEventFlagMaskControl;
	}
	if (option) {
		flags |= kCGEventFlagMaskAlternate;
	}
	if (shift) {
		flags |= kCGEventFlagMaskShift;
	}
	if (command) {
		flags |= kCGEventFlagMaskCommand;
	}

	CGEventSetFlags(down, flags);
	CGEventSetFlags(up, flags);
	CGEventPost(kCGHIDEventTap, down);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(down);
	CFRelease(up);

	return 1;
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// accessibilityAskInterval is how often the watcher asks while it waits.
const accessibilityAskInterval = 2 * time.Second

// errKeyEventNotMade reports that macOS would not create the key events.
var errKeyEventNotMade = errors.New("macOS did not create the key events")

// MacOSKeyPresser is the KeyPresser adapter on macOS: CoreGraphics key events
// posted at the HID level, as a keyboard's own are. macOS only lets a process
// trusted for Accessibility do that; the first press without permission has
// macOS show its prompt, once per run.
type MacOSKeyPresser struct {
	prompt  *sync.Once
	watcher *AccessibilityWatcher
}

// NewKeyPresser returns this machine's KeyPresser. After a press is refused
// for want of Accessibility, it waits for the permission and calls granted
// once it arrives, so the daemon can restart itself to use it.
func NewKeyPresser(watchContext context.Context, granted func()) MacOSKeyPresser {
	return MacOSKeyPresser{
		prompt:  &sync.Once{},
		watcher: NewAccessibilityWatcher(watchContext, freshProcessTrusted, accessibilityAskInterval, granted),
	}
}

// AccessibilityTrusted reports whether macOS trusts this process for
// Accessibility, as it decided when the process started.
func AccessibilityTrusted() bool {
	return C.frontButtonsTrusted(0) != 0
}

// freshProcessTrusted asks a new copy of this binary, which macOS judges
// afresh, whether it is trusted.
func freshProcessTrusted(checkContext context.Context) bool {
	binary, binaryError := os.Executable()
	if binaryError != nil {
		return false
	}

	// This binary, with one fixed flag and no shell.
	return exec.CommandContext(checkContext, binary, "-"+AccessibilityCheckFlag).Run() == nil
}

// RequestAccessibility has macOS show its Accessibility prompt, once per run,
// and waits for the permission, so the daemon restarts itself to use it. A
// refused key press calls it, and so does a refused window key.
func (presser MacOSKeyPresser) RequestAccessibility() {
	presser.prompt.Do(func() { C.frontButtonsTrusted(1) })
	presser.watcher.Start()
}

// PressKey sends one momentary press of a function key.
func (presser MacOSKeyPresser) PressKey(_ context.Context, key string) error {
	code, mapError := MacOSKeyCode(key)
	if mapError != nil {
		return mapError
	}

	if !AccessibilityTrusted() {
		presser.RequestAccessibility()

		return fmt.Errorf("%w: grant The Front Buttons Accessibility in System Settings, Privacy and Security, "+
			"and it restarts itself to use it", platform.ErrKeyNotPermitted)
	}

	return presser.post(code, platform.Shortcut{Key: key, Control: false, Alt: false, Shift: false, Command: false})
}

// PressShortcut sends one momentary press of a key with its modifiers held.
func (presser MacOSKeyPresser) PressShortcut(_ context.Context, shortcut platform.Shortcut) error {
	code, mapError := MacOSShortcutKeyCode(shortcut.Key)
	if mapError != nil {
		return mapError
	}

	if !AccessibilityTrusted() {
		presser.RequestAccessibility()

		return fmt.Errorf("%w: grant The Front Buttons Accessibility in System Settings, Privacy and Security, "+
			"and it restarts itself to use it", platform.ErrKeyNotPermitted)
	}

	return presser.post(code, shortcut)
}

// ForThisPlatform chooses the macOS half of a pair.
func (MacOSKeyPresser) ForThisPlatform(pair platform.ShortcutPair) platform.Shortcut { return pair.Mac }

// DescribeShortcut writes a shortcut as macOS's menus do.
func (MacOSKeyPresser) DescribeShortcut(shortcut platform.Shortcut) string {
	return MacOSShortcutText(shortcut)
}

// post sends the key with the shortcut's modifiers held.
func (MacOSKeyPresser) post(code uint16, shortcut platform.Shortcut) error {
	held := func(on bool) C.int {
		if on {
			return 1
		}

		return 0
	}

	posted := C.frontButtonsPressKey(C.CGKeyCode(code), held(shortcut.Control), held(shortcut.Alt),
		held(shortcut.Shift), held(shortcut.Command))
	if posted == 0 {
		return errKeyEventNotMade
	}

	return nil
}
