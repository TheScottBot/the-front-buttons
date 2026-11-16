package hidtransport

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// The Windows HID driver calls hid.dll, setupapi.dll and kernel32 through
// golang.org/x/sys/windows, needing no C compiler, the way hidapi's Windows
// backend does. Its decisions are in winhidcore.go, which every system
// builds; this file holds only the Windows calls.

func init() { builtDrivers[DriverPurego] = &winhidDriver{} } //nolint:gochecknoinits // registers the driver

// The Win32 values the driver uses: SetupDiGetClassDevsW's flags for present
// devices listed by interface (setupapi.h), the input buffers hidapi asks the
// HID class driver to keep, and how long a write may take before it is
// cancelled, hidapi's own second.
const (
	digcfPresent             = 0x2
	digcfDeviceInterface     = 0x10
	inputBufferCount         = 64
	writeTimeoutMilliseconds = 1000
)

var (
	errNoInterfacePath  = errors.New("SetupDiGetDeviceInterfaceDetailW gave no path length")
	errNoCapabilities   = errors.New("HidP_GetCaps failed")
	errEmptyReport      = errors.New("an empty report cannot be sent")
	errWriteTimedOut    = errors.New("the write did not finish in time and was cancelled")
	errWaitForWriteFail = errors.New("waiting for the write failed")
)

// winhidProcs are the hid.dll and setupapi.dll functions the driver calls.
type winhidProcs struct {
	getHidGUID               *windows.LazyProc
	getAttributes            *windows.LazyProc
	setNumInputBuffers       *windows.LazyProc
	getPreparsedData         *windows.LazyProc
	freePreparsedData        *windows.LazyProc
	getCaps                  *windows.LazyProc
	setFeature               *windows.LazyProc
	getFeature               *windows.LazyProc
	getClassDevs             *windows.LazyProc
	enumDeviceInterfaces     *windows.LazyProc
	getDeviceInterfaceDetail *windows.LazyProc
	destroyDeviceInfoList    *windows.LazyProc
}

func (procs *winhidProcs) all() []*windows.LazyProc {
	return []*windows.LazyProc{
		procs.getHidGUID, procs.getAttributes, procs.setNumInputBuffers, procs.getPreparsedData,
		procs.freePreparsedData, procs.getCaps, procs.setFeature, procs.getFeature, procs.getClassDevs,
		procs.enumDeviceInterfaces, procs.getDeviceInterfaceDetail, procs.destroyDeviceInfoList,
	}
}

// loadWinHIDProcs finds every function the driver calls, once per process,
// in this one place. A missing library or function, or a panic finding one,
// makes the driver unavailable (errWinHIDUnavailable) instead of ending the
// daemon; once found, calling a function cannot panic.
//
//nolint:gochecknoglobals // the system's libraries are process wide
var loadWinHIDProcs = sync.OnceValues(func() (*winhidProcs, error) {
	var found *winhidProcs

	bindError := bindRecovering(func() error {
		hid := windows.NewLazySystemDLL("hid.dll")
		setup := windows.NewLazySystemDLL("setupapi.dll")

		procs := &winhidProcs{
			getHidGUID:               hid.NewProc("HidD_GetHidGuid"),
			getAttributes:            hid.NewProc("HidD_GetAttributes"),
			setNumInputBuffers:       hid.NewProc("HidD_SetNumInputBuffers"),
			getPreparsedData:         hid.NewProc("HidD_GetPreparsedData"),
			freePreparsedData:        hid.NewProc("HidD_FreePreparsedData"),
			getCaps:                  hid.NewProc("HidP_GetCaps"),
			setFeature:               hid.NewProc("HidD_SetFeature"),
			getFeature:               hid.NewProc("HidD_GetFeature"),
			getClassDevs:             setup.NewProc("SetupDiGetClassDevsW"),
			enumDeviceInterfaces:     setup.NewProc("SetupDiEnumDeviceInterfaces"),
			getDeviceInterfaceDetail: setup.NewProc("SetupDiGetDeviceInterfaceDetailW"),
			destroyDeviceInfoList:    setup.NewProc("SetupDiDestroyDeviceInfoList"),
		}

		for _, proc := range procs.all() {
			findError := proc.Find()
			if findError != nil {
				return fmt.Errorf("finding %s: %w", proc.Name, findError)
			}
		}

		found = procs

		return nil
	})
	if bindError != nil {
		return nil, bindError
	}

	return found, nil
})

// failure describes a failed Windows call by its error code (see callError).
func failure(call string, cause error) error {
	if code, isErrno := errors.AsType[windows.Errno](cause); isErrno {
		return callError(call, uintptr(code), cause)
	}

	return fmt.Errorf("%s: %w", call, cause)
}

