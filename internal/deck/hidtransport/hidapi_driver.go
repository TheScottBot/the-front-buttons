//go:build cgo

package hidtransport

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	hid "github.com/sstallion/go-hid"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

func init() { builtDrivers[DriverHIDAPI] = hidapiDriver{} } //nolint:gochecknoinits // registers the driver

// initialiseHIDAPI initialises hidapi once per process, not once per watcher,
// and gives every later caller the same result. It is only called on hidapi's
// thread, holding the process wide driver lock.
//
//nolint:gochecknoglobals // hidapi's initialisation is process wide
var initialiseHIDAPI = sync.OnceValue(func() error {
	initialiseError := hid.Init()
	if initialiseError != nil {
		return fmt.Errorf("initialising hidapi: %w", initialiseError)
	}

	return nil
})

// hidapiCalls is where calls are sent to run on hidapi's thread. On macOS
// hid_init schedules hidapi's one HID manager on the run loop of the thread
// that calls it, and hid_enumerate runs the run loop of the thread that calls
// it to let the manager catch up. A Go thread is shared by every goroutine,
// so on any thread a goroutine later locked to it, such as the IOKit driver's
// input thread, could run the manager on that run loop while another thread
// enumerates, or end the thread and its run loop with the manager still on
// it: either aborted the process inside hid_enumerate. So every call that
// initialises hidapi or enumerates runs on one goroutine locked to its thread
// for the life of the process, where no other goroutine ever runs.
//
//nolint:gochecknoglobals // hidapi's thread is process wide, as its initialisation is
var hidapiCalls = sync.OnceValue(func() chan<- func() {
	calls := make(chan func())

	go serveHIDAPICalls(calls)

	return calls
})

// serveHIDAPICalls runs each call sent, in turn, and never returns nor
// unlocks its thread.
func serveHIDAPICalls(calls <-chan func()) {
	runtime.LockOSThread()

	for call := range calls {
		call()
	}
}

// onHIDAPIThread runs call on hidapi's thread and waits for it. A panic in
// call, which go-hid can raise while enumerating (finding TF5), is caught
// there and raised again here, so the caller's recover still sees it and
// hidapi's thread lives on.
func onHIDAPIThread(call func()) {
	var recovered any

	done := make(chan struct{})

	hidapiCalls() <- func() {
		defer close(done)
		defer func() { recovered = recover() }()

		call()
	}

	<-done

	if recovered != nil {
		panic(recovered)
	}
}

// hidapiDriver is the driver through go-hid, hidapi built with cgo.
type hidapiDriver struct{}

func (hidapiDriver) enumerate(vendorID uint16) ([]deck.Connection, error) {
	var (
		found     []deck.Connection
		listError error
	)

	onHIDAPIThread(func() { found, listError = enumerateOnHIDAPIThread(vendorID) })

	return found, listError
}

// enumerateOnHIDAPIThread lists through hidapi, initialising it first if need
// be.
func enumerateOnHIDAPIThread(vendorID uint16) ([]deck.Connection, error) {
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
	var (
		opened    handle
		openError error
	)

	onHIDAPIThread(func() { opened, openError = openOnHIDAPIThread(path) })

	return opened, openError
}

// openOnHIDAPIThread opens through hidapi, initialising it first if need be.
//
//nolint:ireturn // the driver contract returns a handle interface
func openOnHIDAPIThread(path string) (handle, error) {
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
