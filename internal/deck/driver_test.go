package deck_test

import (
	"errors"
	"image"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const fakeDeckPath = "fake-deck-1"

func TestDeckDeviceTestDouble_PassesTheDeckDeviceContract(t *testing.T) {
	t.Parallel()

	model := xlModel(t)

	testsupport.RunDeckDeviceContract(t, model, func(*testing.T) deck.DeckDevice {
		return testsupport.NewDeckDeviceTestDouble(model)
	})
}

func TestDeckWatcherTestDouble_PassesTheDeckWatcherContract(t *testing.T) {
	t.Parallel()

	watcher := testsupport.NewDeckWatcherTestDouble()
	watcher.Attach(fakeDeckPath, testsupport.NewDeckDeviceTestDouble(xlModel(t)))

	testsupport.RunDeckWatcherContract(t, watcher)
}

func openFakeDeck(t *testing.T) (*deck.Deck, *testsupport.DeckDeviceTestDouble) {
	t.Helper()

	device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
	watcher := testsupport.NewDeckWatcherTestDouble()
	watcher.Attach(fakeDeckPath, device)

	opened, openError := deck.OpenDeck(watcher, deck.Connection{
		Path: fakeDeckPath, VendorID: testsupport.FakeDeckVendorID, ProductID: testsupport.FakeDeckProductID,
	})
	if openError != nil {
		t.Fatalf("opening: %v", openError)
	}

	return opened, device
}

func TestOpenDeck_ResetsTheKeyStreamBeforeAnythingElse(t *testing.T) {
	t.Parallel()

	_, device := openFakeDeck(t)

	writes := device.Writes()
	if len(writes) != 1 || writes[0][0] != 0x02 || len(writes[0]) != 1024 {
		t.Fatalf("want exactly the reset key stream report written on open, got %d writes", len(writes))
	}
}

func TestOpenDeck_RefusesAnUnsupportedModel(t *testing.T) {
	t.Parallel()

	watcher := testsupport.NewDeckWatcherTestDouble()

	_, openError := deck.OpenDeck(watcher, deck.Connection{Path: fakeDeckPath, VendorID: 0x1234, ProductID: 0x5678})
	if !errors.Is(openError, deck.ErrUnsupportedModel) {
		t.Fatalf("want ErrUnsupportedModel, got %v", openError)
	}
}

func TestDeck_SetKeyImage_WritesEveryPageForTheKey(t *testing.T) {
	t.Parallel()

	opened, device := openFakeDeck(t)
	keyPixels := opened.Model().Protocol.KeyPixels

	imageError := opened.SetKeyImage(7, image.NewRGBA(image.Rect(0, 0, keyPixels, keyPixels)))
	if imageError != nil {
		t.Fatalf("setting a key image: %v", imageError)
	}

	if drawn := device.DrawnKeys(); drawn[7] != 1 || len(drawn) != 1 {
		t.Fatalf("want key 7 drawn once, got %v", drawn)
	}
}

func TestDeck_SetBrightness_SendsTheFeatureReport(t *testing.T) {
	t.Parallel()

	opened, device := openFakeDeck(t)

	brightnessError := opened.SetBrightness(60)
	if brightnessError != nil {
		t.Fatalf("setting brightness: %v", brightnessError)
	}

	features := device.Features()
	if len(features) != 1 || features[0][1] != 0x08 || features[0][2] != 60 {
		t.Fatalf("want one brightness report at 60, got %v", features)
	}
}

func TestDeck_ReadKeyChanges_ReportsPressAndRelease(t *testing.T) {
	t.Parallel()

	opened, device := openFakeDeck(t)

	device.PressKeys(31)
	device.PressKeys()

	pressed, pressError := opened.ReadKeyChanges(time.Second)
	released, releaseError := opened.ReadKeyChanges(time.Second)

	if pressError != nil || releaseError != nil {
		t.Fatalf("reading: %v %v", pressError, releaseError)
	}

	if len(pressed) != 1 || pressed[0] != (deck.KeyChange{KeyIndex: 31, Pressed: true}) {
		t.Fatalf("want key 31 pressed, got %v", pressed)
	}

	if len(released) != 1 || released[0] != (deck.KeyChange{KeyIndex: 31, Pressed: false}) {
		t.Fatalf("want key 31 released, got %v", released)
	}
}

func TestDeck_ReadKeyChanges_ReturnsNothingWhenNoReportArrives(t *testing.T) {
	t.Parallel()

	opened, _ := openFakeDeck(t)

	changes, readError := opened.ReadKeyChanges(10 * time.Millisecond)
	if readError != nil || len(changes) != 0 {
		t.Fatalf("want no changes and no error, got %v (%v)", changes, readError)
	}
}

func TestDeck_ReportsDisconnection_WhenTheDeckGoesAway(t *testing.T) {
	t.Parallel()

	opened, device := openFakeDeck(t)
	device.Unplug()

	_, readError := opened.ReadKeyChanges(time.Second)
	keyPixels := opened.Model().Protocol.KeyPixels
	writeError := opened.SetKeyImage(0, image.NewRGBA(image.Rect(0, 0, keyPixels, keyPixels)))

	if !errors.Is(readError, deck.ErrDisconnected) || !errors.Is(writeError, deck.ErrDisconnected) {
		t.Fatalf("want both to wrap ErrDisconnected, got %v and %v", readError, writeError)
	}
}

func TestDeck_SerialNumber_IsReadFromItsFeatureReport(t *testing.T) {
	t.Parallel()

	opened, _ := openFakeDeck(t)

	serial, serialError := opened.SerialNumber()
	if serialError != nil || serial != testsupport.FakeDeckSerial {
		t.Fatalf("want %q, got %q (%v)", testsupport.FakeDeckSerial, serial, serialError)
	}
}

func TestDiffConnections_ReportsArrivalsAndDepartures_ByPath(t *testing.T) {
	t.Parallel()

	xl := func(path string) deck.Connection {
		return deck.Connection{Path: path, VendorID: testsupport.FakeDeckVendorID, ProductID: testsupport.FakeDeckProductID}
	}

	changes := deck.DiffConnections([]deck.Connection{xl("a"), xl("b")}, []deck.Connection{xl("b"), xl("c")})

	if len(changes.Connected) != 1 || changes.Connected[0].Path != "c" {
		t.Fatalf("want c connected, got %v", changes.Connected)
	}

	if len(changes.Disconnected) != 1 || changes.Disconnected[0].Path != "a" {
		t.Fatalf("want a disconnected, got %v", changes.Disconnected)
	}
}

func TestSupportedConnections_DropsDevicesOutsideTheModelTable(t *testing.T) {
	t.Parallel()

	connections := []deck.Connection{
		{Path: "xl", VendorID: testsupport.FakeDeckVendorID, ProductID: testsupport.FakeDeckProductID},
		{Path: "keyboard", VendorID: 0x05AC, ProductID: 0x0250},
	}

	supported := deck.SupportedConnections(connections)
	if len(supported) != 1 || supported[0].Path != "xl" {
		t.Fatalf("want only the XL, got %v", supported)
	}
}