// callDiscarding makes a call whose results say nothing the caller needs:
// HidD_GetHidGuid returns nothing, the cleanup calls cannot be acted on, and
// SetupDiGetDeviceInterfaceDetailW's first call fails by design, only
// filling in the size it needs.
func callDiscarding(proc *windows.LazyProc, arguments ...uintptr) {
	_, _, _ = proc.Call(arguments...) //nolint:dogsled // see above
}

// pointerTo passes a pointer to a system call. unsafe is the only way to.
func pointerTo[Value any](value *Value) uintptr {
	return uintptr(unsafe.Pointer(value)) //nolint:gosec // a pointer handed to Windows for the call
}

// winhidDriver lists and opens HID devices through hid.dll and setupapi.dll.
type winhidDriver struct{}

// enumerate lists every present HID interface with SetupDiGetClassDevsW and
// SetupDiEnumDeviceInterfaces, reads each one's path with
// SetupDiGetDeviceInterfaceDetailW, opens it with no access to read its vendor
// and product with HidD_GetAttributes, and keeps the vendor's. An interface
// whose path or attributes cannot be read is passed over, as hidapi does.
func (*winhidDriver) enumerate(vendorID uint16) ([]deck.Connection, error) {
	procs, loadError := loadWinHIDProcs()
	if loadError != nil {
		return nil, loadError
	}

	var classGUID windowsGUID

	callDiscarding(procs.getHidGUID, pointerTo(&classGUID))

	set, _, listError := procs.getClassDevs.Call(pointerTo(&classGUID), 0, 0, digcfPresent|digcfDeviceInterface)
	if windows.Handle(set) == windows.InvalidHandle {
		return nil, failure("SetupDiGetClassDevsW", listError)
	}

	defer callDiscarding(procs.destroyDeviceInfoList, set)

	var found []hidInterface

	for index := uintptr(0); ; index++ {
		var data deviceInterfaceData

		data.size = uint32(unsafe.Sizeof(data))

		listed, _, enumerateError := procs.enumDeviceInterfaces.Call(set, 0, pointerTo(&classGUID), index, pointerTo(&data))
		if !boolSucceeded(listed) {
			if errors.Is(enumerateError, windows.ERROR_NO_MORE_ITEMS) {
				break
			}

			return nil, failure("SetupDiEnumDeviceInterfaces", enumerateError)
		}

		path, pathError := interfacePath(procs, set, &data)
		if pathError != nil {
			continue
		}

		attributes, attributesError := readAttributes(procs, path)
		if attributesError != nil {
			continue
		}

		found = append(found, hidInterface{path: path, vendorID: attributes.vendorID, productID: attributes.productID})
	}

	return connectionsOfVendor(found, vendorID), nil
}

// interfacePath asks SetupDiGetDeviceInterfaceDetailW how much room the
// interface's detail needs, then for the detail, and reads the path from it.
func interfacePath(procs *winhidProcs, set uintptr, data *deviceInterfaceData) (string, error) {
	var required uint32

	callDiscarding(procs.getDeviceInterfaceDetail, set, pointerTo(data), 0, 0, pointerTo(&required), 0)

	if required == 0 {
		return "", errNoInterfacePath
	}

	detail := newInterfaceDetailBuffer(required, unsafe.Sizeof(uintptr(0)))

	read, _, readError := procs.getDeviceInterfaceDetail.Call(set, pointerTo(data), pointerTo(&detail[0]),
		uintptr(len(detail)*utf16UnitBytes), 0, 0)
	if !boolSucceeded(read) {
		return "", failure("SetupDiGetDeviceInterfaceDetailW", readError)
	}

	return pathFromInterfaceDetail(detail), nil
}

// readAttributes opens the interface with no access, as hidapi's enumeration
// does, so a device the system holds for itself still answers, and reads its
// vendor and product with HidD_GetAttributes.
func readAttributes(procs *winhidProcs, path string) (hidAttributes, error) {
	device, openError := openPath(path, 0)
	if openError != nil {
		return hidAttributes{}, openError
	}

	defer func() { _ = windows.CloseHandle(device) }()

	var attributes hidAttributes

	attributes.size = uint32(unsafe.Sizeof(attributes))

	read, _, readError := procs.getAttributes.Call(uintptr(device), pointerTo(&attributes))
	if !booleanSucceeded(read) {
		return hidAttributes{}, failure("HidD_GetAttributes", readError)
	}

	return attributes, nil
}

