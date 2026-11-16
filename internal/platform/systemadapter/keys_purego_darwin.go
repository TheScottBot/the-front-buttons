package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// pureAccessibilityAskInterval is how often the watcher asks while it waits,
// as the cgo adapter's accessibilityAskInterval is.
const pureAccessibilityAskInterval = 2 * time.Second

// The modifier flags and the event tap, from CGEventTypes.h and IOLLEvent.h.
const (
	eventFlagShift     uint64 = 0x00020000 // kCGEventFlagMaskShift
	eventFlagControl   uint64 = 0x00040000 // kCGEventFlagMaskControl
	eventFlagAlternate uint64 = 0x00080000 // kCGEventFlagMaskAlternate
	eventFlagCommand   uint64 = 0x00100000 // kCGEventFlagMaskCommand
	hidEventTap        uint32 = 0          // kCGHIDEventTap
)

// errPureKeyEventNotMade reports that macOS would not create the key events,
// as the cgo adapter's errKeyEventNotMade does.
var errPureKeyEventNotMade = errors.New("macOS did not create the key events")

// keyFunctionNames are the C functions and constant the key presser binds,
// so a test can bind a name the system does not have.
type keyFunctionNames struct {
	trusted             string
	trustedWithOptions  string
	promptOption        string
	createKeyboardEvent string
	setFlags            string
	post                string
}

// systemKeyFunctionNames are ApplicationServices' own names.
func systemKeyFunctionNames() keyFunctionNames {
	return keyFunctionNames{
		trusted:             "AXIsProcessTrusted",
		trustedWithOptions:  "AXIsProcessTrustedWithOptions",
		promptOption:        "kAXTrustedCheckOptionPrompt",
		createKeyboardEvent: "CGEventCreateKeyboardEvent",
		setFlags:            "CGEventSetFlags",
		post:                "CGEventPost",
	}
}

// keyFunctions are the C functions and values the key presser calls, bound
// once.
type keyFunctions struct {
	trusted             func() bool
	trustedWithOptions  func(options objcbridge.CFTypeRef) bool
	promptOption        objcbridge.CFTypeRef
	trueValue           objcbridge.CFTypeRef
	createKeyboardEvent func(source uintptr, code uint16, down bool) objcbridge.CFTypeRef
	setFlags            func(event objcbridge.CFTypeRef, flags uint64)
	post                func(tap uint32, event objcbridge.CFTypeRef)
}

//nolint:gochecknoglobals // bound once, on first use
var keyBinding = objcbridge.Bind(func() (*keyFunctions, error) { return bindKeys(systemKeyFunctionNames()) })

// bindKeys loads ApplicationServices, which holds the Accessibility and
// CoreGraphics event functions, and binds each one. A missing function
// panics inside Function, which Bind recovers.
func bindKeys(names keyFunctionNames) (*keyFunctions, error) {
	services, loadError := objcbridge.Framework(objcbridge.PathApplicationServices)
	if loadError != nil {
		return nil, fmt.Errorf("loading ApplicationServices: %w", loadError)
	}

	var functions keyFunctions

	objcbridge.Function(&functions.trusted, services, names.trusted)
	objcbridge.Function(&functions.trustedWithOptions, services, names.trustedWithOptions)
	objcbridge.Function(&functions.createKeyboardEvent, services, names.createKeyboardEvent)
	objcbridge.Function(&functions.setFlags, services, names.setFlags)
	objcbridge.Function(&functions.post, services, names.post)

	promptOption, promptError := objcbridge.Constant(services, names.promptOption)
	trueValue, trueError := objcbridge.CFBoolean(true)

	constantError := errors.Join(promptError, trueError)
	if constantError != nil {
		return nil, constantError
	}

	functions.promptOption, functions.trueValue = promptOption, trueValue

	return &functions, nil
}

// pureKeyPresser posts key presses through purego: CoreGraphics key events
// posted at the HID level, as a keyboard's own are. macOS only lets a process
// trusted for Accessibility do that; the first press without permission has
// macOS show its prompt, once per run.
type pureKeyPresser struct {
	binding func() (*keyFunctions, error)
	prompt  *sync.Once
	watcher *AccessibilityWatcher
}

// newPureKeyPresser returns the key presser through purego. After a press is
// refused for want of Accessibility, it waits for the permission and calls
// granted once it arrives, so the daemon can restart itself to use it.
func newPureKeyPresser(watchContext context.Context, granted func()) pureKeyPresser {
	return pureKeyPresser{
		binding: keyBinding,
		prompt:  &sync.Once{},
		watcher: NewAccessibilityWatcher(watchContext, pureFreshProcessTrusted, pureAccessibilityAskInterval, granted),
	}
}

// pureAccessibilityTrusted reports whether macOS trusts this process for
// Accessibility, as it decided when the process started. Functions that
// could not be bound report it not trusted.
func pureAccessibilityTrusted() bool {
	return trustedThrough(keyBinding)
}

