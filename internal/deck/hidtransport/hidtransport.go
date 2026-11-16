package hidtransport

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// elgatoVendorID narrows enumeration to Elgato's devices; the model table then
// keeps only supported models.
const elgatoVendorID = 0x0FD9

// On macOS a write to the deck is now and then refused with IOKit's general
// error (0xE00002BC) while the deck is plainly still attached: about one write in
// a thousand on the first run against a real deck, each of which the daemon took
// for an unplugging, dropping and redrawing the page. A write is tried this many
// times, this far apart, before it counts as a failure; a deck that has really
// gone fails every attempt within a few milliseconds.
const (
	writeAttempts   = 3
	writeRetryPause = 2 * time.Millisecond
)

// readPollInterval is how often a timed read looks for a report. A read never
// waits inside the driver holding the device's lock, which kept every write waiting
// out the read in progress (evaluation, deck testing); it asks without waiting,
// then sleeps unlocked. Five milliseconds adds no delay a person could feel.
const readPollInterval = 5 * time.Millisecond

// retryTransient makes up to writeAttempts calls, stopping at the first that
// succeeds, and returns the last error if none does.
func retryTransient(call func() error) error {
	var callError error

	for attempt := range writeAttempts {
		if attempt > 0 {
			time.Sleep(writeRetryPause)
		}

		callError = call()
		if callError == nil {
			return nil
		}
	}

	return callError
}

// ErrEnumerationFailed reports that listing devices failed, including a panic
// in the driver, which go-hid can raise on macOS for a device with an invalid
// string (finding TF5). The watcher keeps running and tries again on its next
// poll.
var ErrEnumerationFailed = errors.New("enumerating HID devices failed")

// HID drivers keep process wide state: hidapi its initialisation and, on macOS,
// one HID manager that enumeration, opening and closing all use. None of it is
// safe from two threads at once, and two watchers initialising together
// aborted the process on the first run against a real deck (evaluation, deck
// testing). So every enumerate, open and close, through any driver, holds this
// one lock.
//
//nolint:gochecknoglobals // mirrors the drivers' own state, which is process wide
var driverLock sync.Mutex

// withDriverLock runs call holding the process wide lock. The lock is released
// by a deferred call, so a panic in the driver leaves it free.
func withDriverLock(call func() error) error {
	driverLock.Lock()
	defer driverLock.Unlock()

	return call()
}

// disconnected reports a failed call on an open deck, errHandleGone included,
// as the deck being gone.
func disconnected(action string, cause error) error {
	return fmt.Errorf("%w: %s: %w", deck.ErrDisconnected, action, cause)
}

// Watcher is the DeckWatcher adapter: enumeration and opening through one HID
// driver.
type Watcher struct {
	choice DriverChoice
	driver driver
}

// NewWatcher returns a watcher using the driver named if this build holds it,
// and otherwise another (DriverChoice says which). No driver is touched until
// the first call.
func NewWatcher(name DriverName) *Watcher {
	return newWatcher(name, builtDrivers)
}

func newWatcher(name DriverName, built map[DriverName]driver) *Watcher {
	using, unavailable := chooseDriver(name, built)

	return &Watcher{
		choice: DriverChoice{Asked: name, Using: using, Unavailable: unavailable},
		driver: built[using],
	}
}

// DriverChoice says which driver was asked for and which is used.
func (watcher *Watcher) DriverChoice() DriverChoice {
	return watcher.choice
}

// Enumerate lists attached decks of supported models. Its results are named so
// the deferred recover below can set them.
//
//nolint:nonamedreturns // recover can only change a function's results through their names
func (watcher *Watcher) Enumerate() (connections []deck.Connection, enumerateError error) {
	// TF5: go-hid's Enumerate can panic on macOS when a device reports an
	// invalid string. A panic here would take the whole daemon down over a
	// device that is not even a deck, so it is caught and reported instead.
	defer func() {
		if recovered := recover(); recovered != nil {
			connections = nil
			enumerateError = fmt.Errorf("%w: the %s driver panicked: %v",
				ErrEnumerationFailed, watcher.choice.Using, recovered)
		}
	}()

	if watcher.driver == nil {
		return nil, fmt.Errorf("%w: %w", ErrEnumerationFailed, watcher.choice.Unavailable)
	}

	var found []deck.Connection

	listError := withDriverLock(func() error {
		var driverError error

		found, driverError = watcher.driver.enumerate(elgatoVendorID)

		return driverError
	})
	if listError != nil {
		return nil, fmt.Errorf("%w: %w", ErrEnumerationFailed, listError)
	}

	return deck.SupportedConnections(found), nil
}

