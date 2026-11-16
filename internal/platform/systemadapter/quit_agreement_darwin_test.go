//nolint:testpackage // the quitter's binding and matching are unexported
package systemadapter

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The Dock runs whenever someone is logged in, so the read only tests find
// it; Calculator is what the acting tests open and quit.
const (
	quitTestDockBundleID       = "com.apple.dock"
	quitTestCalculatorBundleID = "com.apple.calculator"
	quitTestUnknownBundleID    = "com.example.front-buttons.no-such-application"
	quitTestWait               = 10 * time.Second
	quitTestPoll               = 100 * time.Millisecond
)

// quitTestApplication is what the tests read of a running application.
type quitTestApplication struct {
	name      string
	fileName  string
	processID int32
}

// readQuitTestApplication reads an NSRunningApplication's name, its bundle's
// file name without the extension, worked out in Go, and its process ID.
func readQuitTestApplication(application objc.ID) quitTestApplication {
	bundlePath := application.Send(objc.RegisterName("bundleURL")).Send(objc.RegisterName("path"))
	fileName := ""

	if bundlePath != 0 {
		base := path.Base(objcbridge.GoStringFromNS(bundlePath))
		fileName = strings.TrimSuffix(base, path.Ext(base))
	}

	return quitTestApplication{
		name:      objcbridge.GoStringFromNS(application.Send(objc.RegisterName("localizedName"))),
		fileName:  fileName,
		processID: objc.Send[int32](application, objc.RegisterName("processIdentifier")),
	}
}

// quitCandidatesFor lists what the quitter would ask to quit for the
// identifier, without asking any of them.
func quitCandidatesFor(t *testing.T, identifier string) []quitTestApplication {
	t.Helper()

	messages, bindError := quitBinding()
	if bindError != nil {
		t.Fatalf("binding the quitter: %v", bindError)
	}

	var found []quitTestApplication

	objcbridge.WithAutoreleasePool(func() {
		eachQuitCandidate(messages, MacOSQuitTarget(identifier), func(application objc.ID) {
			found = append(found, readQuitTestApplication(application))
		})
	})

	return found
}

// everyRunningApplication reads every running application.
func everyRunningApplication() []quitTestApplication {
	var everything []quitTestApplication

	objcbridge.WithAutoreleasePool(func() {
		for _, application := range objcbridge.RunningApplications() {
			everything = append(everything, readQuitTestApplication(application))
		}
	})

	return everything
}

func TestPureQuit_FindsTheDock_ByBundleID_ByName_AndByName_dot_app(t *testing.T) {
	t.Parallel()

	byBundle := quitCandidatesFor(t, quitTestDockBundleID)
	if len(byBundle) != 1 || byBundle[0].name != "Dock" || byBundle[0].processID <= 0 {
		t.Fatalf("want the one Dock by its bundle ID, got %v", byBundle)
	}

	for _, identifier := range []string{"Dock", "Dock.app"} {
		if found := quitCandidatesFor(t, identifier); !slices.Equal(found, byBundle) {
			t.Errorf("%s: want the Dock %v, got %v", identifier, byBundle, found)
		}
	}
}

// TestPureQuit_FindsAnApplication_ByItsNameAndByItsBundlesFileName checks
// each way the cgo adapter matches a name on its own, with every running
// application whose bundle's file name is not its name.
func TestPureQuit_FindsAnApplication_ByItsNameAndByItsBundlesFileName(t *testing.T) {
	t.Parallel()

	checked := 0

	for _, application := range everyRunningApplication() {
		if application.fileName == "" || application.name == "" || application.fileName == application.name {
			continue
		}

		checked++

		for _, identifier := range []string{application.name, application.fileName} {
			if MacOSQuitTarget(identifier) != (QuitTarget{Name: identifier, ByBundle: false}) {
				continue
			}

			if found := quitCandidatesFor(t, identifier); !slices.Contains(found, application) {
				t.Errorf("%q: want %v among %v", identifier, application, found)
			}
		}
	}

	t.Logf("checked %d applications whose bundle's file name is not their name", checked)
}

func TestPureQuit_FindsNothing_ForAnUnknownBundleID_Name_OrTextThatIsNotUTF8(t *testing.T) {
	t.Parallel()

	for _, identifier := range []string{quitTestUnknownBundleID, "Front Buttons No Such Application", "Dock\xff", ""} {
		if found := quitCandidatesFor(t, identifier); len(found) != 0 {
			t.Errorf("%q: want nothing, got %v", identifier, found)
		}
	}
}

// TestPureQuit_ANameEndsAtItsFirstNUL, as the cgo adapter's C string does.
func TestPureQuit_ANameEndsAtItsFirstNUL(t *testing.T) {
	t.Parallel()

	want := quitCandidatesFor(t, "Dock")
	if found := quitCandidatesFor(t, "Dock\x00 and more"); len(want) != 1 || !slices.Equal(found, want) {
		t.Errorf("want the Dock %v, got %v", want, found)
	}
}

