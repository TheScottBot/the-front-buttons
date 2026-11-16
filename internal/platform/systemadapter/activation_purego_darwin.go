package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The Accessibility attributes the activator reads and sets, spelt as
// kAXFocusedApplicationAttribute and kAXFrontmostAttribute are in
// AXAttributeConstants.h.
const (
	focusedApplicationAttribute = "AXFocusedApplication"
	frontmostAttribute          = "AXFrontmost"
)

// The purego activator's waits, the same as the cgo adapter's: half a second
// to come to the front, checked every 20 milliseconds, of which Accessibility
// alone has a fifth of a second before AppKit is asked as well.
const (
	pureFrontWait         = 500 * time.Millisecond
	pureFrontCheck        = 20 * time.Millisecond
	pureAccessibilityWait = 200 * time.Millisecond
)

// accessibilityCalls are the Accessibility calls the purego adapters make,
// held in one value so tests can count what is made, copied, set and released
// without acting on the desktop.
type accessibilityCalls struct {
	systemWide  func() objcbridge.CFTypeRef
	application func(processID int) objcbridge.CFTypeRef
	copyValue   func(element objcbridge.CFTypeRef, attribute string) (objcbridge.CFTypeRef, error)
	setValue    func(element objcbridge.CFTypeRef, attribute string, value objcbridge.CFTypeRef) error
}

// systemAccessibility is macOS's own Accessibility.
func systemAccessibility() accessibilityCalls {
	return accessibilityCalls{
		systemWide:  objcbridge.AXSystemWide,
		application: objcbridge.AXApplication,
		copyValue:   objcbridge.AXCopy,
		setValue:    objcbridge.AXSet,
	}
}

// activationMessages are NSRunningApplication's messages and kCFBooleanTrue,
// which the activator binds once.
type activationMessages struct {
	applicationClass  objc.ID
	withProcess       objc.SEL
	processIdentifier objc.SEL
	activate          objc.SEL
	trueValue         objcbridge.CFTypeRef
}

//nolint:gochecknoglobals // bound once, on first use
var activationBinding = objcbridge.Bind(func() (activationMessages, error) {
	_, loadError := objcbridge.Framework(objcbridge.PathAppKit)
	if loadError != nil {
		return activationMessages{}, fmt.Errorf("loading AppKit: %w", loadError)
	}

	class, classError := objcbridge.Class("NSRunningApplication")
	withProcess, withProcessError := objcbridge.ClassSelector(class, "runningApplicationWithProcessIdentifier:")
	processIdentifier, processError := objcbridge.InstanceSelector(class, "processIdentifier")
	activate, activateError := objcbridge.InstanceSelector(class, "activateWithOptions:")
	trueValue, trueError := objcbridge.CFBoolean(true)

	failure := errors.Join(classError, withProcessError, processError, activateError, trueError)
	if failure != nil {
		return activationMessages{}, failure
	}

	// Reading the constant's type loads the Accessibility calls too, so a
	// library missing from them fails here, where the panic is recovered.
	if !objcbridge.IsType(trueValue, objcbridge.KindBoolean) {
		return activationMessages{}, fmt.Errorf("%w: kCFBooleanTrue is not a boolean", objcbridge.ErrSymbolMissing)
	}

	return activationMessages{
		applicationClass:  objc.ID(class),
		withProcess:       withProcess,
		processIdentifier: processIdentifier,
		activate:          activate,
		trueValue:         trueValue,
	}, nil
})

// pureApplicationActivator brings applications to the front through purego,
// making the same calls as the cgo adapter.
type pureApplicationActivator struct {
	messages      func() (activationMessages, error)
	accessibility accessibilityCalls
	// appKitActivate asks AppKit to bring the process forward.
	appKitActivate func(messages activationMessages, process int32)
}

// newPureApplicationActivator returns the ApplicationActivator through purego.
func newPureApplicationActivator() pureApplicationActivator {
	return pureApplicationActivator{
		messages:       activationBinding,
		accessibility:  systemAccessibility(),
		appKitActivate: activateThroughAppKit,
	}
}

// ActivateApplication brings the application to the front, if it is not there
// already, and waits until it is: through Accessibility first, and through
// AppKit only if that has not worked within a fifth of a second, as the cgo
// adapter does.
func (activator pureApplicationActivator) ActivateApplication(
	activateContext context.Context,
	identifier string,
) error {
	messages, bindError := activator.messages()
	if bindError != nil {
		return fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
	}

	process := firstRunningProcess(identifier, messages.processIdentifier)
	if process == 0 {
		return fmt.Errorf("%w: %s", platform.ErrApplicationNotRunning, identifier)
	}

	inFront := func() FrontState { return pureFrontState(activator.accessibility, process) }

	if inFront() == FrontYes {
		return nil
	}

	activator.makeFrontmost(messages, process)

	if WaitUntilInFront(activateContext, identifier, inFront, pureAccessibilityWait, pureFrontCheck) == nil {
		return nil
	}

	activator.appKitActivate(messages, process)

	return WaitUntilInFront(activateContext, identifier, inFront, pureFrontWait, pureFrontCheck)
}

// makeFrontmost asks the process to come to the front through Accessibility,
// which takes milliseconds and brings its front window only.
func (activator pureApplicationActivator) makeFrontmost(messages activationMessages, process int32) {
	element := activator.accessibility.application(int(process))
	defer objcbridge.Release(element)

	// A refusal shows as the application not coming forward, which the wait
	// that follows reports.
	_ = activator.accessibility.setValue(element, frontmostAttribute, messages.trueValue)
}

// activateThroughAppKit asks AppKit to bring the process to the front, which
// on the deck took a second every time, so it is asked only when
// Accessibility has not brought the application forward.
func activateThroughAppKit(messages activationMessages, process int32) {
	objcbridge.WithAutoreleasePool(func() {
		application := messages.applicationClass.Send(messages.withProcess, process)
		application.Send(messages.activate, uint(0))
	})
}

// firstRunningProcess is the process of the first running application with
// the bundle identifier, or 0 if none is running.
func firstRunningProcess(identifier string, processIdentifier objc.SEL) int32 {
	var process int32

	objcbridge.WithAutoreleasePool(func() {
		if running := objcbridge.RunningApplicationsWithBundleID(identifier); len(running) > 0 {
			process = objc.Send[int32](running[0], processIdentifier)
		}
	})

	return process
}

// pureFocusedProcess reads the process of the focused application through
// Accessibility, rather than AppKit, whose idea of the frontmost application
// waits on a run loop the daemon does not run.
func pureFocusedProcess(accessibility accessibilityCalls) (int, error) {
	system := accessibility.systemWide()
	focused, copyError := accessibility.copyValue(system, focusedApplicationAttribute)
	objcbridge.Release(system)

	if copyError != nil {
		return 0, copyError
	}

	defer objcbridge.Release(focused)

	process, readError := objcbridge.AXProcessID(focused)
	if readError != nil {
		return 0, fmt.Errorf("reading the focused application: %w", readError)
	}

	return process, nil
}

// pureFrontState reports whether the process is the focused application, or
// FrontUnknown when Accessibility will not say.
func pureFrontState(accessibility accessibilityCalls, process int32) FrontState {
	focused, readError := pureFocusedProcess(accessibility)

	switch {
	case readError != nil:
		return FrontUnknown
	case focused == int(process):
		return FrontYes
	default:
		return FrontNotYet
	}
}
