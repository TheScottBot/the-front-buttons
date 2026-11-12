package hidtransport

import (
	"errors"
	"fmt"
	"sync"
	"time"

	hid "github.com/sstallion/go-hid"

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
// waits inside hidapi holding the device's lock, which kept every write waiting
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

// ErrEnumerationFailed reports that listing devices failed, including the panic
// go-hid can raise on macOS for a device with an invalid string (finding TF5).
// The watcher keeps running and tries again on its next poll.
var ErrEnumerationFailed = errors.New("enumerating HID devices failed")

// hidapi keeps process wide state: its initialisation, and on macOS one HID
// manager that enumeration, opening and closing all use. None of it is safe
// from two threads at once, and two watchers initialising together aborted the
// process on the first run against a real deck (evaluation, deck testing). So
// every call that touches that state holds this one lock, and hidapi is
// initialised once per process, not once per watcher.
//
//nolint:gochecknoglobals // mirrors hidapi's own state, which is process wide
var hidapi struct {
	mutex       sync.Mutex
	initialised bool
	initFailure error
}

// withHIDAPI runs call holding the process wide lock, with hidapi initialised.
func withHIDAPI(call func() error) error {
	hidapi.mutex.Lock()
	defer hidapi.mutex.Unlock()

	if !hidapi.initialised {
		hidapi.initialised = true

		initialiseError := hid.Init()
		if initialiseError != nil {
			hidapi.initFailure = fmt.Errorf("initialising hidapi: %w", initialiseError)
		}
	}

	if hidapi.initFailure != nil {
		return hidapi.initFailure
	}

	return call()
}

// Watcher is the DeckWatcher adapter: hidapi enumeration and opening.
type Watcher struct{}

// NewWatcher returns a watcher; hidapi is not touched until the first call.
func NewWatcher() *Watcher {
	return &Watcher{}
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
			enumerateError = fmt.Errorf("%w: hidapi panicked: %v", ErrEnumerationFailed, recovered)
		}
	}()

	var found []deck.Connection

	listError := withHIDAPI(func() error {
		return hid.Enumerate(elgatoVendorID, hid.ProductIDAny, func(information *hid.DeviceInfo) error {
			found = append(found, deck.Connection{
				Path:      information.Path,
				VendorID:  information.VendorID,
				ProductID: information.ProductID,
			})

			return nil
		})
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
	var handle *hid.Device

	openError := withHIDAPI(func() error {
		var pathError error

		handle, pathError = hid.OpenPath(connection.Path)
		if pathError != nil {
			return fmt.Errorf("opening the path: %w", pathError)
		}

		return nil
	})
	if openError == nil {
		return &device{mutex: sync.Mutex{}, handle: handle, closed: false}, nil
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
	handle *hid.Device
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
		_, attemptError := opened.handle.Write(report)
		if attemptError != nil {
			return fmt.Errorf("attempt: %w", attemptError)
		}

		return nil
	})
	if writeError != nil {
		return fmt.Errorf("%w: writing: %w", deck.ErrDisconnected, writeError)
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

	_, sendError := opened.handle.SendFeatureReport(report)
	if sendError != nil {
		return fmt.Errorf("%w: sending a feature report: %w", deck.ErrDisconnected, sendError)
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

	buffer := make([]byte, length)
	buffer[0] = reportID

	readLength, readError := opened.handle.GetFeatureReport(buffer)
	if readError != nil {
		return nil, fmt.Errorf("%w: reading a feature report: %w", deck.ErrDisconnected, readError)
	}

	return buffer[:readLength], nil
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

// Close releases the deck once; hidapi must never be given a closed handle.
func (opened *device) Close() error {
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return nil
	}

	opened.closed = true

	closeError := withHIDAPI(opened.handle.Close)
	if closeError != nil {
		return fmt.Errorf("closing: %w", closeError)
	}

	return nil
}

// readWithoutWaiting takes a report hidapi already holds, if any, holding the
// device's lock only for that.
func (opened *device) readWithoutWaiting(buffer []byte) (int, error) {
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return 0, fmt.Errorf("reading: %w", deck.ErrDisconnected)
	}

	readLength, readError := opened.handle.ReadWithTimeout(buffer, 0)

	switch {
	case errors.Is(readError, hid.ErrTimeout):
		return 0, nil
	case readError != nil:
		return 0, fmt.Errorf("%w: reading: %w", deck.ErrDisconnected, readError)
	default:
		return readLength, nil
	}
}
