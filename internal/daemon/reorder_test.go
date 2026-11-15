package daemon_test

import (
	"testing"
	"testing/synctest"
	"time"
)

// TestDaemon_KeepsTheRegularPageShowingWhenPagesAreReordered: pages
// reordered in the editor keep the deck on the page it was showing, found by
// its name, rather than on whichever page now has its place.
func TestDaemon_KeepsTheRegularPageShowingWhenPagesAreReordered(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startTwoScreenHarness(t)

		tap(device, forwardKey)

		reordered := twoScreenConfiguration()
		reordered.Pages[1], reordered.Pages[2] = reordered.Pages[2], reordered.Pages[1]

		running.configurations <- reordered

		time.Sleep(settle)
		synctest.Wait()
		tap(device, besideKeyIndex) // still Second, now first in the file

		assertPresses(t, running, "second")

		running.stop(t)
	})
}
