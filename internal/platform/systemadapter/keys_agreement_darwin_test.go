//nolint:testpackage // the key presser's binding, events and prompt options are unexported
package systemadapter

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The CGEvent types of a key's two events, from CGEventTypes.h.
const (
	eventTypeKeyDown = 10
	eventTypeKeyUp   = 11
)

// keyTestFunctions read back what the adapter made, without posting it.
type keyTestFunctions struct {
	retainCount     func(ref objcbridge.CFTypeRef) int
	eventFlags      func(event objcbridge.CFTypeRef) uint64
	eventType       func(event objcbridge.CFTypeRef) uint32
	eventField      func(event objcbridge.CFTypeRef, field uint32) int64
	dictionaryCount func(dictionary objcbridge.CFTypeRef) int
	dictionaryValue func(dictionary, key objcbridge.CFTypeRef) objcbridge.CFTypeRef
}

func loadKeyTestFunctions(t *testing.T) keyTestFunctions {
	t.Helper()

	var functions keyTestFunctions

	_, bindError := objcbridge.Bind(func() (struct{}, error) {
		coreFoundation, cfError := objcbridge.Framework(objcbridge.PathCoreFoundation)
		services, servicesError := objcbridge.Framework(objcbridge.PathApplicationServices)

		loadError := errors.Join(cfError, servicesError)
		if loadError != nil {
			return struct{}{}, loadError
		}

		objcbridge.Function(&functions.retainCount, coreFoundation, "CFGetRetainCount")
		objcbridge.Function(&functions.dictionaryCount, coreFoundation, "CFDictionaryGetCount")
		objcbridge.Function(&functions.dictionaryValue, coreFoundation, "CFDictionaryGetValue")
		objcbridge.Function(&functions.eventFlags, services, "CGEventGetFlags")
		objcbridge.Function(&functions.eventType, services, "CGEventGetType")
		objcbridge.Function(&functions.eventField, services, "CGEventGetIntegerValueField")

		return struct{}{}, nil
	})()
	if bindError != nil {
		t.Fatalf("binding the test's own functions: %v", bindError)
	}

	return functions
}

func boundKeyFunctions(t *testing.T) *keyFunctions {
	t.Helper()

	functions, bindError := keyBinding()
	if bindError != nil {
		t.Fatalf("binding the key presser: %v", bindError)
	}

	return functions
}

// postedEvent is what the test read of one event handed to post.
type postedEvent struct {
	eventType uint32
	keyCode   int64
	flags     uint64
}

func TestMacOSEventFlags_HoldEachModifierAsCoreGraphicsNamesIt(t *testing.T) {
	t.Parallel()

	for _, example := range []struct {
		shortcut platform.Shortcut
		want     uint64
	}{
		{platform.Shortcut{Key: "F13", Control: false, Alt: false, Shift: false, Command: false}, 0},
		{platform.Shortcut{Key: "A", Control: true, Alt: false, Shift: false, Command: false}, 0x40000},
		{platform.Shortcut{Key: "A", Control: false, Alt: true, Shift: false, Command: false}, 0x80000},
		{platform.Shortcut{Key: "A", Control: false, Alt: false, Shift: true, Command: false}, 0x20000},
		{platform.Shortcut{Key: "A", Control: false, Alt: false, Shift: false, Command: true}, 0x100000},
		{platform.Shortcut{Key: "A", Control: true, Alt: true, Shift: true, Command: true}, 0x1E0000},
	} {
		if got := macOSEventFlags(example.shortcut); got != example.want {
			t.Errorf("%s: want flags %#x, got %#x", MacOSShortcutText(example.shortcut), example.want, got)
		}
	}
}

