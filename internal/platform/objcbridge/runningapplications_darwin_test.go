package objcbridge_test

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// dockBundleID is the Dock's, which runs whenever someone is logged in.
const dockBundleID = "com.apple.dock"

// runningApplication is what the tests read of an NSRunningApplication.
type runningApplication struct {
	bundleID  string
	processID int32
}

func describeApplications(applications []objc.ID) []runningApplication {
	bundleIdentifier := objc.RegisterName("bundleIdentifier")
	processIdentifier := objc.RegisterName("processIdentifier")

	described := make([]runningApplication, 0, len(applications))

	for _, application := range applications {
		described = append(described, runningApplication{
			bundleID:  objcbridge.GoStringFromNS(application.Send(bundleIdentifier)),
			processID: objc.Send[int32](application, processIdentifier),
		})
	}

	return described
}

func TestRunningApplications_ListsTheDock_AndTheBundleLookupFindsTheSameProcess(t *testing.T) {
	t.Parallel()

	var everything, dock []runningApplication

	objcbridge.WithAutoreleasePool(func() {
		everything = describeApplications(objcbridge.RunningApplications())
		dock = describeApplications(objcbridge.RunningApplicationsWithBundleID(dockBundleID))
	})

	if len(dock) != 1 || dock[0].bundleID != dockBundleID || dock[0].processID <= 0 {
		t.Fatalf("want the one Dock by its bundle ID, got %v", dock)
	}

	if !slices.Contains(everything, dock[0]) {
		t.Errorf("want the Dock %v among all %d running applications", dock[0], len(everything))
	}
}

func TestRunningApplicationsWithBundleID_GivesNone_ForAnUnknownOrEmptyID(t *testing.T) {
	t.Parallel()

	objcbridge.WithAutoreleasePool(func() {
		for _, bundleID := range []string{"", "com.example.front-buttons.no-such-application"} {
			if found := objcbridge.RunningApplicationsWithBundleID(bundleID); len(found) != 0 {
				t.Errorf("%q: want none, got %d", bundleID, len(found))
			}
		}
	})
}

// workspaceApplications is NSWorkspace's own list of running applications,
// which a process that runs no main run loop sees as it was at its first
// call and never again.
func workspaceApplications() []runningApplication {
	workspace := objc.ID(objc.GetClass("NSWorkspace")).Send(objc.RegisterName("sharedWorkspace"))
	array := workspace.Send(objc.RegisterName("runningApplications"))

	items := objcbridge.CFArrayItems(objcbridge.CFTypeRef(array))
	applications := make([]objc.ID, 0, len(items))

	for _, item := range items {
		applications = append(applications, objc.ID(item))
	}

	return describeApplications(applications)
}

// sameApplications compares two lists of running applications as sets.
func sameApplications(first, second []runningApplication) bool {
	byProcess := func(left, right runningApplication) int { return int(left.processID - right.processID) }

	first, second = slices.Clone(first), slices.Clone(second)
	slices.SortFunc(first, byProcess)
	slices.SortFunc(second, byProcess)

	return slices.Equal(first, second)
}

// TestRunningApplications_ListsWhatTheWorkspaceListsAtItsFirstCall: the
// list is built process by process, and at NSWorkspace's first call in this
// process, while its list is current, the two hold the same applications.
// A process may start or end between the two reads, so it tries a few times.
func TestRunningApplications_ListsWhatTheWorkspaceListsAtItsFirstCall(t *testing.T) {
	t.Parallel()

	const attempts = 3

	var listed, workspace []runningApplication

	for range attempts {
		objcbridge.WithAutoreleasePool(func() {
			workspace = workspaceApplications()
			listed = describeApplications(objcbridge.RunningApplications())
		})

		if len(listed) > 0 && sameApplications(listed, workspace) {
			return
		}
	}

	t.Errorf("want the workspace's %d applications, got %d listed", len(workspace), len(listed))
}

// TestRunningApplications_SeesAnApplicationLaunchedAfterTheFirstCall opens
// and quits Calculator, so it acts on the desktop and runs only with
// THE_FRONT_BUTTONS_ACTING set. NSWorkspace's own list, read first, never
// shows it; the list built process by process shows it while it runs and
// not once it has quit.
//
//nolint:paralleltest // opens and quits an application on the shared desktop
func TestRunningApplications_SeesAnApplicationLaunchedAfterTheFirstCall(t *testing.T) {
	if os.Getenv("THE_FRONT_BUTTONS_ACTING") == "" {
		t.Skip("acts on the desktop; set THE_FRONT_BUTTONS_ACTING to run it")
	}

	const calculator = "com.apple.calculator"

	running := func() bool {
		found := false

		objcbridge.WithAutoreleasePool(func() {
			found = slices.ContainsFunc(describeApplications(objcbridge.RunningApplications()),
				func(application runningApplication) bool { return application.bundleID == calculator })
		})

		return found
	}

	objcbridge.WithAutoreleasePool(func() { _ = workspaceApplications() })

	if running() {
		t.Skip("Calculator is already running")
	}

	openError := exec.CommandContext(t.Context(), "/usr/bin/open", "-g", "-j", "-b", calculator).Run()
	if openError != nil {
		t.Fatalf("opening Calculator: %v", openError)
	}

	t.Cleanup(func() {
		_ = exec.CommandContext(context.Background(), "/usr/bin/osascript", "-e", `quit app id "`+calculator+`"`).Run()
	})

	if !eventually(running, true) {
		t.Fatal("want Calculator listed once it has launched")
	}

	quitError := exec.CommandContext(t.Context(), "/usr/bin/osascript", "-e", `quit app id "`+calculator+`"`).Run()
	if quitError != nil {
		t.Fatalf("quitting Calculator: %v", quitError)
	}

	if !eventually(running, false) {
		t.Error("want Calculator gone from the list once it has quit")
	}
}

// eventually waits up to ten seconds for check to give want.
func eventually(check func() bool, want bool) bool {
	const (
		wait = 10 * time.Second
		poll = 100 * time.Millisecond
	)

	for deadline := time.Now().Add(wait); time.Now().Before(deadline); time.Sleep(poll) {
		if check() == want {
			return true
		}
	}

	return false
}
