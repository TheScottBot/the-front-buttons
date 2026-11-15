package daemon_test

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// deckAttached attaches a deck and waits for it to be drawn.
func deckAttached(t *testing.T, running *harness) *testsupport.DeckDeviceTestDouble {
	t.Helper()

	device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
	running.watcher.Attach("deck-a", device)
	time.Sleep(settle)
	synctest.Wait()

	return device
}

// pageMemoryTestDouble remembers a page as the state folder would, and
// records every page it is told of.
type pageMemoryTestDouble struct {
	mutex      sync.Mutex
	remembered string
	kept       []string
}

func (memory *pageMemoryTestDouble) Remembered() (string, bool) {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	return memory.remembered, memory.remembered != ""
}

func (memory *pageMemoryTestDouble) Remember(name string) {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	memory.remembered = name
	memory.kept = append(memory.kept, name)
}

func (memory *pageMemoryTestDouble) keptPages() []string {
	memory.mutex.Lock()
	defer memory.mutex.Unlock()

	return slices.Clone(memory.kept)
}

// TestDaemon_StartsOnThePageItLastShowed: a daemon restarted, as by an
// install, opens on the regular page the deck last showed, found by its name,
// rather than the first.
func TestDaemon_StartsOnThePageItLastShowed(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.memory.remembered = secondPageName

		running.configurations <- twoScreenConfiguration()

		device := deckAttached(t, running)
		tap(device, besideKeyIndex) // Second, the page remembered

		assertPresses(t, running, "second")

		running.stop(t)
	})
}

// TestDaemon_RemembersEachPageItShows: paging tells the memory each regular
// page the deck lands on, so the next start can open there.
func TestDaemon_RemembersEachPageItShows(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoScreenConfiguration()

		device := deckAttached(t, running)
		tap(device, forwardKey)
		tap(device, forwardKey)

		if got := running.memory.keptPages(); !slices.Equal(got[len(got)-2:], []string{secondPageName, firstPageName}) {
			t.Fatalf("want Second then First remembered, got %v", got)
		}

		running.stop(t)
	})
}

// TestDaemon_StartsOnTheFirstPage_WhenTheRememberedOneIsGone: a remembered
// page no longer there leaves the deck on the first page.
func TestDaemon_StartsOnTheFirstPage_WhenTheRememberedOneIsGone(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.memory.remembered = "Gone"

		running.configurations <- twoScreenConfiguration()

		device := deckAttached(t, running)
		tap(device, besideKeyIndex)

		assertPresses(t, running, "first")

		running.stop(t)
	})
}