// TestPureKeys_MakesADownAndAnUp_WithTheFlags_AndReleasesBoth hands the
// events to a recorder in place of CGEventPost, so nothing is posted.
func TestPureKeys_MakesADownAndAnUp_WithTheFlags_AndReleasesBoth(t *testing.T) {
	t.Parallel()

	reader := loadKeyTestFunctions(t)
	functions := boundKeyFunctions(t)

	var (
		posted []postedEvent
		held   []objcbridge.CFTypeRef
	)

	const flags = 0x120000

	postError := pressKeyEvents(functions, macOSKeyF19, flags, func(event objcbridge.CFTypeRef) {
		held = append(held, objcbridge.Retain(event))
		posted = append(posted, postedEvent{
			eventType: reader.eventType(event),
			keyCode:   reader.eventField(event, keyboardEventKeycodeField),
			flags:     reader.eventFlags(event),
		})
	})
	if postError != nil {
		t.Fatal(postError)
	}

	want := []postedEvent{
		{eventType: eventTypeKeyDown, keyCode: macOSKeyF19, flags: flags},
		{eventType: eventTypeKeyUp, keyCode: macOSKeyF19, flags: flags},
	}
	if len(posted) != len(want) || posted[0] != want[0] || posted[1] != want[1] {
		t.Errorf("want a key down then a key up %v, got %v", want, posted)
	}

	for index, event := range held {
		if count := reader.retainCount(event); count != 1 {
			t.Errorf("event %d: want only the test's own reference left, got a retain count of %d", index, count)
		}

		objcbridge.Release(event)
	}
}

// keyboardEventKeycodeField is kCGKeyboardEventKeycode, from CGEventTypes.h.
const keyboardEventKeycodeField = 9

// TestPureKeys_AsksWithThePromptOption_AndReleasesTheOptions hands the
// options to a recorder in place of AXIsProcessTrustedWithOptions, so no
// prompt is shown.
func TestPureKeys_AsksWithThePromptOption_AndReleasesTheOptions(t *testing.T) {
	t.Parallel()

	reader := loadKeyTestFunctions(t)
	functions := boundKeyFunctions(t)

	trueValue, booleanError := objcbridge.CFBoolean(true)
	if booleanError != nil {
		t.Fatal(booleanError)
	}

	var options objcbridge.CFTypeRef

	trusted := askTrust(functions, func(given objcbridge.CFTypeRef) bool {
		options = objcbridge.Retain(given)

		return true
	})
	if !trusted || options == 0 {
		t.Fatalf("want the check's answer and the options it was given, got %v and %#x", trusted, options)
	}

	defer objcbridge.Release(options)

	if objcbridge.GoString(functions.promptOption) != "AXTrustedCheckOptionPrompt" {
		t.Errorf("want the key kAXTrustedCheckOptionPrompt, got %q", objcbridge.GoString(functions.promptOption))
	}

	count, value := reader.dictionaryCount(options), reader.dictionaryValue(options, functions.promptOption)
	if count != 1 || value != trueValue {
		t.Errorf("want only the prompt option, set to true, got %d entries and %#x", count, value)
	}

	if count := reader.retainCount(options); count != 1 {
		t.Errorf("want only the test's own reference to the options left, got a retain count of %d", count)
	}
}

func TestPureKeys_BindingAMissingName_ReportsTheBridgeUnavailable_WithoutAsking(t *testing.T) {
	t.Parallel()

	for field, rename := range map[string]func(*keyFunctionNames){
		"trusted":              func(names *keyFunctionNames) { names.trusted = "AXIsProcessTrustd" },
		"trusted with options": func(names *keyFunctionNames) { names.trustedWithOptions = "AXIsProcessTrustedWithOption" },
		"prompt option":        func(names *keyFunctionNames) { names.promptOption = "kAXTrustedCheckOptionPrompts" },
		"create":               func(names *keyFunctionNames) { names.createKeyboardEvent = "CGEventCreateKeyEvent" },
		"flags":                func(names *keyFunctionNames) { names.setFlags = "CGEventSetFlag" },
		"post":                 func(names *keyFunctionNames) { names.post = "CGEventPosts" },
	} {
		names := systemKeyFunctionNames()
		rename(&names)

		presser := newPureKeyPresser(t.Context(), func() { t.Errorf("%s: want no grant from a failed binding", field) })
		presser.binding = objcbridge.Bind(func() (*keyFunctions, error) { return bindKeys(names) })

		// The binding fails before any system call, so this neither asks nor
		// posts.
		presser.RequestAccessibility()

		shortcut := platform.Shortcut{Key: "F9", Control: false, Alt: false, Shift: true, Command: true}
		for name, pressError := range map[string]error{
			"PressKey":      presser.PressKey(t.Context(), "F13"),
			"PressShortcut": presser.PressShortcut(t.Context(), shortcut),
		} {
			if !errors.Is(pressError, ErrBridgeUnavailable) || !errors.Is(pressError, objcbridge.ErrBindingFailed) {
				t.Errorf("%s, %s: want ErrBridgeUnavailable for a failed binding, got %v", field, name, pressError)
			}
		}

		if trustedThrough(presser.binding) {
			t.Errorf("%s: want a failed binding to report Accessibility not trusted", field)
		}
	}
}

