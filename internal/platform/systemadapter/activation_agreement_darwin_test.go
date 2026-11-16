//nolint:testpackage // the agreement tests reach the unexported adapters through Bridge
package systemadapter

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The applications the activation tests look for: the Dock always runs and is
// never the focused application, Finder always runs and can come to the
// front, and the last runs nowhere.
const (
	activationDock       = "com.apple.dock"
	activationFinder     = "com.apple.finder"
	activationNotRunning = "org.thefrontbuttons.never-running"
)

// activationRetainCount reads an object's retain count, to see that what the
// adapter took it gave back.
//
//nolint:gochecknoglobals // bound once, for every test
var activationRetainCount = sync.OnceValue(func() func(objcbridge.CFTypeRef) int {
	library, loadError := objcbridge.Framework(objcbridge.PathCoreFoundation)
	if loadError != nil {
		panic(loadError)
	}

	var retainCount func(objcbridge.CFTypeRef) int

	objcbridge.Function(&retainCount, library, "CFGetRetainCount")

	return retainCount
})

// runningProcessOf finds an application's process through NSWorkspace's list,
// a different way from the adapter's, or skips when it is not running.
func runningProcessOf(t *testing.T, identifier string) int {
	t.Helper()

	process := 0

	objcbridge.WithAutoreleasePool(func() {
		for _, application := range objcbridge.RunningApplications() {
			bundle := objcbridge.GoStringFromNS(application.Send(objc.RegisterName("bundleIdentifier")))
			if bundle == identifier && process == 0 {
				process = int(objc.Send[int32](application, objc.RegisterName("processIdentifier")))
			}
		}
	})

	if process == 0 {
		t.Skipf("%s is not running", identifier)
	}

	return process
}

// accessibilityHeaderConstants reads the attribute names Apple's
// AXAttributeConstants.h defines, by constant, from the SDK the Command Line
// Tools install, or skips when there is no SDK.
func accessibilityHeaderConstants(t *testing.T) map[string]string {
	t.Helper()

	sdk, sdkError := exec.CommandContext(t.Context(), "xcrun", "--show-sdk-path").Output()
	if sdkError != nil {
		t.Skipf("no macOS SDK to read the attribute constants from: %v", sdkError)
	}

	header := filepath.Join(strings.TrimSpace(string(sdk)), "System", "Library", "Frameworks",
		"ApplicationServices.framework", "Frameworks", "HIServices.framework", "Headers", "AXAttributeConstants.h")

	text, readError := os.ReadFile(header) //nolint:gosec // a fixed header under the SDK xcrun names
	if readError != nil {
		t.Skipf("reading %s: %v", header, readError)
	}

	constants := map[string]string{}

	definition := regexp.MustCompile(`#define\s+(kAX\w+)\s+CFSTR\("([^"]+)"\)`)

	for _, match := range definition.FindAllStringSubmatch(string(text), -1) {
		constants[match[1]] = match[2]
	}

	return constants
}

// checkAttributeNames wants each name the adapter passes to be the one
// Apple's constant stands for.
func checkAttributeNames(t *testing.T, names map[string]string) {
	t.Helper()

	constants := accessibilityHeaderConstants(t)

	for constant, name := range names {
		if constants[constant] != name {
			t.Errorf("%s: want %q, as AXAttributeConstants.h defines it, got %q", constant, constants[constant], name)
		}
	}
}

func TestActivation_AgreesBetweenBridges_OnAnApplicationNotRunning(t *testing.T) {
	t.Parallel()

	for name, identifier := range map[string]string{"unknown": activationNotRunning, "empty": ""} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) error {
				t.Helper()

				return NewApplicationActivator(bridge).ActivateApplication(t.Context(), identifier)
			})

			if !errors.Is(cgo, platform.ErrApplicationNotRunning) || !errors.Is(purego, platform.ErrApplicationNotRunning) {
				t.Fatalf("want ErrApplicationNotRunning through both, got cgo %v and purego %v", cgo, purego)
			}

			if cgo.Error() != purego.Error() {
				t.Errorf("want the same error through both, got cgo %q and purego %q", cgo, purego)
			}
		})
	}
}

func TestPureActivation_BindsEveryMessage(t *testing.T) {
	t.Parallel()

	messages, bindError := activationBinding()
	if bindError != nil {
		t.Fatalf("want every class, selector and constant bound, got %v", bindError)
	}

	if truth, isBoolean := objcbridge.CFBool(messages.trueValue); !isBoolean || !truth {
		t.Errorf("want kCFBooleanTrue to set AXFrontmost, got %v, %v", truth, isBoolean)
	}
}

