package testsupport

import (
	"fmt"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// The fake deck's identity: the first XL row of the model table.
const (
	FakeDeckVendorID  = 0x0FD9
	FakeDeckProductID = 0x006C
	FakeDeckSerial    = "FAKE00001"
)

// imageCommandPosition and the values below read a key image page's header in a
// recorded write, so tests can count which keys were drawn.
const (
	imageCommandPosition = 1
	imageKeyPosition     = 2
	imageLastPosition    = 3
)

// DeckDeviceTestDouble is a fake open deck. It records every report written,
// replays scripted input reports, answers feature reads, and can be made to
// disconnect, including midway through a run of writes.
type DeckDeviceTestDouble struct {
	mutex             sync.Mutex
	model             deck.Model
	writes            [][]byte
	features          [][]byte
	pendingReports    [][]byte
	reportArrived     chan struct{}
	closed            bool
	disconnectAfter   int
	writesBeforeError int
}

// NewDeckDeviceTestDouble returns a fake XL that never disconnects on its own.
func NewDeckDeviceTestDouble(model deck.Model) *DeckDeviceTestDouble {
	return &DeckDeviceTestDouble{
		mutex:             sync.Mutex{},
		model:             model,
		writes:            nil,
		features:          nil,
		pendingReports:    nil,
		reportArrived:     make(chan struct{}, 1),
		closed:            false,
		disconnectAfter:   -1,
		writesBeforeError: 0,
	}
}

// DisconnectAfterWrites makes the deck vanish once it has accepted the given
// number of further writes, as when the dock switches mid repaint.
func (device *DeckDeviceTestDouble) DisconnectAfterWrites(writeCount int) {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	device.disconnectAfter = writeCount
	device.writesBeforeError = 0
}

// Unplug makes the deck vanish now.
func (device *DeckDeviceTestDouble) Unplug() {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	device.disconnectAfter = 0
	device.writesBeforeError = 0
	device.signalReport()
}

// PressKeys queues an input report with exactly the given keys down.
func (device *DeckDeviceTestDouble) PressKeys(keyIndices ...int) {
	report := make([]byte, device.model.Protocol.InputReportLength)
	for _, keyIndex := range keyIndices {
		report[device.model.Protocol.InputHeaderLength+keyIndex] = 1
	}

	device.mutex.Lock()
	defer device.mutex.Unlock()

	device.pendingReports = append(device.pendingReports, report)
	device.signalReport()
}

// Write records an output report.
func (device *DeckDeviceTestDouble) Write(report []byte) error {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	if device.unusable() {
		return fmt.Errorf("fake deck write: %w", deck.ErrDisconnected)
	}

	device.writes = append(device.writes, append([]byte(nil), report...))

	if device.disconnectAfter > 0 {
		device.disconnectAfter--
	}

	return nil
}

// SendFeatureReport records a feature report.
func (device *DeckDeviceTestDouble) SendFeatureReport(report []byte) error {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	if device.unusable() {
		return fmt.Errorf("fake deck feature: %w", deck.ErrDisconnected)
	}

	device.features = append(device.features, append([]byte(nil), report...))

	return nil
}

// GetFeatureReport answers the serial number report with FakeDeckSerial.
func (device *DeckDeviceTestDouble) GetFeatureReport(reportID byte, length int) ([]byte, error) {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	if device.unusable() {
		return nil, fmt.Errorf("fake deck feature read: %w", deck.ErrDisconnected)
	}

	report := make([]byte, length)
	report[0] = reportID
	copy(report[device.model.Protocol.SerialTextOffset:], FakeDeckSerial)

	return report, nil
}

// ReadInputReport returns the next queued report, or zero after the timeout.
func (device *DeckDeviceTestDouble) ReadInputReport(buffer []byte, timeout time.Duration) (int, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		device.mutex.Lock()

		if device.unusable() {
			device.mutex.Unlock()

			return 0, fmt.Errorf("fake deck read: %w", deck.ErrDisconnected)
		}

		if len(device.pendingReports) > 0 {
			report := device.pendingReports[0]
			device.pendingReports = device.pendingReports[1:]
			device.mutex.Unlock()

			return copy(buffer, report), nil
		}

		device.mutex.Unlock()

		select {
		case <-device.reportArrived:
		case <-deadline.C:
			return 0, nil
		}
	}
}

