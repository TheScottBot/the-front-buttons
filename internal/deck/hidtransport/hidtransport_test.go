package hidtransport_test

import (
	"os"
	"sync"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// deckTestsVariable enables the device half of the contract on a machine with a
// deck attached. It is off by default so the suite runs anywhere, including the
// CI runners; running it on the deck is part of the TE6 author gate.
const deckTestsVariable = "THE_FRONT_BUTTONS_DECK_TESTS"

func TestWatcher_PassesTheDeckWatcherContract_WithOrWithoutADeck(t *testing.T) {
	t.Parallel()

	testsupport.RunDeckWatcherContract(t, hidtransport.NewWatcher())
}

// TestWatcher_SeveralWatchersAtOnce_DoNotCrashHIDAPI enumerates from several
// watchers on several goroutines at once. hidapi keeps process wide state that
// is not safe to initialise or enumerate from two threads together; when each
// watcher initialised it separately, the device contract aborted the test
// process on the first run against a real deck (evaluation, deck testing).
func TestWatcher_SeveralWatchersAtOnce_DoNotCrashHIDAPI(t *testing.T) {
	t.Parallel()

	const watcherCount = 8

	var group sync.WaitGroup

	for range watcherCount {
		group.Go(func() {
			watcher := hidtransport.NewWatcher()

			for range 4 {
				_, enumerateError := watcher.Enumerate()
				if enumerateError != nil {
					t.Errorf("want enumeration to succeed, got %v", enumerateError)
				}
			}
		})
	}

	group.Wait()
}

func TestDevice_PassesTheDeckDeviceContract_OnAnAttachedDeck(t *testing.T) {
	t.Parallel()

	if os.Getenv(deckTestsVariable) != "1" {
		t.Skipf("set %s=1 with a deck attached to run the device contract (TE6 author gate)", deckTestsVariable)
	}

	watcher := hidtransport.NewWatcher()

	connections, enumerateError := watcher.Enumerate()
	if enumerateError != nil || len(connections) == 0 {
		t.Fatalf("want an attached deck, found %d (%v)", len(connections), enumerateError)
	}

	model, _ := deck.LookupModel(connections[0].VendorID, connections[0].ProductID)

	testsupport.RunDeckDeviceContract(t, model, func(t *testing.T) deck.DeckDevice {
		t.Helper()

		device, openError := watcher.Open(connections[0])
		if openError != nil {
			t.Fatalf("opening the deck: %v", openError)
		}

		return device
	})
}