func TestPureActivation_PassesApplesAttributeNames(t *testing.T) {
	t.Parallel()

	checkAttributeNames(t, map[string]string{
		"kAXFocusedApplicationAttribute": focusedApplicationAttribute,
		"kAXFrontmostAttribute":          frontmostAttribute,
	})
}

func TestPureActivation_ReportsTheBridgeUnavailable_WhenBindingFailed(t *testing.T) {
	t.Parallel()

	activator := newPureApplicationActivator()
	activator.messages = func() (activationMessages, error) {
		return activationMessages{}, objcbridge.ErrBindingFailed
	}
	activator.accessibility, activator.appKitActivate = refusingAccessibility(t), refusingAppKit(t)

	failure := activator.ActivateApplication(t.Context(), activationFinder)
	if !errors.Is(failure, ErrBridgeUnavailable) {
		t.Fatalf("want ErrBridgeUnavailable, got %v", failure)
	}
}

// refusingAccessibility fails the test if the adapter makes any Accessibility
// call.
func refusingAccessibility(t *testing.T) accessibilityCalls {
	t.Helper()

	return accessibilityCalls{
		systemWide: func() objcbridge.CFTypeRef {
			t.Error("want no system wide element")

			return 0
		},
		application: func(int) objcbridge.CFTypeRef {
			t.Error("want no application element")

			return 0
		},
		copyValue: func(_ objcbridge.CFTypeRef, attribute string) (objcbridge.CFTypeRef, error) {
			t.Errorf("want nothing copied, got %s", attribute)

			return 0, objcbridge.ErrAXCannotComplete
		},
		setValue: func(_ objcbridge.CFTypeRef, attribute string, _ objcbridge.CFTypeRef) error {
			t.Errorf("want nothing set, got %s", attribute)

			return objcbridge.ErrAXCannotComplete
		},
	}
}

// refusingAppKit fails the test if the adapter asks AppKit to activate.
func refusingAppKit(t *testing.T) func(activationMessages, int32) {
	t.Helper()

	return func(_ activationMessages, process int32) { t.Errorf("want AppKit not asked, got process %d", process) }
}

// activationWitness stands in for the Accessibility calls that would act, and
// for the elements macOS would make, with real elements the test holds, so
// what the adapter takes and gives back can be counted.
type activationWitness struct {
	t           *testing.T
	system      objcbridge.CFTypeRef
	application objcbridge.CFTypeRef
	focused     objcbridge.CFTypeRef
	sets        []string
	appKitAsks  []int32
}

func newActivationWitness(t *testing.T, applicationProcess, focusedProcess int) *activationWitness {
	t.Helper()

	witness := &activationWitness{
		t:           t,
		system:      objcbridge.AXSystemWide(),
		application: objcbridge.AXApplication(applicationProcess),
		focused:     objcbridge.AXApplication(focusedProcess),
		sets:        nil,
		appKitAsks:  nil,
	}

	t.Cleanup(func() {
		for _, element := range []objcbridge.CFTypeRef{witness.system, witness.application, witness.focused} {
			objcbridge.Release(element)
		}
	})

	return witness
}

func (witness *activationWitness) calls() accessibilityCalls {
	return accessibilityCalls{
		systemWide: func() objcbridge.CFTypeRef { return objcbridge.Retain(witness.system) },
		application: func(processID int) objcbridge.CFTypeRef {
			if owner, _ := objcbridge.AXProcessID(witness.application); owner != processID {
				witness.t.Errorf("want the element of process %d, asked for %d", owner, processID)
			}

			return objcbridge.Retain(witness.application)
		},
		copyValue: func(element objcbridge.CFTypeRef, attribute string) (objcbridge.CFTypeRef, error) {
			if element != witness.system || attribute != focusedApplicationAttribute {
				witness.t.Errorf("want %s copied from the system wide element, got %s from %#x",
					focusedApplicationAttribute, attribute, element)
			}

			return objcbridge.Retain(witness.focused), nil
		},
		setValue: func(element objcbridge.CFTypeRef, attribute string, value objcbridge.CFTypeRef) error {
			truth, _ := objcbridge.CFBool(value)
			if element != witness.application || attribute != frontmostAttribute || !truth {
				witness.t.Errorf("want %s set true on the application, got %s on %#x", frontmostAttribute, attribute, element)
			}

			witness.sets = append(witness.sets, attribute)

			return nil
		},
	}
}

func (witness *activationWitness) activator() pureApplicationActivator {
	activator := newPureApplicationActivator()
	activator.accessibility = witness.calls()
	activator.appKitActivate = func(_ activationMessages, process int32) {
		witness.appKitAsks = append(witness.appKitAsks, process)
	}

	return activator
}

