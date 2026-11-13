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

// frontButtonsPressKey posts one key down and one key up, as a keyboard would.
static int frontButtonsPressKey(CGKeyCode code) {
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

	if C.frontButtonsPressKey(C.CGKeyCode(code)) == 0 {
		return errKeyEventNotMade
	}

	return nil
}
