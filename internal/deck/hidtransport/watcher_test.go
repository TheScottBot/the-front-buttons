package hidtransport_test

import (
	"errors"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const (
	fakeDeckPath       = "DevSrvsID:4294968000"
	unsupportedProduct = 0x0060
	otherVendorID      = 0x05AC
	shortReadTimeout   = 20 * time.Millisecond
	featureLength      = 32
	featureReportID    = 0x06
)

var errHeldElsewhere = errors.New("exclusive access held by another process")

func fakeDeck() deck.Connection {
	return deck.Connection{
		Path:      fakeDeckPath,
		VendorID:  testsupport.FakeDeckVendorID,
		ProductID: testsupport.FakeDeckProductID,
	}
}

// openFakeDeck opens the fake's deck through a watcher, closing it when the
// test ends.
//
//nolint:ireturn // the port's own type is what the tests drive
func openFakeDeck(t *testing.T, fake *hidtransport.FakeDriver) deck.DeckDevice {
	t.Helper()

	device, openError := hidtransport.NewWatcherOver(fake).Open(fakeDeck())
	if openError != nil {
		t.Fatalf("opening the fake deck: %v", openError)
	}

	t.Cleanup(func() { _ = device.Close() })

	return device
}

func TestWatcher_RetriesATransientWriteFailure_UntilAnAttemptSucceeds(t *testing.T) {
	t.Parallel()

	fake := hidtransport.NewFakeDriver(fakeDeck())
	device := openFakeDeck(t, fake)

	fake.FailWrites(hidtransport.WriteAttempts - 1)

	writeError := device.Write([]byte{0x02})
	if writeError != nil || fake.WriteCalls() != hidtransport.WriteAttempts {
		t.Fatalf("want success on attempt %d, got %v after %d calls",
			hidtransport.WriteAttempts, writeError, fake.WriteCalls())
	}
}

func TestWatcher_ReportsAWriteFailingEveryAttempt_AsDisconnected(t *testing.T) {
	t.Parallel()

	fake := hidtransport.NewFakeDriver(fakeDeck())
	device := openFakeDeck(t, fake)

	fake.FailWrites(hidtransport.WriteAttempts)

	writeError := device.Write([]byte{0x02})
	if !errors.Is(writeError, deck.ErrDisconnected) || fake.WriteCalls() != hidtransport.WriteAttempts {
		t.Fatalf("want ErrDisconnected after %d calls, got %v after %d",
			hidtransport.WriteAttempts, writeError, fake.WriteCalls())
	}
}

func TestWatcher_ReportsEveryCallOnARemovedDevice_AsDisconnected(t *testing.T) {
	t.Parallel()

	fake := hidtransport.NewFakeDriver(fakeDeck())
	device := openFakeDeck(t, fake)

	fake.Unplug(fakeDeckPath)

	_, readError := device.ReadInputReport(make([]byte, 64), shortReadTimeout)
	_, getError := device.GetFeatureReport(featureReportID, featureLength)

	calls := map[string]error{
		"write":         device.Write([]byte{0x02}),
		"send feature":  device.SendFeatureReport([]byte{0x03}),
		"get feature":   getError,
		"read":          readError,
		"close":         device.Close(),
		"write, closed": device.Write([]byte{0x02}),
	}

	for call, callError := range calls {
		if !errors.Is(callError, deck.ErrDisconnected) {
			t.Errorf("%s: want ErrDisconnected, got %v", call, callError)
		}
	}
}

func TestWatcher_ReportsAPanickingEnumeration_AsAnError_AndCarriesOn(t *testing.T) {
	t.Parallel()

	fake := hidtransport.NewFakeDriver(fakeDeck())
	fake.PanicOnEnumerate()

	watcher := hidtransport.NewWatcherOver(fake)

	for range 2 {
		connections, enumerateError := watcher.Enumerate()
		if !errors.Is(enumerateError, hidtransport.ErrEnumerationFailed) || connections != nil {
			t.Fatalf("want ErrEnumerationFailed and nothing listed, got %v and %v", enumerateError, connections)
		}
	}
}

func TestWatcher_ListsOnlySupportedModelsOfTheVendor(t *testing.T) {
	t.Parallel()

	unsupported := deck.Connection{
		Path: "unsupported", VendorID: testsupport.FakeDeckVendorID, ProductID: unsupportedProduct,
	}
	otherVendor := deck.Connection{Path: "keyboard", VendorID: otherVendorID, ProductID: testsupport.FakeDeckProductID}

	connections, enumerateError := hidtransport.NewWatcherOver(
		hidtransport.NewFakeDriver(unsupported, fakeDeck(), otherVendor)).Enumerate()
	if enumerateError != nil || len(connections) != 1 || connections[0] != fakeDeck() {
		t.Fatalf("want only the fake deck, got %v (%v)", connections, enumerateError)
	}
}

func TestWatcher_ReportsADeckThatIsListedButWillNotOpen_AsBusy(t *testing.T) {
	t.Parallel()

	fake := hidtransport.NewFakeDriver(fakeDeck())
	fake.RefuseOpen(errHeldElsewhere)

	_, openError := hidtransport.NewWatcherOver(fake).Open(fakeDeck())
	if !errors.Is(openError, deck.ErrDeckBusy) || !errors.Is(openError, errHeldElsewhere) {
		t.Fatalf("want ErrDeckBusy wrapping the refusal, got %v", openError)
	}
}

func TestWatcher_ReportsADeckThatIsNoLongerListed_AsDisconnected(t *testing.T) {
	t.Parallel()

	_, openError := hidtransport.NewWatcherOver(hidtransport.NewFakeDriver()).Open(fakeDeck())
	if !errors.Is(openError, deck.ErrDisconnected) || errors.Is(openError, deck.ErrDeckBusy) {
		t.Fatalf("want ErrDisconnected and not busy, got %v", openError)
	}
}

func TestWatcher_ClosesTheHandleOnce_AndRefusesCallsAfterwards(t *testing.T) {
	t.Parallel()

	fake := hidtransport.NewFakeDriver(fakeDeck())
	device := openFakeDeck(t, fake)

	for range 2 {
		closeError := device.Close()
		if closeError != nil {
			t.Fatalf("want closing to succeed, got %v", closeError)
		}
	}

	if fake.CloseCalls() != 1 {
		t.Fatalf("want the handle closed once, got %d", fake.CloseCalls())
	}

	_, readError := device.ReadInputReport(make([]byte, 64), shortReadTimeout)
	if !errors.Is(readError, deck.ErrDisconnected) {
		t.Fatalf("want a read after close to be ErrDisconnected, got %v", readError)
	}
}

func TestWatcher_ReadsAWaitingReport_AndTimesOutWithNothingWhenNoneComes(t *testing.T) {
	t.Parallel()

	fake := hidtransport.NewFakeDriver(fakeDeck())
	device := openFakeDeck(t, fake)
	buffer := make([]byte, 64)

	readLength, readError := device.ReadInputReport(buffer, shortReadTimeout)
	if readLength != 0 || readError != nil {
		t.Fatalf("want nothing after the timeout, got %d bytes (%v)", readLength, readError)
	}

	fake.QueueReport(fakeDeckPath, []byte{0x01, 0x00, 0x20})

	readLength, readError = device.ReadInputReport(buffer, time.Second)
	if readLength != 3 || readError != nil || buffer[0] != 0x01 {
		t.Fatalf("want the queued report, got %d bytes %v (%v)", readLength, buffer[:readLength], readError)
	}
}

func TestWatcher_GetsAFeatureReport_StartingWithItsID(t *testing.T) {
	t.Parallel()

	report, getError := openFakeDeck(t, hidtransport.NewFakeDriver(fakeDeck())).
		GetFeatureReport(featureReportID, featureLength)
	if getError != nil || len(report) != featureLength || report[0] != featureReportID {
		t.Fatalf("want a %d byte report starting %#x, got %v (%v)", featureLength, featureReportID, report, getError)
	}
}

func TestNewWatcher_UsesTheDriverAskedFor_WhenItIsBuiltIn(t *testing.T) {
	t.Parallel()

	choice := hidtransport.NewWatcherWithDrivers(hidtransport.DriverPurego,
		hidtransport.DriverHIDAPI, hidtransport.DriverPurego).DriverChoice()
	if choice.Using != hidtransport.DriverPurego || choice.Unavailable != nil || choice.FellBack() {
		t.Fatalf("want purego without falling back, got %+v", choice)
	}
}

func TestNewWatcher_FallsBackToTheOtherDriver_WhenTheOneAskedForIsNotBuiltIn(t *testing.T) {
	t.Parallel()

	cases := map[hidtransport.DriverName]hidtransport.DriverName{
		hidtransport.DriverPurego: hidtransport.DriverHIDAPI,
		hidtransport.DriverHIDAPI: hidtransport.DriverPurego,
	}

	for asked, built := range cases {
		choice := hidtransport.NewWatcherWithDrivers(asked, built).DriverChoice()
		if choice.Asked != asked || choice.Using != built || choice.Unavailable != nil || !choice.FellBack() {
			t.Errorf("asked %s with only %s built in: want a fall back to %s, got %+v", asked, built, built, choice)
		}
	}
}

func TestNewWatcher_ReportsNoDriver_WhenNoneIsBuiltIn(t *testing.T) {
	t.Parallel()

	watcher := hidtransport.NewWatcherWithDrivers(hidtransport.DriverPurego)

	choice := watcher.DriverChoice()
	if choice.Using != "" || !errors.Is(choice.Unavailable, hidtransport.ErrNoDriver) {
		t.Fatalf("want no driver and ErrNoDriver, got %+v", choice)
	}

	_, enumerateError := watcher.Enumerate()
	if !errors.Is(enumerateError, hidtransport.ErrEnumerationFailed) ||
		!errors.Is(enumerateError, hidtransport.ErrNoDriver) {
		t.Fatalf("want enumeration to fail with ErrNoDriver, got %v", enumerateError)
	}

	_, openError := watcher.Open(fakeDeck())
	if !errors.Is(openError, deck.ErrDisconnected) || !errors.Is(openError, hidtransport.ErrNoDriver) {
		t.Fatalf("want opening to fail as disconnected with ErrNoDriver, got %v", openError)
	}
}
