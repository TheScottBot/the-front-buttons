package hidtransport

import (
	"errors"
	"fmt"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

var (
	errFakeTransient = errors.New("IOHIDDeviceSetReport failed: (0xE00002BC) general error")
	errFakeNoDevice  = errors.New("no device at that path")
)

// FakeDriver is an in memory driver keeping the driver contract, so the
// watcher's own logic can be tested without a deck and the contract can be
// proven against something that keeps it.
type FakeDriver struct {
	mutex            sync.Mutex
	connections      []deck.Connection
	handles          map[string][]*fakeHandle
	panicOnEnumerate bool
	refuseOpen       error
	failingWrites    int
	writeCalls       int
	closeCalls       int
}

// NewFakeDriver returns a driver listing the given devices.
func NewFakeDriver(connections ...deck.Connection) *FakeDriver {
	return &FakeDriver{
		mutex:            sync.Mutex{},
		connections:      connections,
		handles:          map[string][]*fakeHandle{},
		panicOnEnumerate: false,
		refuseOpen:       nil,
		failingWrites:    0,
		writeCalls:       0,
		closeCalls:       0,
	}
}

// PanicOnEnumerate makes every enumeration panic, as go-hid can on macOS for a
// device with an invalid string.
func (fake *FakeDriver) PanicOnEnumerate() {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.panicOnEnumerate = true
}

// RefuseOpen makes opening fail with refusal while the device stays listed,
// as when another process holds it.
func (fake *FakeDriver) RefuseOpen(refusal error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.refuseOpen = refusal
}

// FailWrites makes the next count writes fail with IOKit's transient error.
func (fake *FakeDriver) FailWrites(count int) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.failingWrites = count
}

// Unplug removes a device: it is no longer listed, and every handle open on
// it returns errHandleGone from then on.
func (fake *FakeDriver) Unplug(path string) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	kept := fake.connections[:0]

	for _, connection := range fake.connections {
		if connection.Path != path {
			kept = append(kept, connection)
		}
	}

	fake.connections = kept

	for _, opened := range fake.handles[path] {
		opened.gone = true
	}
}

// QueueReport gives every handle open on path an input report to read.
func (fake *FakeDriver) QueueReport(path string, report []byte) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	for _, opened := range fake.handles[path] {
		opened.reports = append(opened.reports, append([]byte(nil), report...))
	}
}

// WriteCalls counts write calls on every handle, failed ones included.
func (fake *FakeDriver) WriteCalls() int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return fake.writeCalls
}

// CloseCalls counts close calls on every handle.
func (fake *FakeDriver) CloseCalls() int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return fake.closeCalls
}

func (fake *FakeDriver) enumerate(vendorID uint16) ([]deck.Connection, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	if fake.panicOnEnumerate {
		panic("invalid string from a device")
	}

	var listed []deck.Connection

	for _, connection := range fake.connections {
		if connection.VendorID == vendorID {
			listed = append(listed, connection)
		}
	}

	return listed, nil
}

//nolint:ireturn // the driver contract returns a handle interface
func (fake *FakeDriver) open(path string) (handle, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	if fake.refuseOpen != nil {
		return nil, fake.refuseOpen
	}

	for _, connection := range fake.connections {
		if connection.Path == path {
			opened := &fakeHandle{driver: fake, reports: nil, gone: false}
			fake.handles[path] = append(fake.handles[path], opened)

			return opened, nil
		}
	}

	return nil, fmt.Errorf("opening %q: %w", path, errFakeNoDevice)
}

// fakeHandle is one open device; its state is guarded by its driver's mutex.
type fakeHandle struct {
	driver  *FakeDriver
	reports [][]byte
	gone    bool
}

func (opened *fakeHandle) write(_ []byte) error {
	opened.driver.mutex.Lock()
	defer opened.driver.mutex.Unlock()

	opened.driver.writeCalls++

	if opened.gone {
		return errHandleGone
	}

	if opened.driver.failingWrites > 0 {
		opened.driver.failingWrites--

		return errFakeTransient
	}

	return nil
}

func (opened *fakeHandle) sendFeature(_ []byte) error {
	opened.driver.mutex.Lock()
	defer opened.driver.mutex.Unlock()

	if opened.gone {
		return errHandleGone
	}

	return nil
}

func (opened *fakeHandle) getFeature(reportID byte, length int) ([]byte, error) {
	opened.driver.mutex.Lock()
	defer opened.driver.mutex.Unlock()

	if opened.gone {
		return nil, errHandleGone
	}

	report := make([]byte, length)
	report[0] = reportID

	return report, nil
}

func (opened *fakeHandle) readWithoutWaiting(buffer []byte) (int, error) {
	opened.driver.mutex.Lock()
	defer opened.driver.mutex.Unlock()

	if opened.gone {
		return 0, errHandleGone
	}

	if len(opened.reports) == 0 {
		return 0, nil
	}

	next := opened.reports[0]
	opened.reports = opened.reports[1:]

	return copy(buffer, next), nil
}

func (opened *fakeHandle) close() error {
	opened.driver.mutex.Lock()
	defer opened.driver.mutex.Unlock()

	opened.driver.closeCalls++

	if opened.gone {
		return errHandleGone
	}

	return nil
}
