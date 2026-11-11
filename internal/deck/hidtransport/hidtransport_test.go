package hidtransport_test

import (
	"os"
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
