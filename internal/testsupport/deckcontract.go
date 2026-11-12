package testsupport

import (
	"errors"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// contractReadTimeout is how long the contract waits for an input report that
// may not come; long enough for a real deck to answer, short enough to keep the
// suite quick when no key is pressed.
const contractReadTimeout = 50 * time.Millisecond

// contractBrightness is a mid brightness, so the contract changes nothing a
// person would notice on a real deck.
const contractBrightness = 50

// A read left waiting this long, with no key pressed, must not hold up a write:
// the daemon reads keys on one goroutine and draws on another, and on the first
// run against a real deck every write waited out the read in progress, so a page
// took seconds to draw (evaluation, deck testing).
const (
	contractPendingReadTimeout = 500 * time.Millisecond
	contractReadStartPause     = 20 * time.Millisecond
	contractWriteBound         = 100 * time.Millisecond
)

// RunDeckDeviceContract is the one contract suite for the DeckDevice port. The
// adapter and the test double both run it (specification section 2), so the
// double cannot promise behaviour the real deck does not have. openDevice must
// return a freshly opened device for the given model each time it is called.
func RunDeckDeviceContract(
	t *testing.T,
	model deck.Model,
	openDevice func(t *testing.T) deck.DeckDevice,
) {
	t.Helper()

	clauses := map[string]func(t *testing.T, model deck.Model, device deck.DeckDevice){
		"accepts the reset key stream report":                contractAcceptsResetKeyStream,
		"accepts a brightness feature report":                contractAcceptsBrightness,
		"returns a decodable serial number feature report":   contractReturnsSerial,
		"reads nothing or exactly one whole input report":    contractReadsWholeReports,
		"refuses use after close, and closing twice is safe": contractRefusesUseAfterClose,
		"a write is not held up by a read in progress":       contractWriteDuringRead,
	}

	for clauseName, clause := range clauses {
		t.Run(clauseName, func(t *testing.T) {
			clause(t, model, openDevice(t))
		})
	}
}

func contractAcceptsResetKeyStream(t *testing.T, model deck.Model, device deck.DeckDevice) {
	t.Helper()
	defer closeQuietly(t, device)

	writeError := device.Write(deck.EncodeResetKeyStream(model))
	if writeError != nil {
		t.Fatalf("want the reset key stream accepted, got %v", writeError)
	}
}

func contractAcceptsBrightness(t *testing.T, model deck.Model, device deck.DeckDevice) {
	t.Helper()
	defer closeQuietly(t, device)

	brightness, encodeError := deck.EncodeBrightness(model, contractBrightness)
	if encodeError != nil {
		t.Fatalf("encoding: %v", encodeError)
	}

	featureError := device.SendFeatureReport(brightness)
	if featureError != nil {
		t.Fatalf("want brightness accepted, got %v", featureError)
	}
}

func contractReturnsSerial(t *testing.T, model deck.Model, device deck.DeckDevice) {
	t.Helper()
	defer closeQuietly(t, device)

	report, readError := device.GetFeatureReport(model.Protocol.SerialFeatureID, model.Protocol.FeatureReportLength)
	if readError != nil {
		t.Fatalf("reading the serial report: %v", readError)
	}

	_, decodeError := deck.DecodeSerialNumber(model, report)
	if decodeError != nil {
		t.Fatalf("want a decodable serial report, got %v", decodeError)
	}
}

func contractReadsWholeReports(t *testing.T, model deck.Model, device deck.DeckDevice) {
	t.Helper()
	defer closeQuietly(t, device)

	buffer := make([]byte, model.Protocol.InputReportLength)

	readLength, readError := device.ReadInputReport(buffer, contractReadTimeout)
	if readError != nil {
		t.Fatalf("want a timed read to succeed, got %v", readError)
	}

	if readLength != 0 && readLength != model.Protocol.InputReportLength {
		t.Fatalf("want 0 or %d bytes, got %d", model.Protocol.InputReportLength, readLength)
	}
}

func contractWriteDuringRead(t *testing.T, model deck.Model, device deck.DeckDevice) {
	t.Helper()
	defer closeQuietly(t, device)

	readDone := make(chan error, 1)

	go func() {
		buffer := make([]byte, model.Protocol.InputReportLength)

		_, readError := device.ReadInputReport(buffer, contractPendingReadTimeout)
		readDone <- readError
	}()

	time.Sleep(contractReadStartPause)

	writeStart := time.Now()
	writeError := device.Write(deck.EncodeResetKeyStream(model))
	writeTook := time.Since(writeStart)

	if writeError != nil {
		t.Fatalf("want the write accepted, got %v", writeError)
	}

	if writeTook > contractWriteBound {
		t.Errorf("want a write within %v while a read waits, took %v", contractWriteBound, writeTook)
	}

	readError := <-readDone
	if readError != nil {
		t.Fatalf("want the pending read to end cleanly, got %v", readError)
	}
}

func contractRefusesUseAfterClose(t *testing.T, model deck.Model, device deck.DeckDevice) {
	t.Helper()

	firstClose := device.Close()
	secondClose := device.Close()

	if firstClose != nil || secondClose != nil {
		t.Fatalf("want both closes to succeed, got %v and %v", firstClose, secondClose)
	}

	writeError := device.Write(deck.EncodeResetKeyStream(model))
	if !errors.Is(writeError, deck.ErrDisconnected) {
		t.Fatalf("want a write after close to wrap ErrDisconnected, got %v", writeError)
	}
}

// RunDeckWatcherContract is the one contract suite for the DeckWatcher port. It
// needs no deck attached: it checks that every listed deck is a supported model
// and that opening a deck that is not attached is refused.
func RunDeckWatcherContract(t *testing.T, watcher deck.DeckWatcher) {
	t.Helper()

	t.Run("lists only supported models", func(t *testing.T) {
		connections, enumerateError := watcher.Enumerate()
		if enumerateError != nil {
			t.Fatalf("enumerating: %v", enumerateError)
		}

		for _, connection := range connections {
			if _, supported := deck.LookupModel(connection.VendorID, connection.ProductID); !supported {
				t.Errorf("listed an unsupported device %#04x:%#04x", connection.VendorID, connection.ProductID)
			}
		}
	})

	t.Run("refuses to open a deck that is not attached", func(t *testing.T) {
		absent := deck.Connection{Path: "no-such-deck-path", VendorID: FakeDeckVendorID, ProductID: FakeDeckProductID}

		device, openError := watcher.Open(absent)
		if openError == nil {
			closeQuietly(t, device)
			t.Fatal("want an error opening a deck that is not attached")
		}
	})
}

func closeQuietly(t *testing.T, device deck.DeckDevice) {
	t.Helper()

	closeError := device.Close()
	if closeError != nil {
		t.Errorf("closing: %v", closeError)
	}
}
