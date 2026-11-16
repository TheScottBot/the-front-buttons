package hidtransport

import (
	"errors"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// DriverName names a HID driver, as the platform_bridge setting does.
type DriverName string

const (
	// DriverHIDAPI is hidapi through cgo (hidapi_driver.go), built only when
	// cgo is on.
	DriverHIDAPI DriverName = "cgo"
	// DriverPurego calls the system's own HID API without cgo: IOKit on macOS,
	// hid.dll on Windows.
	DriverPurego DriverName = "purego"
)

// ErrNoDriver reports a build holding no HID driver at all, so no deck can be
// listed or opened.
var ErrNoDriver = errors.New("no HID driver is built in")

// errHandleGone is returned by every handle call after the device is removed;
// Watcher reports it as deck.ErrDisconnected.
var errHandleGone = errors.New("device removed")

// driver is what a HID driver provides. Watcher keeps the process wide lock,
// the write retry, the read poll, the panic recovery and the busy against
// disconnected decision above it, so a driver holds none of them: enumerate,
// open and a handle's close are only ever called holding the lock.
type driver interface {
	// enumerate lists the attached devices of one vendor, every product.
	enumerate(vendorID uint16) ([]deck.Connection, error)
	open(path string) (handle, error)
}

// handle is one open device. Watcher serialises the calls on each handle.
type handle interface {
	write(report []byte) error
	sendFeature(report []byte) error
	// getFeature reads feature report reportID into length bytes, the first
	// being the ID, and returns the bytes read.
	getFeature(reportID byte, length int) ([]byte, error)
	// readWithoutWaiting returns 0, nil when no report waits.
	readWithoutWaiting(buffer []byte) (int, error)
	close() error
}

// builtDrivers holds the drivers built into this binary, by name. Each
// driver's own file adds itself from an init function, in one line:
//
//	func init() { builtDrivers[DriverPurego] = &iokitDriver{} } //nolint:gochecknoinits // registers the driver
//
// so a build leaving a driver's file out (cgo off, another system) simply
// lacks it, and chooseDriver falls back to another.
//
//nolint:gochecknoglobals // filled by init functions before main, then only read
var builtDrivers = map[DriverName]driver{}

// fallbackOrder is the order drivers are tried in when the one asked for is
// not built in.
func fallbackOrder() []DriverName {
	return []DriverName{DriverHIDAPI, DriverPurego}
}

// chooseDriver picks the driver asked for if it is built in, and otherwise the
// first built in driver in fallbackOrder.
func chooseDriver(asked DriverName, built map[DriverName]driver) (DriverName, error) {
	if _, present := built[asked]; present {
		return asked, nil
	}

	for _, name := range fallbackOrder() {
		if _, present := built[name]; present {
			return name, nil
		}
	}

	return "", fmt.Errorf("%w: %q was asked for", ErrNoDriver, asked)
}

// DriverChoice is the driver a watcher was asked for and the one it uses, for
// the daemon to log.
type DriverChoice struct {
	Asked DriverName
	// Using is empty when no driver is built in.
	Using DriverName
	// Unavailable says why no driver is used; nil whenever Using is set.
	Unavailable error
}

// FellBack reports whether the watcher uses a driver other than the one asked
// for, because that one is not built in.
func (choice DriverChoice) FellBack() bool {
	return choice.Using != "" && choice.Using != choice.Asked
}
