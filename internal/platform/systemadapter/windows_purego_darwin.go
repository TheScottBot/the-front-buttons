package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The Accessibility names the window cycler reads and sets. They are CFSTR
// macros in the headers, not exported symbols, so they are spelt here.
const (
	attributeWindows       = "AXWindows"
	attributeSubrole       = "AXSubrole"
	attributeTitle         = "AXTitle"
	attributeFocusedWindow = "AXFocusedWindow"
	attributeMinimised     = "AXMinimized"
	attributeMain          = "AXMain"
	attributeFrontmost     = "AXFrontmost"
	actionRaise            = "AXRaise"
	standardWindowSubrole  = "AXStandardWindow"
)

// activationPolicyRegular is NSApplicationActivationPolicyRegular: an
// application in the Dock with windows of its own.
const activationPolicyRegular = 0

// errPureWindowNotRaised reports a window macOS would not raise.
var errPureWindowNotRaised = errors.New("macOS did not raise the window")

// pureWindowCycler is the WindowCycler through purego: the same decisions,
// with windows read and raised through Accessibility by calls purego makes.
type pureWindowCycler struct{ WindowCycler }

// newPureWindowCycler returns the WindowCycler through purego. A refusal
// calls ask, the key presser's RequestAccessibility, so both share one prompt
// and one restart.
func newPureWindowCycler(ask func()) pureWindowCycler {
	lister := pureWindowLister{calls: realWindowCalls()}

	return pureWindowCycler{NewWindowCyclerWith(pureAccessibilityTrusted, lister.list, ask)}
}

// windowBindings are the Objective-C objects and selectors the lister sends,
// and the Core Foundation booleans Raise sets, found once.
type windowBindings struct {
	workspace            objc.ID
	frontmostApplication objc.SEL
	processIdentifier    objc.SEL
	activationPolicy     objc.SEL
	localizedName        objc.SEL
	bundleIdentifier     objc.SEL
	booleanFalse         objcbridge.CFTypeRef
	booleanTrue          objcbridge.CFTypeRef
}

// pureWindowBindings finds the bindings once. A framework, class, selector
// or constant missing from this Mac, or a panic while finding them, makes
// every call report ErrBridgeUnavailable rather than stop the daemon.
//
//nolint:gochecknoglobals // the bindings are found once, on first use
var pureWindowBindings = objcbridge.Bind(bindWindows)

func bindWindows() (*windowBindings, error) {
	// The pool's first use loads every framework objcbridge needs, AppKit
	// among them, so a failure to load panics here, where Bind recovers it.
	objcbridge.WithAutoreleasePool(func() {})

	workspaceClass, workspaceError := objcbridge.Class("NSWorkspace")
	applicationClass, applicationError := objcbridge.Class("NSRunningApplication")
	shared, sharedError := objcbridge.ClassSelector(workspaceClass, "sharedWorkspace")
	frontmost, frontmostError := objcbridge.InstanceSelector(workspaceClass, "frontmostApplication")
	processIdentifier, processError := objcbridge.InstanceSelector(applicationClass, "processIdentifier")
	activationPolicy, policyError := objcbridge.InstanceSelector(applicationClass, "activationPolicy")
	localizedName, nameError := objcbridge.InstanceSelector(applicationClass, "localizedName")
	bundleIdentifier, identifierError := objcbridge.InstanceSelector(applicationClass, "bundleIdentifier")
	booleanFalse, falseError := objcbridge.CFBoolean(false)
	booleanTrue, trueError := objcbridge.CFBoolean(true)

	joined := errors.Join(workspaceError, applicationError, sharedError, frontmostError, processError,
		policyError, nameError, identifierError, falseError, trueError)
	if joined != nil {
		return nil, joined
	}

	return &windowBindings{
		workspace:            objc.ID(workspaceClass).Send(shared),
		frontmostApplication: frontmost,
		processIdentifier:    processIdentifier,
		activationPolicy:     activationPolicy,
		localizedName:        localizedName,
		bundleIdentifier:     bundleIdentifier,
		booleanFalse:         booleanFalse,
		booleanTrue:          booleanTrue,
	}, nil
}

// boundWindows gives the bindings, or ErrBridgeUnavailable with the reason.
func boundWindows() (*windowBindings, error) {
	bindings, bindError := pureWindowBindings()
	if bindError != nil {
		return nil, fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
	}

	return bindings, nil
}

// windowCalls are the calls that take, give up or act on references. The
// lister and its snapshots make them only through here, so a test can count
// every reference taken against every one released.
type windowCalls struct {
	application func(processID int) objcbridge.CFTypeRef
	copy        func(element objcbridge.CFTypeRef, attribute string) (objcbridge.CFTypeRef, error)
	retain      func(ref objcbridge.CFTypeRef) objcbridge.CFTypeRef
	release     func(ref objcbridge.CFTypeRef)
	set         func(element objcbridge.CFTypeRef, attribute string, value objcbridge.CFTypeRef) error
	perform     func(element objcbridge.CFTypeRef, action string) error
}

// realWindowCalls are the objcbridge calls themselves.
func realWindowCalls() windowCalls {
	return windowCalls{
		application: objcbridge.AXApplication,
		copy:        objcbridge.AXCopy,
		retain:      objcbridge.Retain,
		release:     objcbridge.Release,
		set:         objcbridge.AXSet,
		perform:     objcbridge.AXPerform,
	}
}

// pureWindowLister reads windows through purego.
type pureWindowLister struct{ calls windowCalls }