// trustedThrough asks AXIsProcessTrusted with the functions binding gives.
func trustedThrough(binding func() (*keyFunctions, error)) bool {
	functions, bindError := binding()

	return bindError == nil && functions.trusted()
}

// pureFreshProcessTrusted asks a new copy of this binary, which macOS judges
// afresh, whether it is trusted, as the cgo adapter's freshProcessTrusted
// does.
func pureFreshProcessTrusted(checkContext context.Context) bool {
	binary, binaryError := os.Executable()
	if binaryError != nil {
		return false
	}

	// This binary, with one fixed flag and no shell.
	return exec.CommandContext(checkContext, binary, "-"+AccessibilityCheckFlag).Run() == nil //nolint:gosec // see above
}

// askTrust asks check whether this process is trusted, with options that
// have macOS show its Accessibility prompt when it is not, and releases the
// options.
func askTrust(functions *keyFunctions, check func(options objcbridge.CFTypeRef) bool) bool {
	options := objcbridge.CFDictionary(map[objcbridge.CFTypeRef]objcbridge.CFTypeRef{
		functions.promptOption: functions.trueValue,
	})
	defer objcbridge.Release(options)

	return check(options)
}

// RequestAccessibility has macOS show its Accessibility prompt, once per run,
// and waits for the permission, so the daemon restarts itself to use it. A
// refused key press calls it, and so does a refused window key. Functions
// that could not be bound can neither ask nor be granted, so it does nothing.
func (presser pureKeyPresser) RequestAccessibility() {
	functions, bindError := presser.binding()
	if bindError != nil {
		return
	}

	presser.prompt.Do(func() { askTrust(functions, functions.trustedWithOptions) })
	presser.watcher.Start()
}

// PressKey sends one momentary press of a function key.
func (presser pureKeyPresser) PressKey(_ context.Context, key string) error {
	code, mapError := MacOSKeyCode(key)
	if mapError != nil {
		return mapError
	}

	return presser.press(code, platform.Shortcut{Key: key, Control: false, Alt: false, Shift: false, Command: false})
}

// PressShortcut sends one momentary press of a key with its modifiers held.
func (presser pureKeyPresser) PressShortcut(_ context.Context, shortcut platform.Shortcut) error {
	code, mapError := MacOSShortcutKeyCode(shortcut.Key)
	if mapError != nil {
		return mapError
	}

	return presser.press(code, shortcut)
}

// ForThisPlatform chooses the macOS half of a pair.
func (pureKeyPresser) ForThisPlatform(pair platform.ShortcutPair) platform.Shortcut { return pair.Mac }

// DescribeShortcut writes a shortcut as macOS's menus do.
func (pureKeyPresser) DescribeShortcut(shortcut platform.Shortcut) string {
	return MacOSShortcutText(shortcut)
}

// press posts the key with the shortcut's modifiers held, once this process
// is trusted for Accessibility; untrusted, it asks for the permission.
func (presser pureKeyPresser) press(code uint16, shortcut platform.Shortcut) error {
	functions, bindError := presser.binding()
	if bindError != nil {
		return fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
	}

	if !functions.trusted() {
		presser.RequestAccessibility()

		return fmt.Errorf("%w: grant The Front Buttons Accessibility in System Settings, Privacy and Security, "+
			"and it restarts itself to use it", platform.ErrKeyNotPermitted)
	}

	return pressKeyEvents(functions, code, macOSEventFlags(shortcut), func(event objcbridge.CFTypeRef) {
		functions.post(hidEventTap, event)
	})
}

// macOSEventFlags are the CoreGraphics flags that hold the shortcut's
// modifiers.
func macOSEventFlags(shortcut platform.Shortcut) uint64 {
	var flags uint64

	for _, modifier := range []struct {
		held bool
		flag uint64
	}{
		{shortcut.Control, eventFlagControl},
		{shortcut.Alt, eventFlagAlternate},
		{shortcut.Shift, eventFlagShift},
		{shortcut.Command, eventFlagCommand},
	} {
		if modifier.held {
			flags |= modifier.flag
		}
	}

	return flags
}

// pressKeyEvents makes one key down and one key up, as a keyboard would, with
// the flags set on both, hands each to post, down first, and releases both.
func pressKeyEvents(functions *keyFunctions, code uint16, flags uint64, post func(event objcbridge.CFTypeRef)) error {
	keyDown := functions.createKeyboardEvent(0, code, true)
	keyUp := functions.createKeyboardEvent(0, code, false)

	defer objcbridge.Release(keyDown)
	defer objcbridge.Release(keyUp)

	if keyDown == 0 || keyUp == 0 {
		return errPureKeyEventNotMade
	}

	functions.setFlags(keyDown, flags)
	functions.setFlags(keyUp, flags)
	post(keyDown)
	post(keyUp)

	return nil
}