// balanced wants every element the adapter was given back to the one
// reference the witness holds.
func (witness *activationWitness) balanced() {
	witness.t.Helper()

	for name, element := range map[string]objcbridge.CFTypeRef{
		"system wide": witness.system, "application": witness.application, "focused": witness.focused,
	} {
		if count := activationRetainCount()(element); count != 1 {
			witness.t.Errorf("%s element: want it released back to 1 reference, got %d", name, count)
		}
	}
}

func TestPureActivation_LeavesAnApplicationAlreadyInFront(t *testing.T) {
	t.Parallel()

	dock := runningProcessOf(t, activationDock)
	witness := newActivationWitness(t, dock, dock)

	failure := witness.activator().ActivateApplication(t.Context(), activationDock)
	if failure != nil {
		t.Fatalf("want nil for the application in front, got %v", failure)
	}

	if len(witness.sets) != 0 || len(witness.appKitAsks) != 0 {
		t.Errorf("want nothing asked of an application in front, got sets %v and AppKit %v", witness.sets, witness.appKitAsks)
	}

	witness.balanced()
}

func TestPureActivation_AsksAccessibilityThenAppKit_AndReleasesWhatItTook(t *testing.T) {
	t.Parallel()

	dock := runningProcessOf(t, activationDock)
	witness := newActivationWitness(t, dock, os.Getpid())
	started := time.Now()

	failure := witness.activator().ActivateApplication(t.Context(), activationDock)
	if !errors.Is(failure, platform.ErrApplicationNotInFront) {
		t.Fatalf("want ErrApplicationNotInFront for an application that never comes forward, got %v", failure)
	}

	if len(witness.sets) != 1 {
		t.Errorf("want %s set once, got %v", frontmostAttribute, witness.sets)
	}

	if len(witness.appKitAsks) != 1 || int(witness.appKitAsks[0]) != dock {
		t.Errorf("want AppKit asked once for process %d, got %v", dock, witness.appKitAsks)
	}

	if waited := time.Since(started); waited < pureAccessibilityWait+pureFrontWait {
		t.Errorf("want Accessibility given %s and AppKit %s, gave up after %s", pureAccessibilityWait, pureFrontWait, waited)
	}

	witness.balanced()
}

// TestActivation_BringsFinderForward_ThroughEachBridge acts: it brings Finder
// to the front through each bridge, reads the focused application back
// through Accessibility, and puts the application that was in front back.
//
//nolint:paralleltest // acting tests run one at a time, under the desktop lock
func TestActivation_BringsFinderForward_ThroughEachBridge(t *testing.T) {
	actingAllowed(t)

	finder := runningProcessOf(t, activationFinder)

	eachBridge(t, func(t *testing.T, bridge Bridge) struct{} {
		t.Helper()

		before, readError := pureFocusedProcess(systemAccessibility())
		if readError != nil {
			t.Fatalf("reading the focused application: %v", readError)
		}

		activateError := NewApplicationActivator(bridge).ActivateApplication(t.Context(), activationFinder)
		if activateError != nil {
			t.Fatalf("activating Finder: %v", activateError)
		}

		if after, _ := pureFocusedProcess(systemAccessibility()); after != finder {
			t.Errorf("want Finder (process %d) focused, got process %d", finder, after)
		}

		restoreFocus(t, before)

		return struct{}{}
	})
}

// TestPureActivation_BringsFinderForward_ThroughAppKitAlone acts: with the
// Accessibility request made to do nothing, only AppKit's activateWithOptions:
// can bring Finder forward.
//
//nolint:paralleltest // acting tests run one at a time, under the desktop lock
func TestPureActivation_BringsFinderForward_ThroughAppKitAlone(t *testing.T) {
	actingAllowed(t)

	finder := runningProcessOf(t, activationFinder)

	before, readError := pureFocusedProcess(systemAccessibility())
	if readError != nil {
		t.Fatalf("reading the focused application: %v", readError)
	}

	activator := newPureApplicationActivator()
	activator.accessibility.setValue = func(objcbridge.CFTypeRef, string, objcbridge.CFTypeRef) error { return nil }

	activateError := activator.ActivateApplication(t.Context(), activationFinder)
	if activateError != nil {
		t.Fatalf("activating Finder through AppKit: %v", activateError)
	}

	if after, _ := pureFocusedProcess(systemAccessibility()); after != finder {
		t.Errorf("want Finder (process %d) focused, got process %d", finder, after)
	}

	restoreFocus(t, before)
}

// restoreFocus brings back the process that was in front before an acting
// test, through AppKit.
func restoreFocus(t *testing.T, process int) {
	t.Helper()

	messages, bindError := activationBinding()
	if bindError != nil {
		t.Fatalf("binding activation: %v", bindError)
	}

	activateThroughAppKit(messages, int32(process)) //nolint:gosec // a process ID read from macOS fits pid_t
}