// heldWindow is one listed window's Accessibility element, retained by the
// snapshot until it is released, and its application's process.
type heldWindow struct {
	element   objcbridge.CFTypeRef
	processID int
}

// pureWindows is a snapshot of windows whose elements it holds until
// released, so one of them can be raised later.
type pureWindows struct {
	calls   windowCalls
	held    []heldWindow
	windows []AppWindow
}

// list snapshots the standard windows of every running regular application
// whose identifier starts with the prefix; an empty prefix matches none, as
// NSString's hasPrefix: does. Reading windows cannot be cancelled part way
// and takes milliseconds, so the context is not consulted.
//
//nolint:ireturn // a WindowLister returns the snapshot as its interface
func (lister pureWindowLister) list(_ context.Context, prefix string) (WindowSnapshot, error) {
	bindings, bindError := boundWindows()
	if bindError != nil {
		return nil, bindError
	}

	snapshot := &pureWindows{calls: lister.calls, held: nil, windows: nil}

	objcbridge.WithAutoreleasePool(func() {
		focused := lister.focusedWindow(bindings)
		defer lister.calls.release(focused)

		for _, running := range objcbridge.RunningApplications() {
			identifier := objcbridge.GoStringFromNS(running.Send(bindings.bundleIdentifier))
			if objc.Send[int](running, bindings.activationPolicy) != activationPolicyRegular || identifier == "" ||
				prefix == "" || !strings.HasPrefix(identifier, prefix) {
				continue
			}

			lister.appendWindows(snapshot, bindings, running, identifier, focused)
		}
	})

	return snapshot, nil
}

// focusedWindow is the window in front: the focused window of the frontmost
// application, owned by the caller, or nil.
func (lister pureWindowLister) focusedWindow(bindings *windowBindings) objcbridge.CFTypeRef {
	front := bindings.workspace.Send(bindings.frontmostApplication)
	if front == 0 {
		return 0
	}

	application := lister.calls.application(int(objc.Send[int32](front, bindings.processIdentifier)))
	defer lister.calls.release(application)

	window, copyError := lister.calls.copy(application, attributeFocusedWindow)
	if copyError != nil {
		return 0
	}

	return window
}

// appendWindows adds an application's standard windows to the snapshot,
// retaining each so it outlives the windows array.
func (lister pureWindowLister) appendWindows(snapshot *pureWindows, bindings *windowBindings, running objc.ID,
	identifier string, focused objcbridge.CFTypeRef,
) {
	processID := int(objc.Send[int32](running, bindings.processIdentifier))

	application := lister.calls.application(processID)
	defer lister.calls.release(application)

	windows, copyError := lister.calls.copy(application, attributeWindows)
	if copyError != nil {
		return
	}

	defer lister.calls.release(windows)

	name := objcbridge.GoStringFromNS(running.Send(bindings.localizedName))

	for _, window := range objcbridge.CFArrayItems(windows) {
		if !lister.isStandardWindow(window) {
			continue
		}

		snapshot.windows = append(snapshot.windows, AppWindow{
			Application: name,
			Identifier:  identifier,
			Title:       lister.text(window, attributeTitle),
			Focused:     focused != 0 && objcbridge.Equal(window, focused),
			Index:       len(snapshot.held),
		})
		snapshot.held = append(snapshot.held, heldWindow{element: lister.calls.retain(window), processID: processID})
	}
}

// isStandardWindow reports a window whose subrole is a standard window,
// leaving out dialogs, palettes and pop ups.
func (lister pureWindowLister) isStandardWindow(window objcbridge.CFTypeRef) bool {
	return lister.text(window, attributeSubrole) == standardWindowSubrole
}

// text reads a string attribute, giving "" when it is missing or not a
// string.
func (lister pureWindowLister) text(element objcbridge.CFTypeRef, attribute string) string {
	value, copyError := lister.calls.copy(element, attribute)
	if copyError != nil {
		return ""
	}

	defer lister.calls.release(value)

	return objcbridge.GoString(value)
}

func (snapshot *pureWindows) Windows() []AppWindow { return snapshot.windows }

// Raise brings one window forward: restored if minimised, raised, made its
// application's main window, and its application made frontmost, all through
// Accessibility, as the cgo adapter does. Only the raise itself decides the
// answer; the other steps are made whatever they give.
func (snapshot *pureWindows) Raise(index int) error {
	if index < 0 || index >= len(snapshot.held) {
		return fmt.Errorf("%w: no window %d in a snapshot of %d", errPureWindowNotRaised, index, len(snapshot.held))
	}

	bindings, bindError := boundWindows()
	if bindError != nil {
		return bindError
	}

	window, calls := snapshot.held[index], snapshot.calls

	minimised, copyError := calls.copy(window.element, attributeMinimised)
	if copyError == nil {
		if objcbridge.Equal(minimised, bindings.booleanTrue) {
			_ = calls.set(window.element, attributeMinimised, bindings.booleanFalse)
		}

		calls.release(minimised)
	}

	raiseError := calls.perform(window.element, actionRaise)
	_ = calls.set(window.element, attributeMain, bindings.booleanTrue)

	application := calls.application(window.processID)
	_ = calls.set(application, attributeFrontmost, bindings.booleanTrue)
	calls.release(application)

	if raiseError != nil {
		return fmt.Errorf("%w: %w", errPureWindowNotRaised, raiseError)
	}

	return nil
}

// Release gives up the snapshot's windows. A second call does nothing.
func (snapshot *pureWindows) Release() {
	for _, window := range snapshot.held {
		snapshot.calls.release(window.element)
	}

	snapshot.held = nil
}
