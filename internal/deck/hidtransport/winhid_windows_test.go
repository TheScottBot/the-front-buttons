package hidtransport //nolint:testpackage // reads the driver's unexported state: its lengths, its pending read

import (
	"os"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// The Windows driver's own calls, on Windows. The first tests need no device;
// those that open the deck run only with THE_FRONT_BUTTONS_EXPERIMENT=1, a
// deck attached and nothing else holding it. The driver contract
// (driver_contract_test.go) also runs against this driver under the same
// variable.

const (
	winhidExperimentVariable = "THE_FRONT_BUTTONS_EXPERIMENT"
	winhidAbsentVendor       = 0xFFFE
	winhidInterfaceClass     = "{4d1e55b2-f16f-11cf-88cb-001111000030}"
	winhidAbsentPath         = `\\?\hid#vid_fffe&pid_fffe#no-such-device#` + winhidInterfaceClass
)

func TestWinHID_IsBuiltInAsThePuregoDriver(t *testing.T) {
	t.Parallel()

	if _, isWinHID := builtDrivers[DriverPurego].(*winhidDriver); !isWinHID {
		t.Errorf("want the purego driver to be the Windows driver, got %T", builtDrivers[DriverPurego])
	}
}

func TestWinHID_FindsEveryFunctionItCalls(t *testing.T) {
	t.Parallel()

	procs, loadError := loadWinHIDProcs()
	if loadError != nil || procs == nil {
		t.Fatalf("want hid.dll and setupapi.dll to hold every function, got %v", loadError)
	}
}

//nolint:paralleltest // enumeration holds the process wide driver lock
func TestWinHID_EnumeratesNothingForAVendorWithNothingAttached(t *testing.T) {
	var listed []deck.Connection

	listError := withDriverLock(func() error {
		var enumerateError error

		listed, enumerateError = (&winhidDriver{}).enumerate(winhidAbsentVendor)

		return enumerateError
	})
	if listError != nil || len(listed) != 0 {
		t.Errorf("want nothing and no error, got %v (%v)", listed, listError)
	}
}

//nolint:paralleltest // opening holds the process wide driver lock
func TestWinHID_RefusesToOpenAnAbsentPath(t *testing.T) {
	openError := withDriverLock(func() error {
		opened, pathError := (&winhidDriver{}).open(winhidAbsentPath)
		if pathError == nil {
			_ = opened.close()
		}

		return pathError
	})
	if openError == nil {
		t.Error("want an error opening an absent path")
	}
}

// openWinHIDDeck skips unless experiments were asked for, and otherwise opens
// the first attached deck through this driver, closing it when the test ends.
func openWinHIDDeck(t *testing.T) (deck.Connection, deck.Model, *winhidHandle) {
	t.Helper()

	if os.Getenv(winhidExperimentVariable) != "1" {
		t.Skipf("set %s=1 with a deck attached and nothing else holding it", winhidExperimentVariable)
	}

	driver := &winhidDriver{}

	var connections []deck.Connection

	listError := withDriverLock(func() error {
		var enumerateError error

		connections, enumerateError = driver.enumerate(elgatoVendorID)

		return enumerateError
	})

	supported := deck.SupportedConnections(connections)
	if listError != nil || len(supported) == 0 {
		t.Fatalf("want an attached deck, found %d (%v)", len(supported), listError)
	}

	model, _ := deck.LookupModel(supported[0].VendorID, supported[0].ProductID)

	var opened handle

	openError := withDriverLock(func() error {
		var pathError error

		opened, pathError = driver.open(supported[0].Path)

		return pathError
	})
	if openError != nil {
		t.Fatalf("opening %q: %v", supported[0].Path, openError)
	}

	winhid, isWinHID := opened.(*winhidHandle)
	if !isWinHID {
		t.Fatalf("want a Windows handle, got %T", opened)
	}

	t.Cleanup(func() {
		closeError := withDriverLock(winhid.close)
		if closeError != nil {
			t.Errorf("closing: %v", closeError)
		}
	})

	return supported[0], model, winhid
}

//nolint:paralleltest // drives the one physical deck
func TestWinHID_OnTheDeck_PathIsAHIDInterfacePath(t *testing.T) {
	connection, _, _ := openWinHIDDeck(t)

	lower := strings.ToLower(connection.Path)
	if !strings.HasPrefix(lower, `\\?\hid#vid_0fd9&pid_`) || !strings.HasSuffix(lower, winhidInterfaceClass) {
		t.Errorf("want a HID interface path, as hidapi lists, got %q", connection.Path)
	}
}

//nolint:paralleltest // drives the one physical deck
func TestWinHID_OnTheDeck_ReportLengthsMatchTheModel(t *testing.T) {
	_, model, opened := openWinHIDDeck(t)

	want := reportLengths{
		input:   model.Protocol.InputReportLength,
		output:  model.Protocol.OutputReportLength,
		feature: model.Protocol.FeatureReportLength,
	}
	if opened.lengths != want {
		t.Errorf("want HidP_GetCaps to give the model's lengths %+v, got %+v", want, opened.lengths)
	}
}

//nolint:paralleltest // drives the one physical deck
func TestWinHID_OnTheDeck_SerialFeatureReportStartsWithItsID(t *testing.T) {
	_, model, opened := openWinHIDDeck(t)

	protocol := model.Protocol

	report, getError := opened.getFeature(protocol.SerialFeatureID, protocol.FeatureReportLength)
	if getError != nil || len(report) != protocol.FeatureReportLength || report[0] != protocol.SerialFeatureID {
		t.Fatalf("want %d bytes starting with %#x, got %v (%v)",
			protocol.FeatureReportLength, protocol.SerialFeatureID, report, getError)
	}

	serial, decodeError := deck.DecodeSerialNumber(model, report)
	if decodeError != nil || serial == "" {
		t.Errorf("want the serial number decoded, got %q (%v)", serial, decodeError)
	}
}

//nolint:paralleltest // drives the one physical deck
func TestWinHID_OnTheDeck_ReadKeepsOneReadPendingWithNoKeyPressed(t *testing.T) {
	_, model, opened := openWinHIDDeck(t)

	for range 3 {
		readLength, readError := opened.readWithoutWaiting(make([]byte, model.Protocol.InputReportLength))
		if readLength != 0 || readError != nil {
			t.Fatalf("want 0 and no error with no key pressed, got %d (%v)", readLength, readError)
		}
	}

	if !opened.pending {
		t.Error("want the read left pending")
	}
}

// TestWinHID_OnTheDeck_PathsMatchHIDAPI lists the deck through this driver and
// through hidapi, when this build holds hidapi, and wants the same paths, and
// each driver's path to open with the other.
//
//nolint:paralleltest // drives the one physical deck
func TestWinHID_OnTheDeck_PathsMatchHIDAPI(t *testing.T) {
	if os.Getenv(winhidExperimentVariable) != "1" {
		t.Skipf("set %s=1 with a deck attached and nothing else holding it", winhidExperimentVariable)
	}

	hidapi, built := builtDrivers[DriverHIDAPI]
	if !built {
		t.Skip("this build holds no hidapi driver to compare with; build with cgo")
	}

	ours := listWith(t, &winhidDriver{})
	theirs := listWith(t, hidapi)

	if len(ours) == 0 || len(ours) != len(theirs) {
		t.Fatalf("want the same devices listed, got %v and from hidapi %v", ours, theirs)
	}

	for index := range ours {
		if ours[index] != theirs[index] {
			t.Errorf("want the same connection, got %+v and from hidapi %+v", ours[index], theirs[index])
		}
	}

	opensWith(t, hidapi, ours[0].Path)
	opensWith(t, &winhidDriver{}, theirs[0].Path)
}

// opensWith wants path to open, and close, with opener.
func opensWith(t *testing.T, opener driver, path string) {
	t.Helper()

	openError := withDriverLock(func() error {
		opened, pathError := opener.open(path)
		if pathError != nil {
			return pathError
		}

		return opened.close()
	})
	if openError != nil {
		t.Errorf("want %q to open with the other driver, got %v", path, openError)
	}
}

func listWith(t *testing.T, listing driver) []deck.Connection {
	t.Helper()

	var connections []deck.Connection

	listError := withDriverLock(func() error {
		var enumerateError error

		connections, enumerateError = listing.enumerate(elgatoVendorID)

		return enumerateError
	})
	if listError != nil {
		t.Fatalf("enumerating: %v", listError)
	}

	return deck.SupportedConnections(connections)
}
