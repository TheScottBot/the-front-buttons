package hidtransport

import "github.com/TheScottBot/the-front-buttons/internal/deck"

// RetryTransient exposes the write retry rule to the external tests.
func RetryTransient(call func() error) error { return retryTransient(call) }

// WriteAttempts is how many times a write is tried.
const WriteAttempts = writeAttempts

// ErrHandleGone is what every handle call returns once its device is removed.
var ErrHandleGone = errHandleGone

// NewWatcherOver returns a watcher driving fake, as if fake were the only
// driver built in.
func NewWatcherOver(fake *FakeDriver) *Watcher {
	return newWatcher(DriverHIDAPI, map[DriverName]driver{DriverHIDAPI: fake})
}

// NewWatcherWithDrivers returns a watcher choosing among fakes built in under
// the names given, so the choice can be tested whatever this binary holds.
func NewWatcherWithDrivers(asked DriverName, built ...DriverName) *Watcher {
	drivers := map[DriverName]driver{}

	for _, name := range built {
		drivers[name] = NewFakeDriver()
	}

	return newWatcher(asked, drivers)
}

// TestDriver lets the external tests drive a driver directly, holding the
// process wide lock where the watcher would.
type TestDriver struct {
	inner driver
}

// AsTestDriver wraps the fake for the driver contract.
func (fake *FakeDriver) AsTestDriver() TestDriver { return TestDriver{inner: fake} }

// BuiltInDrivers wraps every driver built into this test binary.
func BuiltInDrivers() map[DriverName]TestDriver {
	wrapped := map[DriverName]TestDriver{}

	for name, built := range builtDrivers {
		wrapped[name] = TestDriver{inner: built}
	}

	return wrapped
}

// Enumerate lists the devices of one vendor.
func (wrapped TestDriver) Enumerate(vendorID uint16) ([]deck.Connection, error) {
	var listed []deck.Connection

	listError := withDriverLock(func() error {
		var enumerateError error

		listed, enumerateError = wrapped.inner.enumerate(vendorID)

		return enumerateError
	})

	return listed, listError
}

// Open opens a listed path.
func (wrapped TestDriver) Open(path string) (TestHandle, error) {
	var opened handle

	openError := withDriverLock(func() error {
		var pathError error

		opened, pathError = wrapped.inner.open(path)

		return pathError
	})

	return TestHandle{inner: opened}, openError
}

// TestHandle lets the external tests drive an open handle directly.
type TestHandle struct {
	inner handle
}

// Write sends one output report.
func (wrapped TestHandle) Write(report []byte) error { return wrapped.inner.write(report) }

// SendFeature sends one feature report.
func (wrapped TestHandle) SendFeature(report []byte) error { return wrapped.inner.sendFeature(report) }

// GetFeature reads one feature report.
func (wrapped TestHandle) GetFeature(reportID byte, length int) ([]byte, error) {
	return wrapped.inner.getFeature(reportID, length)
}

// ReadWithoutWaiting takes a waiting input report, if any.
func (wrapped TestHandle) ReadWithoutWaiting(buffer []byte) (int, error) {
	return wrapped.inner.readWithoutWaiting(buffer)
}

// Close releases the handle, holding the process wide lock.
func (wrapped TestHandle) Close() error { return withDriverLock(wrapped.inner.close) }
