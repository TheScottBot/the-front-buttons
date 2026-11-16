//nolint:testpackage // the agreement tests reach the unexported adapters through Bridge
package systemadapter

import (
	"os"
	"testing"
)

// actingVariable lets the agreement tests that change the desktop run. They
// are off by default because they activate and quit applications, post keys,
// write the clipboard and raise windows; they run one at a time, under the
// tracker's desktop lock.
const actingVariable = "THE_FRONT_BUTTONS_ACTING"

// eachBridge runs check once through each bridge, as subtests named cgo
// and purego, and returns what each gave, for the test to compare.
//
//nolint:nonamedreturns,ireturn // the names say which bridge gave which answer, of the test's own type
func eachBridge[T any](t *testing.T, check func(t *testing.T, bridge Bridge) T) (cgo T, purego T) {
	t.Helper()

	t.Run(string(BridgeCgo), func(t *testing.T) { cgo = check(t, BridgeCgo) })
	t.Run(string(BridgePurego), func(t *testing.T) { purego = check(t, BridgePurego) })

	return cgo, purego
}

// actingAllowed skips a test that changes the desktop (activates, quits,
// posts keys, writes the clipboard, raises a window) unless
// THE_FRONT_BUTTONS_ACTING is set; those run one at a time, under the
// tracker's desktop lock.
func actingAllowed(t *testing.T) {
	t.Helper()

	if os.Getenv(actingVariable) == "" {
		t.Skipf("set %s=1 to run tests that change the desktop, under the desktop lock", actingVariable)
	}
}

func TestEachBridge_RunsTheCheckThroughBothBridges_AndReturnsEachAnswer(t *testing.T) {
	t.Parallel()

	var seen []Bridge

	cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) string {
		t.Helper()

		if t.Name() != "TestEachBridge_RunsTheCheckThroughBothBridges_AndReturnsEachAnswer/"+string(bridge) {
			t.Errorf("want the subtest named after the bridge %s, got %s", bridge, t.Name())
		}

		seen = append(seen, bridge)

		return "through " + string(bridge)
	})

	if cgo != "through cgo" || purego != "through purego" {
		t.Errorf("want each bridge's own answer, got cgo %q and purego %q", cgo, purego)
	}

	if len(seen) != 2 || seen[0] != BridgeCgo || seen[1] != BridgePurego {
		t.Errorf("want the check run once through cgo then once through purego, got %v", seen)
	}
}

// TestActingAllowed_SkipsWithoutTheVariable_AndRunsWithIt sets the variable,
// so it cannot run in parallel with other tests.
//
//nolint:paralleltest // t.Setenv cannot be used in a parallel test
func TestActingAllowed_SkipsWithoutTheVariable_AndRunsWithIt(t *testing.T) {
	for _, example := range []struct {
		value    string
		skipping bool
	}{
		{value: "", skipping: true},
		{value: "1", skipping: false},
	} {
		t.Setenv(actingVariable, example.value)

		reachedPastTheGuard := false

		var inner *testing.T

		t.Run("acting="+example.value, func(t *testing.T) {
			inner = t

			actingAllowed(t)

			reachedPastTheGuard = true
		})

		if inner.Skipped() != example.skipping || reachedPastTheGuard == example.skipping {
			t.Errorf("%s=%q: want skipped %v, got skipped %v and reached past the guard %v",
				actingVariable, example.value, example.skipping, inner.Skipped(), reachedPastTheGuard)
		}
	}
}