func TestPureKeys_RefusesAnUnknownKey_BeforeAskingAnything(t *testing.T) {
	t.Parallel()

	presser := newPureKeyPresser(t.Context(), func() {})
	shortcut := platform.Shortcut{Key: "F24", Control: false, Alt: false, Shift: false, Command: false}

	for name, pressError := range map[string]error{
		"PressKey":      presser.PressKey(t.Context(), "F24"),
		"PressShortcut": presser.PressShortcut(t.Context(), shortcut),
	} {
		if !errors.Is(pressError, platform.ErrKeyUnsupported) {
			t.Errorf("%s: want ErrKeyUnsupported, got %v", name, pressError)
		}
	}
}

func TestKeysAgreement_AccessibilityTrust_IsTheSameThroughBothBridges(t *testing.T) {
	t.Parallel()

	cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) bool {
		t.Helper()
		skipCgoWithoutIt(t, bridge)

		return AccessibilityTrusted(bridge)
	})

	if CgoAvailable && cgo != purego {
		t.Errorf("want the same trust through each bridge, got cgo %v and purego %v", cgo, purego)
	}

	t.Logf("trusted for Accessibility: %v", purego)
}

func TestKeysAgreement_ChoosesAndWritesShortcuts_TheSameThroughBothBridges(t *testing.T) {
	t.Parallel()

	shortcut := platform.Shortcut{Key: "F9", Control: true, Alt: true, Shift: true, Command: true}
	pair := platform.ShortcutPair{
		Mac:     shortcut,
		Windows: platform.Shortcut{Key: "F9", Control: true, Alt: false, Shift: false, Command: false},
	}

	type answer struct {
		chosen  platform.Shortcut
		written string
	}

	cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) answer {
		t.Helper()
		skipCgoWithoutIt(t, bridge)

		presser := NewKeyPresser(t.Context(), func() {}, bridge)

		return answer{chosen: presser.ForThisPlatform(pair), written: presser.DescribeShortcut(shortcut)}
	})

	if purego != (answer{chosen: shortcut, written: "⌃⌥⇧⌘F9"}) || (CgoAvailable && cgo != purego) {
		t.Errorf("want the macOS half written ⌃⌥⇧⌘F9 through each bridge, got cgo %v and purego %v", cgo, purego)
	}
}

// TestKeysAgreement_EachBridgePostsF20 presses F20, then Shift F20, through
// each bridge, when this process is trusted for Accessibility; untrusted, a
// press would show the prompt, so the test skips. It acts on the desktop:
// whatever the person has bound to F20 or Shift F20 runs twice each.
//
//nolint:paralleltest // posts keys to the shared desktop
func TestKeysAgreement_EachBridgePostsF20(t *testing.T) {
	actingAllowed(t)

	if !AccessibilityTrusted(BridgePurego) {
		t.Skip("not trusted for Accessibility; pressing would show the prompt")
	}

	shortcut := platform.Shortcut{Key: "F20", Control: false, Alt: false, Shift: true, Command: false}

	cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) error {
		t.Helper()
		skipCgoWithoutIt(t, bridge)

		presser := NewKeyPresser(t.Context(), func() {}, bridge)

		return errors.Join(presser.PressKey(t.Context(), "F20"), presser.PressShortcut(t.Context(), shortcut))
	})

	if purego != nil || (CgoAvailable && cgo != nil) {
		t.Errorf("want F20 and Shift F20 posted through each bridge, got cgo %v and purego %v", cgo, purego)
	}
}
