package objcbridge_test

import (
	"slices"
	"testing"

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
