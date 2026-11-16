//nolint:testpackage // compares the cgo files' list with the purego one
package systemadapter

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// TestRunningApplications_AgreeBetweenBridges: the cgo quitter and window
// cycler list running applications as the purego ones do, process by
// process, so both see the same applications. A process may start or end
// between the two reads, so it tries a few times.
func TestRunningApplications_AgreeBetweenBridges(t *testing.T) {
	t.Parallel()

	if !CgoAvailable {
		t.Skip("cgo is not in this build")
	}

	const attempts = 3

	var cgo, purego []int32

	for range attempts {
		cgo = cgoRunningProcessIDs()
		purego = puregoRunningProcessIDs()

		if len(cgo) > 0 && slices.Equal(cgo, purego) {
			return
		}
	}

	t.Errorf("want the same running applications, got %d through cgo and %d through purego", len(cgo), len(purego))
}

// puregoRunningProcessIDs is the process of each running application the
// purego adapters see, in their order.
func puregoRunningProcessIDs() []int32 {
	var processIDs []int32

	processIdentifier := objc.RegisterName("processIdentifier")

	objcbridge.WithAutoreleasePool(func() {
		for _, application := range objcbridge.RunningApplications() {
			processIDs = append(processIDs, objc.Send[int32](application, processIdentifier))
		}
	})

	return processIDs
}

// TestNoAdapter_ListsApplicationsThroughTheWorkspace: NSWorkspace's
// runningApplications stays as it was at its first call in a process with no
// main run loop, as the daemon is, so no adapter or helper may list running
// applications through it, through either bridge.
func TestNoAdapter_ListsApplicationsThroughTheWorkspace(t *testing.T) {
	t.Parallel()

	sources, globError := filepath.Glob("*.go")
	if globError != nil {
		t.Fatal(globError)
	}

	helpers, helperError := filepath.Glob("../objcbridge/*.go")
	if helperError != nil {
		t.Fatal(helperError)
	}

	for _, source := range append(sources, helpers...) {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}

		content, readError := os.ReadFile(source) //nolint:gosec // the package's own files
		if readError != nil {
			t.Fatal(readError)
		}

		for _, frozen := range []string{"sharedWorkspace] runningApplications]", `"runningApplications"`} {
			if strings.Contains(string(content), frozen) {
				t.Errorf("%s lists running applications through NSWorkspace (%s)", source, frozen)
			}
		}
	}
}
