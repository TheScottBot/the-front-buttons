package hidtransport_test

import (
	"errors"
	"os"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const (
	// absentVendorID is a vendor no attached device has. It is not zero,
	// which hidapi reads as any vendor.
	absentVendorID       = 0xFFFE
	absentPath           = "no-such-device-path"
	contractReadBuffer   = 512
	contractBrightness   = 70
	unpluggedReadBuffers = 64
)

// driverContractSubject is a driver and what the contract needs to drive it.
type driverContractSubject struct {
	driver   hidtransport.TestDriver
	vendorID uint16
	// output and feature are reports the device accepts.
	output          []byte
	feature         []byte
	featureReportID byte
	featureLength   int
	// unplug removes the opened device; nil when only a person can.
	unplug func(path string)
}

// runDriverContract checks what the watcher relies on from any driver:
// enumeration filtered by vendor, opening, reports out, feature reports, a
// read that never waits, closing, and errHandleGone from every call once the
// device is removed.
func runDriverContract(t *testing.T, subject driverContractSubject) {
	t.Helper()

	path := checkEnumeration(t, subject)
	checkOpenRefusesAbsentPath(t, subject)

	opened, openError := subject.driver.Open(path)
	if openError != nil {
		t.Fatalf("opening a listed path: %v", openError)
	}

	checkReports(t, subject, opened)

	closeError := opened.Close()
	if closeError != nil {
		t.Fatalf("closing: %v", closeError)
	}

	if subject.unplug == nil {
		t.Log("removal not checked: only unplugging the device can remove it")

		return
	}

	checkRemoval(t, subject, path)
}

// checkEnumeration wants the vendor's devices and only them, and nothing for
// a vendor with none attached; it returns the first supported deck's path.
func checkEnumeration(t *testing.T, subject driverContractSubject) string {
	t.Helper()

	connections, enumerateError := subject.driver.Enumerate(subject.vendorID)
	if enumerateError != nil || len(connections) == 0 {
		t.Fatalf("enumerating vendor %#04x: want a device, got %v (%v)", subject.vendorID, connections, enumerateError)
	}

	for _, connection := range connections {
		if connection.VendorID != subject.vendorID || connection.Path == "" {
			t.Errorf("enumerating vendor %#04x listed %+v", subject.vendorID, connection)
		}
	}

	absent, absentError := subject.driver.Enumerate(absentVendorID)
	if absentError != nil || len(absent) != 0 {
		t.Errorf("enumerating vendor %#04x: want nothing, got %v (%v)", absentVendorID, absent, absentError)
	}

	supported := deck.SupportedConnections(connections)
	if len(supported) == 0 {
		t.Fatalf("enumerating vendor %#04x: want a supported deck, got %v", subject.vendorID, connections)
	}

	return supported[0].Path
}

func checkOpenRefusesAbsentPath(t *testing.T, subject driverContractSubject) {
	t.Helper()

	opened, openError := subject.driver.Open(absentPath)
	if openError == nil {
		_ = opened.Close()

		t.Errorf("want an error opening %q", absentPath)
	}
}

func checkReports(t *testing.T, subject driverContractSubject, opened hidtransport.TestHandle) {
	t.Helper()

	writeError := opened.Write(subject.output)
	if writeError != nil {
		t.Errorf("writing an output report: %v", writeError)
	}

	sendError := opened.SendFeature(subject.feature)
	if sendError != nil {
		t.Errorf("sending a feature report: %v", sendError)
	}

	report, getError := opened.GetFeature(subject.featureReportID, subject.featureLength)
	if getError != nil || len(report) == 0 || len(report) > subject.featureLength || report[0] != subject.featureReportID {
		t.Errorf("getting feature report %#x: want at most %d bytes starting with its ID, got %v (%v)",
			subject.featureReportID, subject.featureLength, report, getError)
	}

	readLength, readError := opened.ReadWithoutWaiting(make([]byte, contractReadBuffer))
	if readLength != 0 || readError != nil {
		t.Errorf("reading with no report waiting: want 0 and no error, got %d (%v)", readLength, readError)
	}
}

// checkRemoval opens the device again, removes it, and wants errHandleGone
// from every call on the handle.
func checkRemoval(t *testing.T, subject driverContractSubject, path string) {
	t.Helper()

	opened, openError := subject.driver.Open(path)
	if openError != nil {
		t.Fatalf("opening again before removal: %v", openError)
	}

	subject.unplug(path)

	_, getError := opened.GetFeature(subject.featureReportID, subject.featureLength)
	_, readError := opened.ReadWithoutWaiting(make([]byte, unpluggedReadBuffers))

	calls := map[string]error{
		"write":        opened.Write(subject.output),
		"send feature": opened.SendFeature(subject.feature),
		"get feature":  getError,
		"read":         readError,
		"close":        opened.Close(),
	}

	for call, callError := range calls {
		if !errors.Is(callError, hidtransport.ErrHandleGone) {
			t.Errorf("%s after removal: want errHandleGone, got %v", call, callError)
		}
	}
}

func TestDriverContract_HoldsForTheFakeDriver(t *testing.T) {
	t.Parallel()

	otherVendor := deck.Connection{Path: "keyboard", VendorID: otherVendorID, ProductID: testsupport.FakeDeckProductID}
	fake := hidtransport.NewFakeDriver(otherVendor, fakeDeck())

	runDriverContract(t, driverContractSubject{
		driver:          fake.AsTestDriver(),
		vendorID:        testsupport.FakeDeckVendorID,
		output:          []byte{0x02},
		feature:         []byte{0x03},
		featureReportID: featureReportID,
		featureLength:   featureLength,
		unplug:          fake.Unplug,
	})
}

// TestDriverContract_HoldsForEachBuiltInDriver_OnAnAttachedDeck runs the
// contract against every driver in this build with a real deck. It writes to
// the deck (resetting its key stream, setting its brightness), so it runs
// only with the experiments, with nothing else holding the deck and no key
// pressed. Removal is left out: only unplugging the deck can check it.
//
//nolint:paralleltest // drives the one physical deck, which two drivers cannot share
func TestDriverContract_HoldsForEachBuiltInDriver_OnAnAttachedDeck(t *testing.T) {
	if os.Getenv(experimentVariable) != "1" {
		t.Skipf("set %s=1 with a deck attached, nothing else holding it and a person at it", experimentVariable)
	}

	for name, driver := range hidtransport.BuiltInDrivers() {
		t.Run(string(name), func(t *testing.T) { //nolint:paralleltest // one physical deck
			runDriverContract(t, attachedDeckSubject(t, driver))
		})
	}
}

func attachedDeckSubject(t *testing.T, driver hidtransport.TestDriver) driverContractSubject {
	t.Helper()

	connections, enumerateError := driver.Enumerate(testsupport.FakeDeckVendorID)

	supported := deck.SupportedConnections(connections)
	if enumerateError != nil || len(supported) == 0 {
		t.Fatalf("want an attached deck, found %d (%v)", len(supported), enumerateError)
	}

	model, _ := deck.LookupModel(supported[0].VendorID, supported[0].ProductID)

	brightness, encodeError := deck.EncodeBrightness(model, contractBrightness)
	if encodeError != nil {
		t.Fatal(encodeError)
	}

	return driverContractSubject{
		driver:          driver,
		vendorID:        supported[0].VendorID,
		output:          deck.EncodeResetKeyStream(model),
		feature:         brightness,
		featureReportID: model.Protocol.SerialFeatureID,
		featureLength:   model.Protocol.FeatureReportLength,
		unplug:          nil,
	}
}