// Open opens a listed deck. If the deck is still listed but will not open,
// something else holds it, which is reported as busy; if it is gone, as
// disconnected.
//
//nolint:ireturn // implements the port, which returns a device interface
func (watcher *Watcher) Open(connection deck.Connection) (deck.DeckDevice, error) {
	if watcher.driver == nil {
		return nil, fmt.Errorf("%w: %w", deck.ErrDisconnected, watcher.choice.Unavailable)
	}

	var opened handle

	openError := withDriverLock(func() error {
		var pathError error

		opened, pathError = watcher.driver.open(connection.Path)

		return pathError
	})
	if openError == nil {
		return &device{mutex: sync.Mutex{}, handle: opened, closed: false}, nil
	}

	if watcher.isListed(connection) {
		return nil, fmt.Errorf("%w: %w", deck.ErrDeckBusy, openError)
	}

	return nil, fmt.Errorf("%w: %w", deck.ErrDisconnected, openError)
}

func (watcher *Watcher) isListed(connection deck.Connection) bool {
	connections, enumerateError := watcher.Enumerate()
	if enumerateError != nil {
		return false
	}

	for _, listed := range connections {
		if listed.Path == connection.Path {
			return true
		}
	}

	return false
}

// device is the DeckDevice adapter for one open deck. hidapi does not promise
// that one device handle is safe to use from several threads, so every call is
// serialised, as the reference library does with its own mutex.
type device struct {
	mutex  sync.Mutex
	handle handle
	closed bool
}

// Write sends one output report.
func (opened *device) Write(report []byte) error {
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return fmt.Errorf("writing: %w", deck.ErrDisconnected)
	}

	writeError := retryTransient(func() error {
		attemptError := opened.handle.write(report)
		if attemptError != nil {
			return fmt.Errorf("attempt: %w", attemptError)
		}

		return nil
	})
	if writeError != nil {
		return disconnected("writing", writeError)
	}

	return nil
}

// SendFeatureReport sends one feature report.
func (opened *device) SendFeatureReport(report []byte) error {
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return fmt.Errorf("sending a feature report: %w", deck.ErrDisconnected)
	}

	sendError := opened.handle.sendFeature(report)
	if sendError != nil {
		return disconnected("sending a feature report", sendError)
	}

	return nil
}

// GetFeatureReport reads one feature report. Whether macOS needs the reference's
// extra byte with the bundled hidapi 0.15.0 is unverified until experiment 1
// (evaluation 3.6); the decoder accepts either length.
func (opened *device) GetFeatureReport(reportID byte, length int) ([]byte, error) {
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return nil, fmt.Errorf("reading a feature report: %w", deck.ErrDisconnected)
	}

	report, readError := opened.handle.getFeature(reportID, length)
	if readError != nil {
		return nil, disconnected("reading a feature report", readError)
	}

	return report, nil
}

// ReadInputReport waits up to timeout for one input report.
func (opened *device) ReadInputReport(buffer []byte, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)

	for {
		readLength, readError := opened.readWithoutWaiting(buffer)
		if readError != nil || readLength > 0 {
			return readLength, readError
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, nil
		}

		time.Sleep(min(readPollInterval, remaining))
	}
}

// Close releases the deck once; a driver must never be given a closed handle.
// A deck already removed is reported as disconnected, once released.
func (opened *device) Close() error {
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return nil
	}

	opened.closed = true

	closeError := withDriverLock(opened.handle.close)

	switch {
	case errors.Is(closeError, errHandleGone):
		return disconnected("closing", closeError)
	case closeError != nil:
		return fmt.Errorf("closing: %w", closeError)
	default:
		return nil
	}
}

// readWithoutWaiting takes a report the driver already holds, if any, holding
// the device's lock only for that.
func (opened *device) readWithoutWaiting(buffer []byte) (int, error) {
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return 0, fmt.Errorf("reading: %w", deck.ErrDisconnected)
	}

	readLength, readError := opened.handle.readWithoutWaiting(buffer)
	if readError != nil {
		return 0, disconnected("reading", readError)
	}

	return readLength, nil
}