// openPath opens an interface path with CreateFileW for overlapped I/O, shared
// for reading and writing, as hidapi opens it.
func openPath(path string, access uint32) (windows.Handle, error) {
	name, nameError := windows.UTF16PtrFromString(path)
	if nameError != nil {
		return windows.InvalidHandle, fmt.Errorf("the path: %w", nameError)
	}

	device, createError := windows.CreateFile(name, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
	if createError != nil {
		return windows.InvalidHandle, failure("CreateFileW", createError)
	}

	return device, nil
}

// open opens the path for reading and writing. As hidapi does, a device that
// will not open so is opened again with no access, which still carries
// feature reports.
//
//nolint:ireturn // the driver contract returns a handle interface
func (*winhidDriver) open(path string) (handle, error) {
	procs, loadError := loadWinHIDProcs()
	if loadError != nil {
		return nil, loadError
	}

	device, openError := openPath(path, windows.GENERIC_READ|windows.GENERIC_WRITE)
	if openError != nil {
		var retryError error

		device, retryError = openPath(path, 0)
		if retryError != nil {
			return nil, openError
		}
	}

	calls, setUpError := setUpDevice(procs, device)
	if setUpError != nil {
		_ = windows.CloseHandle(device)

		return nil, setUpError
	}

	return newWinHIDHandle(calls, calls.lengths), nil
}

// setUpDevice asks for hidapi's 64 input buffers with HidD_SetNumInputBuffers,
// reads the report lengths, and makes an event for each of the read and the
// write.
func setUpDevice(procs *winhidProcs, device windows.Handle) (*winhidDevice, error) {
	set, _, setError := procs.setNumInputBuffers.Call(uintptr(device), inputBufferCount)
	if !booleanSucceeded(set) {
		return nil, failure("HidD_SetNumInputBuffers", setError)
	}

	lengths, lengthsError := readLengths(procs, device)
	if lengthsError != nil {
		return nil, lengthsError
	}

	readEvent, readEventError := windows.CreateEvent(nil, 1, 0, nil)
	if readEventError != nil {
		return nil, failure("CreateEventW", readEventError)
	}

	writeEvent, writeEventError := windows.CreateEvent(nil, 1, 0, nil)
	if writeEventError != nil {
		_ = windows.CloseHandle(readEvent)

		return nil, failure("CreateEventW", writeEventError)
	}

	calls := &winhidDevice{
		procs:           procs,
		device:          device,
		lengths:         lengths,
		readEvent:       readEvent,
		writeEvent:      writeEvent,
		readOverlapped:  overlappedWith(readEvent),
		writeOverlapped: overlappedWith(writeEvent),
		readBuffer:      make([]byte, max(lengths.input, 1)),
		pinner:          runtime.Pinner{},
	}

	// Windows writes into the overlapped structures and the read buffer
	// after the calls that hand them over return, so they are pinned until
	// the device is released.
	calls.pinner.Pin(calls)
	calls.pinner.Pin(&calls.readBuffer[0])

	return calls, nil
}

func overlappedWith(event windows.Handle) windows.Overlapped {
	return windows.Overlapped{Internal: 0, InternalHigh: 0, Offset: 0, OffsetHigh: 0, HEvent: event}
}

// readLengths reads the longest input, output and feature reports with
// HidD_GetPreparsedData and HidP_GetCaps, as hidapi does on opening.
func readLengths(procs *winhidProcs, device windows.Handle) (reportLengths, error) {
	var preparsed uintptr

	got, _, getError := procs.getPreparsedData.Call(uintptr(device), pointerTo(&preparsed))
	if !booleanSucceeded(got) {
		return reportLengths{}, failure("HidD_GetPreparsedData", getError)
	}

	defer callDiscarding(procs.freePreparsedData, preparsed)

	var capabilities hidCapabilities

	status, _, _ := procs.getCaps.Call(preparsed, pointerTo(&capabilities))
	if !ntStatusIs(status, hidpStatusSuccess) {
		return reportLengths{}, fmt.Errorf("%w: status %#x", errNoCapabilities, status)
	}

	return capabilities.lengths(), nil
}

// winhidDevice is the Windows side of one open device: its handle, its
// events, the overlapped structures of its read and its write, and the
// buffer its pending read fills.
type winhidDevice struct {
	procs           *winhidProcs
	device          windows.Handle
	lengths         reportLengths
	readEvent       windows.Handle
	writeEvent      windows.Handle
	readOverlapped  windows.Overlapped
	writeOverlapped windows.Overlapped
	readBuffer      []byte
	pinner          runtime.Pinner
}

// writeOutput sends report with an overlapped WriteFile and waits up to a
// second for it, as hidapi's hid_write does. A write still going then is
// cancelled, and waited for, so Windows is done with the report on return.
func (opened *winhidDevice) writeOutput(report []byte) error {
	if len(report) == 0 {
		return errEmptyReport
	}

	var pinner runtime.Pinner

	pinner.Pin(&report[0])
	defer pinner.Unpin()

	writeError := windows.WriteFile(opened.device, report, nil, &opened.writeOverlapped)
	if writeError == nil {
		return nil
	}

	if !errors.Is(writeError, windows.ERROR_IO_PENDING) {
		return failure("WriteFile", writeError)
	}

	event, waitError := windows.WaitForSingleObject(opened.writeEvent, writeTimeoutMilliseconds)
	if waitError != nil || event != windows.WAIT_OBJECT_0 {
		cancelError := opened.cancelAndWait(&opened.writeOverlapped)
		if cancelError != nil {
			return cancelError
		}

		if waitError != nil {
			return fmt.Errorf("%w: %w", errWaitForWriteFail, waitError)
		}

		return errWriteTimedOut
	}

	var written uint32

	resultError := windows.GetOverlappedResult(opened.device, &opened.writeOverlapped, &written, false)
	if resultError != nil {
		return failure("WriteFile", resultError)
	}

	return nil
}

// setFeature sends report with HidD_SetFeature.
func (opened *winhidDevice) setFeature(report []byte) error {
	if len(report) == 0 {
		return errEmptyReport
	}

	sent, _, sendError := opened.procs.setFeature.Call(uintptr(opened.device), pointerTo(&report[0]),
		uintptr(len(report)))
	if !booleanSucceeded(sent) {
		return failure("HidD_SetFeature", sendError)
	}

	return nil
}

// getFeature fills request, the report ID in its first byte, with
// HidD_GetFeature.
func (opened *winhidDevice) getFeature(request []byte) error {
	if len(request) == 0 {
		return errEmptyReport
	}

	got, _, getError := opened.procs.getFeature.Call(uintptr(opened.device), pointerTo(&request[0]),
		uintptr(len(request)))
	if !booleanSucceeded(got) {
		return failure("HidD_GetFeature", getError)
	}

	return nil
}

// startRead starts an overlapped ReadFile of the longest input report into
// the device's buffer. Finishing at once and ERROR_IO_PENDING both leave the
// read for readResult to collect.
func (opened *winhidDevice) startRead() error {
	readError := windows.ReadFile(opened.device, opened.readBuffer, nil, &opened.readOverlapped)
	if readError != nil && !errors.Is(readError, windows.ERROR_IO_PENDING) {
		return failure("ReadFile", readError)
	}

	return nil
}

// readResult asks GetOverlappedResult without waiting; ERROR_IO_INCOMPLETE
// means the read is still pending.
func (opened *winhidDevice) readResult() ([]byte, bool, error) {
	var read uint32

	resultError := windows.GetOverlappedResult(opened.device, &opened.readOverlapped, &read, false)

	switch {
	case errors.Is(resultError, windows.ERROR_IO_INCOMPLETE):
		return nil, false, nil
	case resultError != nil:
		return nil, false, failure("ReadFile", resultError)
	default:
		return opened.readBuffer[:min(int(read), len(opened.readBuffer))], true, nil
	}
}

// cancelRead cancels the pending read and waits for it to end.
func (opened *winhidDevice) cancelRead() error {
	return opened.cancelAndWait(&opened.readOverlapped)
}

// cancelAndWait cancels one overlapped call with CancelIoEx and waits for it
// to end with GetOverlappedResult. ERROR_NOT_FOUND from the cancel means the
// call had already ended, and ERROR_OPERATION_ABORTED from the wait that the
// cancel stopped it; any other ending is returned, so a device removed under
// the call gives errHandleGone.
func (opened *winhidDevice) cancelAndWait(overlapped *windows.Overlapped) error {
	cancelError := windows.CancelIoEx(opened.device, overlapped)
	if cancelError != nil && !errors.Is(cancelError, windows.ERROR_NOT_FOUND) {
		return failure("CancelIoEx", cancelError)
	}

	var transferred uint32

	resultError := windows.GetOverlappedResult(opened.device, overlapped, &transferred, true)
	if resultError != nil && !errors.Is(resultError, windows.ERROR_OPERATION_ABORTED) {
		return failure("GetOverlappedResult", resultError)
	}

	return nil
}

// release closes the device's handle and its events with CloseHandle and
// unpins what Windows no longer writes into.
func (opened *winhidDevice) release() error {
	closeError := errors.Join(
		windows.CloseHandle(opened.device),
		windows.CloseHandle(opened.readEvent),
		windows.CloseHandle(opened.writeEvent),
	)

	opened.pinner.Unpin()

	if closeError != nil {
		return failure("CloseHandle", closeError)
	}

	return nil
}
