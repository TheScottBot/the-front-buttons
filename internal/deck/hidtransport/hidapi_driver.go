//go:build cgo

package hidtransport

import (
	"errors"
	"fmt"
	"sync"

	hid "github.com/sstallion/go-hid"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

func init() { builtDrivers[DriverHIDAPI] = hidapiDriver{} } //nolint:gochecknoinits // registers the driver

// initialiseHIDAPI initialises hidapi once per process, not once per watcher,
// and gives every later caller the same result. It is only called holding the
// process wide driver lock.
//
//nolint:gochecknoglobals // hidapi's initialisation is process wide
var initialiseHIDAPI = sync.OnceValue(func() error {
	initialiseError := hid.Init()
	if initialiseError != nil {
		return fmt.Errorf("initialising hidapi: %w", initialiseError)
	}

	return nil
})

// hidapiDriver is the driver through go-hid, hidapi built with cgo.
type hidapiDriver struct{}

func (hidapiDriver) enumerate(vendorID uint16) ([]deck.Connection, error) {
	initialiseError := initialiseHIDAPI()
	if initialiseError != nil {
		return nil, initialiseError
	}

	var found []deck.Connection

	listError := hid.Enumerate(vendorID, hid.ProductIDAny, func(information *hid.DeviceInfo) error {
		found = append(found, deck.Connection{
			Path:      information.Path,
			VendorID:  information.VendorID,
			ProductID: information.ProductID,
		})

		return nil
	})
	if listError != nil {
		return nil, fmt.Errorf("listing: %w", listError)
	}

	return found, nil
}

//nolint:ireturn // the driver contract returns a handle interface
func (hidapiDriver) open(path string) (handle, error) {
	initialiseError := initialiseHIDAPI()
	if initialiseError != nil {
		return nil, initialiseError
	}

	opened, pathError := hid.OpenPath(path)
	if pathError != nil {
		return nil, fmt.Errorf("opening the path: %w", pathError)
	}

	return hidapiHandle{device: opened}, nil
}

// hidapiHandle is one device open through go-hid. hidapi cannot tell a
// removed device from any other failure, so it never returns errHandleGone;
// the watcher reports its failures as disconnected all the same.
type hidapiHandle struct {
	device *hid.Device
}

func (opened hidapiHandle) write(report []byte) error {
	_, writeError := opened.device.Write(report)
	if writeError != nil {
		return fmt.Errorf("hidapi write: %w", writeError)
	}

	return nil
}

func (opened hidapiHandle) sendFeature(report []byte) error {
	_, sendError := opened.device.SendFeatureReport(report)
	if sendError != nil {
		return fmt.Errorf("hidapi feature report: %w", sendError)
	}

	return nil
}

func (opened hidapiHandle) getFeature(reportID byte, length int) ([]byte, error) {
	buffer := make([]byte, length)
	buffer[0] = reportID

	readLength, readError := opened.device.GetFeatureReport(buffer)
	if readError != nil {
		return nil, fmt.Errorf("hidapi feature report: %w", readError)
	}

	return buffer[:readLength], nil
}

// readWithoutWaiting asks hidapi with a timeout of zero, which reports no
// waiting report as hid.ErrTimeout.
func (opened hidapiHandle) readWithoutWaiting(buffer []byte) (int, error) {
	readLength, readError := opened.device.ReadWithTimeout(buffer, 0)

	switch {
	case errors.Is(readError, hid.ErrTimeout):
		return 0, nil
	case readError != nil:
		return 0, fmt.Errorf("hidapi read: %w", readError)
	default:
		return readLength, nil
	}
}

func (opened hidapiHandle) close() error {
	closeError := opened.device.Close()
	if closeError != nil {
		return fmt.Errorf("hidapi close: %w", closeError)
	}

	return nil
}