// Close marks the deck closed. Closing twice is safe.
func (device *DeckDeviceTestDouble) Close() error {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	device.closed = true
	device.signalReport()

	return nil
}

// Closed reports whether Close was called.
func (device *DeckDeviceTestDouble) Closed() bool {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	return device.closed
}

// Writes returns every output report written.
func (device *DeckDeviceTestDouble) Writes() [][]byte {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	return append([][]byte(nil), device.writes...)
}

// Features returns every feature report sent.
func (device *DeckDeviceTestDouble) Features() [][]byte {
	device.mutex.Lock()
	defer device.mutex.Unlock()

	return append([][]byte(nil), device.features...)
}

// DrawnKeys counts the complete key images written, by key index.
func (device *DeckDeviceTestDouble) DrawnKeys() map[int]int {
	drawn := map[int]int{}

	for _, report := range device.Writes() {
		if report[0] == device.model.Protocol.ImageReportID &&
			report[imageCommandPosition] == device.model.Protocol.ImageCommand && report[imageLastPosition] == 1 {
			drawn[int(report[imageKeyPosition])]++
		}
	}

	return drawn
}

func (device *DeckDeviceTestDouble) signalReport() {
	select {
	case device.reportArrived <- struct{}{}:
	default:
	}
}

func (device *DeckDeviceTestDouble) unusable() bool {
	return device.closed || device.disconnectAfter == 0
}

// DeckWatcherTestDouble is a fake set of attached decks.
type DeckWatcherTestDouble struct {
	mutex       sync.Mutex
	attached    map[string]*DeckDeviceTestDouble
	busy        map[string]bool
	enumeration int
}

// NewDeckWatcherTestDouble returns a watcher with no decks attached.
func NewDeckWatcherTestDouble() *DeckWatcherTestDouble {
	return &DeckWatcherTestDouble{
		mutex:       sync.Mutex{},
		attached:    map[string]*DeckDeviceTestDouble{},
		busy:        map[string]bool{},
		enumeration: 0,
	}
}

// Attach plugs a fake deck in at path.
func (watcher *DeckWatcherTestDouble) Attach(path string, device *DeckDeviceTestDouble) {
	watcher.mutex.Lock()
	defer watcher.mutex.Unlock()

	watcher.attached[path] = device
}

// Detach unplugs the deck at path.
func (watcher *DeckWatcherTestDouble) Detach(path string) {
	watcher.mutex.Lock()
	defer watcher.mutex.Unlock()

	if device, attached := watcher.attached[path]; attached {
		device.Unplug()
	}

	delete(watcher.attached, path)
}

// SetBusy makes the deck at path refuse to open, as when other software holds it.
func (watcher *DeckWatcherTestDouble) SetBusy(path string, busy bool) {
	watcher.mutex.Lock()
	defer watcher.mutex.Unlock()

	watcher.busy[path] = busy
}

// Enumerations counts calls to Enumerate.
func (watcher *DeckWatcherTestDouble) Enumerations() int {
	watcher.mutex.Lock()
	defer watcher.mutex.Unlock()

	return watcher.enumeration
}

// Enumerate lists the attached fake decks.
func (watcher *DeckWatcherTestDouble) Enumerate() ([]deck.Connection, error) {
	watcher.mutex.Lock()
	defer watcher.mutex.Unlock()

	watcher.enumeration++

	connections := make([]deck.Connection, 0, len(watcher.attached))
	for path := range watcher.attached {
		connections = append(connections, deck.Connection{
			Path: path, VendorID: FakeDeckVendorID, ProductID: FakeDeckProductID,
		})
	}

	return connections, nil
}

// Open opens an attached fake deck.
//
//nolint:ireturn // implements the port, which returns a device interface
func (watcher *DeckWatcherTestDouble) Open(connection deck.Connection) (deck.DeckDevice, error) {
	watcher.mutex.Lock()
	defer watcher.mutex.Unlock()

	device, attached := watcher.attached[connection.Path]

	switch {
	case !attached:
		return nil, fmt.Errorf("fake deck %s: %w", connection.Path, deck.ErrDisconnected)
	case watcher.busy[connection.Path]:
		return nil, fmt.Errorf("fake deck %s: %w", connection.Path, deck.ErrDeckBusy)
	default:
		return device, nil
	}
}