func TestPureQuit_BindingAMissingName_ReportsTheBridgeUnavailable(t *testing.T) {
	t.Parallel()

	for field, rename := range map[string]func(*quitNames){
		"application class": func(names *quitNames) { names.applicationClass = "NSRunningApp" },
		"terminate":         func(names *quitNames) { names.terminate = "terminat" },
		"name":              func(names *quitNames) { names.localizedName = "localisedName" },
		"bundle":            func(names *quitNames) { names.bundleURL = "bundleURI" },
		"URL class":         func(names *quitNames) { names.urlClass = "NSURI" },
		"last component":    func(names *quitNames) { names.lastPathComponent = "lastComponent" },
		"string class":      func(names *quitNames) { names.stringClass = "NSStrin" },
		"extension":         func(names *quitNames) { names.deletingPathExtension = "stringByDeletingExtension" },
	} {
		names := systemQuitNames()
		rename(&names)

		// The binding fails before any application is looked for, so this
		// quits nothing.
		binding := objcbridge.Bind(func() (quitMessages, error) { return bindQuit(names) })

		quitError := quitApplication(binding, quitTestUnknownBundleID)
		if !errors.Is(quitError, ErrBridgeUnavailable) || !errors.Is(quitError, objcbridge.ErrBindingFailed) {
			t.Errorf("%s: want ErrBridgeUnavailable for a failed binding, got %v", field, quitError)
		}
	}
}

func TestQuitAgreement_BothBridgesMakeAQuitter(t *testing.T) {
	t.Parallel()

	cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) bool {
		t.Helper()

		return NewApplicationQuitter(bridge) != nil
	})

	if !cgo || !purego {
		t.Errorf("want a quitter through each bridge, got cgo %v and purego %v", cgo, purego)
	}
}

// skipCgoWithoutIt skips the cgo half of an agreement in a build without cgo.
func skipCgoWithoutIt(t *testing.T, bridge Bridge) {
	t.Helper()

	if bridge == BridgeCgo && !CgoAvailable {
		t.Skip("this build has no cgo")
	}
}

// TestQuitAgreement_NeitherBridgeQuitsAnythingForAnUnknownApplication asks
// each bridge to quit applications that are not running. It is guarded as
// acting: a quitter whose matching was wrong would quit what it found.
//
//nolint:paralleltest // may quit applications on the shared desktop if the matching is wrong
func TestQuitAgreement_NeitherBridgeQuitsAnythingForAnUnknownApplication(t *testing.T) {
	actingAllowed(t)

	before := len(everyRunningApplication())

	for _, identifier := range []string{quitTestUnknownBundleID, "Front Buttons No Such Application"} {
		cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) error {
			t.Helper()
			skipCgoWithoutIt(t, bridge)

			return NewApplicationQuitter(bridge).QuitApplication(t.Context(), identifier)
		})

		if cgo != nil || purego != nil {
			t.Errorf("%s: want no error through each bridge, got cgo %v and purego %v", identifier, cgo, purego)
		}
	}

	if after := len(everyRunningApplication()); after < before {
		t.Errorf("want nothing quit, got %d running applications before and %d after", before, after)
	}
}

// waitForCalculator waits until Calculator is running, or not, as wanted.
func waitForCalculator(t *testing.T, running bool) bool {
	t.Helper()

	for deadline := time.Now().Add(quitTestWait); time.Now().Before(deadline); time.Sleep(quitTestPoll) {
		if (len(quitCandidatesFor(t, quitTestCalculatorBundleID)) > 0) == running {
			return true
		}
	}

	return false
}

// openCalculator opens Calculator in the background, without raising it.
func openCalculator(t *testing.T) error {
	t.Helper()

	// The system's open command, with fixed arguments and no shell.
	output, openError := exec.CommandContext(t.Context(), "/usr/bin/open", "-g", "-j", "-b",
		quitTestCalculatorBundleID).CombinedOutput()
	if openError != nil {
		return fmt.Errorf("opening Calculator: %w: %s", openError, output)
	}

	if !waitForCalculator(t, true) {
		return fmt.Errorf("opening Calculator: not running after %v", quitTestWait) //nolint:err113 // a test's own message
	}

	return nil
}

// TestQuitAgreement_EachBridgeQuitsCalculator_ByBundleIDAndByName opens
// Calculator and quits it through each bridge, each way the identifier can
// name it. It acts on the desktop, so it runs alone.
//
//nolint:paralleltest // opens and quits an application on the shared desktop
func TestQuitAgreement_EachBridgeQuitsCalculator_ByBundleIDAndByName(t *testing.T) {
	actingAllowed(t)

	for _, identifier := range []string{quitTestCalculatorBundleID, "Calculator", "Calculator.app"} {
		cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) bool {
			t.Helper()
			skipCgoWithoutIt(t, bridge)

			openError := openCalculator(t)
			if openError != nil {
				t.Fatal(openError)
			}

			quitError := NewApplicationQuitter(bridge).QuitApplication(t.Context(), identifier)
			if quitError != nil {
				t.Fatalf("quitting %s: %v", identifier, quitError)
			}

			return waitForCalculator(t, false)
		})

		if !purego || (CgoAvailable && !cgo) {
			t.Errorf("%s: want Calculator quit through each bridge, got cgo %v and purego %v", identifier, cgo, purego)
		}
	}
}
