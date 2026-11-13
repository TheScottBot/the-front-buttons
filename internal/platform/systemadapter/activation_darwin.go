package systemadapter

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices
#include <stdlib.h>
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>

// frontButtonsFreeIdentifier frees a string made by C.CString.
static void frontButtonsFreeIdentifier(char *identifier) {
	free(identifier);
}

// frontButtonsRunningProcess is the process of the first running application
// with the bundle identifier, or 0 if none is running.
static pid_t frontButtonsRunningProcess(const char *identifier) {
	@autoreleasepool {
		NSString *wanted = [NSString stringWithUTF8String:identifier];
		NSRunningApplication *application =
			[NSRunningApplication runningApplicationsWithBundleIdentifier:wanted].firstObject;

		return application == nil ? 0 : [application processIdentifier];
	}
}

// frontButtonsMakeFrontmost asks the process to come to the front through
// Accessibility, which takes milliseconds and brings its front window only.
static void frontButtonsMakeFrontmost(pid_t process) {
	AXUIElementRef element = AXUIElementCreateApplication(process);
	AXUIElementSetAttributeValue(element, kAXFrontmostAttribute, kCFBooleanTrue);
	CFRelease(element);
}

// frontButtonsActivate asks AppKit to bring the process to the front, which
// on the deck took a second every time, so it is asked only when
// Accessibility has not brought the application forward.
static void frontButtonsActivate(pid_t process) {
	@autoreleasepool {
		[[NSRunningApplication runningApplicationWithProcessIdentifier:process] activateWithOptions:0];
	}
}

// frontButtonsInFront reports whether the process is the focused application:
// 1 it is, 2 not yet, 0 Accessibility will not say. Accessibility is asked
// rather than AppKit, whose idea of the frontmost application waits on a run
// loop the daemon does not run.
static int frontButtonsInFront(pid_t process) {
	AXUIElementRef system = AXUIElementCreateSystemWide();
	CFTypeRef focused = NULL;
	AXError copied = AXUIElementCopyAttributeValue(system, kAXFocusedApplicationAttribute, &focused);
	CFRelease(system);

	if (copied != kAXErrorSuccess || focused == NULL) {
		return 0;
	}

	pid_t focusedProcess = 0;
	AXError read = AXUIElementGetPid((AXUIElementRef)focused, &focusedProcess);
	CFRelease(focused);

	if (read != kAXErrorSuccess) {
		return 0;
	}

	return focusedProcess == process ? 1 : 2;
}
*/
import "C"

import (
	"context"
	"fmt"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// How long an application has to come to the front after being asked, and how
// often it is checked meanwhile. Activation usually takes tens of
// milliseconds; half a second leaves room for a busy machine without a key
// press feeling late.
const (
	frontWait  = 500 * time.Millisecond
	frontCheck = 20 * time.Millisecond
	// accessibilityWait is how long Accessibility alone has, before AppKit is
	// asked as well.
	accessibilityWait = 200 * time.Millisecond
)

// frontStates reads frontButtonsInFront's answers.
//
//nolint:gochecknoglobals // a fixed table
var frontStates = map[C.int]FrontState{1: FrontYes, 2: FrontNotYet}

// ApplicationActivator brings applications to the front on macOS.
type ApplicationActivator struct{}

// NewApplicationActivator returns this machine's ApplicationActivator.
func NewApplicationActivator() ApplicationActivator { return ApplicationActivator{} }

// ActivateApplication brings the application to the front, if it is not there
// already, and waits until it is: through Accessibility first, and through
// AppKit only if that has not worked within a fifth of a second. An application already in front is left
// alone, so a key pressed while working in it costs one question to the
// system rather than a round trip to the application, which for a Java IDE is
// slow. Asking needs no permission; seeing where it is needs the
// Accessibility the key presses need anyway.
func (ApplicationActivator) ActivateApplication(activateContext context.Context, identifier string) error {
	wanted := C.CString(identifier)
	defer C.frontButtonsFreeIdentifier(wanted)

	process := C.frontButtonsRunningProcess(wanted)
	if process == 0 {
		return fmt.Errorf("%w: %s", platform.ErrApplicationNotRunning, identifier)
	}

	inFront := func() FrontState {
		state, known := frontStates[C.frontButtonsInFront(process)]
		if !known {
			return FrontUnknown
		}

		return state
	}

	if inFront() == FrontYes {
		return nil
	}

	C.frontButtonsMakeFrontmost(process)

	if WaitUntilInFront(activateContext, identifier, inFront, accessibilityWait, frontCheck) == nil {
		return nil
	}

	C.frontButtonsActivate(process)

	return WaitUntilInFront(activateContext, identifier, inFront, frontWait, frontCheck)
}
