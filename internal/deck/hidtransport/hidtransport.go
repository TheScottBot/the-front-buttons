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

// ErrEnumerationFailed reports that listing devices failed, including the panic
// go-hid can raise on macOS for a device with an invalid string (finding TF5).
// The watcher keeps running and tries again on its next poll.
var ErrEnumerationFailed = errors.New("enumerating HID devices failed")

// Watcher is the DeckWatcher adapter: hidapi enumeration and opening. hidapi is
// initialised once, on first use.
type Watcher struct {
	initialiseOnce  sync.Once
	initialiseError error
}

// NewWatcher returns a watcher; hidapi is not touched until the first call.
func NewWatcher() *Watcher {
	return &Watcher{initialiseOnce: sync.Once{}, initialiseError: nil}
}

// Enumerate lists attached decks of supported models. Its results are named so
// the deferred recover below can set them.
//
//nolint:nonamedreturns // recover can only change a function's results through their names
func (watcher *Watcher) Enumerate() (connections []deck.Connection, enumerateError error) {
	initialiseError := watcher.initialise()
	if initialiseError != nil {
		return nil, initialiseError
	}

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

	listError := hid.Enumerate(elgatoVendorID, hid.ProductIDAny, func(information *hid.DeviceInfo) error {
		found = append(found, deck.Connection{
			Path:      information.Path,
			VendorID:  information.VendorID,
			ProductID: information.ProductID,
		})

		return nil
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
	initialiseError := watcher.initialise()
	if initialiseError != nil {
		return nil, initialiseError
	}

	handle, openError := hid.OpenPath(connection.Path)
	if openError == nil {
		return &device{mutex: sync.Mutex{}, handle: handle, closed: false}, nil
	}

	if watcher.isListed(connection) {
		return nil, fmt.Errorf("%w: %w", deck.ErrDeckBusy, openError)
	}

	return nil, fmt.Errorf("%w: %w", deck.ErrDisconnected, openError)
}

func (watcher *Watcher) initialise() error {
	watcher.initialiseOnce.Do(func() {
		initialiseError := hid.Init()
		if initialiseError != nil {
			watcher.initialiseError = fmt.Errorf("initialising hidapi: %w", initialiseError)
		}
	})

	return watcher.initialiseError
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

	_, writeError := opened.handle.Write(report)
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
	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	if opened.closed {
		return 0, fmt.Errorf("reading: %w", deck.ErrDisconnected)
	}

	readLength, readError := opened.handle.ReadWithTimeout(buffer, timeout)

	switch {
	case errors.Is(readError, hid.ErrTimeout):
		return 0, nil
	case readError != nil:
		return 0, fmt.Errorf("%w: reading: %w", deck.ErrDisconnected, readError)
	default:
		return readLength, nil
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

	closeError := opened.handle.Close()
	if closeError != nil {
		return fmt.Errorf("closing: %w", closeError)
	}

	return nil
}
