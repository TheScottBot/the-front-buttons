package hidtransport

import (
	"errors"
	"fmt"
	"unicode/utf16"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// The Windows HID driver (winhid_windows.go) keeps its Windows calls thin.
// What it decides is here, in plain Go built on every system, so its tests
// run anywhere: report padding, the report ID byte, error mapping, reading
// interface paths, filtering by vendor, the layouts of the structures Windows
// fills, and the handle's state (the pending read, removal, closing).

// The Windows error codes that mean the device has gone, from WinError.h. A
// pending read ends with ERROR_DEVICE_NOT_CONNECTED when the device is
// unplugged; writes and feature reports to it fail with that or
// ERROR_GEN_FAILURE; and a handle Windows has already torn down gives
// ERROR_INVALID_HANDLE.
const (
	windowsErrorInvalidHandle      = 6
	windowsErrorGenFailure         = 31
	windowsErrorDeviceNotConnected = 1167
)

// isRemovalCode reports whether a Windows error code means the device has gone.
func isRemovalCode(code uintptr) bool {
	switch code {
	case windowsErrorInvalidHandle, windowsErrorGenFailure, windowsErrorDeviceNotConnected:
		return true
	default:
		return false
	}
}

// callError describes a failed Windows call, wrapping errHandleGone as well as
// the cause when the code says the device has gone.
func callError(call string, code uintptr, cause error) error {
	if isRemovalCode(code) {
		return fmt.Errorf("%w: %s: %w", errHandleGone, call, cause)
	}

	return fmt.Errorf("%s: %w", call, cause)
}

// padReport returns report at least length bytes long, padded with zeros. As
// hidapi does, a report already that long is returned as it is: Windows wants
// the longest report's length from WriteFile and HidD_SetFeature, refusing
// fewer bytes.
func padReport(report []byte, length int) []byte {
	if len(report) >= length {
		return report
	}

	padded := make([]byte, length)
	copy(padded, report)

	return padded
}

// copyInputReport copies a report ReadFile returned into buffer and returns the
// bytes copied. As hidapi's Windows backend does, a leading report ID of zero,
// which Windows adds for a device that numbers no reports, is left out, so the
// report reads as on other systems; any other ID is kept.
func copyInputReport(buffer, read []byte) int {
	if len(read) > 0 && read[0] == 0 {
		read = read[1:]
	}

	return copy(buffer, read)
}

// errFeatureLength reports a feature report asked for with no room for its ID.
var errFeatureLength = errors.New("a feature report needs at least one byte, its ID")

// featureRequest is the buffer HidD_GetFeature fills: reportID in its first
// byte, and long enough for both the length asked for and the device's
// feature report.
func featureRequest(reportID byte, length, featureLength int) ([]byte, error) {
	if length < 1 {
		return nil, fmt.Errorf("%w: %d asked for", errFeatureLength, length)
	}

	request := make([]byte, max(length, featureLength))
	request[0] = reportID

	return request, nil
}

// hidInterface is one HID interface Windows lists: its path and the vendor and
// product HidD_GetAttributes read from it.
type hidInterface struct {
	path      string
	vendorID  uint16
	productID uint16
}

// connectionsOfVendor keeps the interfaces of one vendor, as hidapi's
// enumeration does; vendor zero keeps every interface, as in hidapi.
func connectionsOfVendor(found []hidInterface, vendorID uint16) []deck.Connection {
	var kept []deck.Connection

	for _, listed := range found {
		if vendorID != 0 && listed.vendorID != vendorID {
			continue
		}

		kept = append(kept, deck.Connection{Path: listed.path, VendorID: listed.vendorID, ProductID: listed.productID})
	}

	return kept
}

// SP_DEVICE_INTERFACE_DETAIL_DATA_W is a DWORD size followed by the path in
// UTF16. Its size field must hold the structure's size with one character of
// path: 8 bytes where pointers are 8 bytes (the DWORD alignment pads 6 to 8),
// and 6 on 32 bit Windows, where setupapi.h packs its structures to one byte.
const (
	interfaceDetailSize64 = 8
	interfaceDetailSize32 = 6
	pointerSize64         = 8
	// interfaceDetailPathStart is where the path starts, in UTF16 units,
	// after the DWORD size field.
	interfaceDetailPathStart = 2
	utf16UnitBytes           = 2
	lowWordMask              = 0xFFFF
	wordBits                 = 16
)

// interfaceDetailHeaderSize is the size field's value for SP_DEVICE_INTERFACE_DETAIL_DATA_W.
func interfaceDetailHeaderSize(pointerSize uintptr) uint32 {
	if pointerSize == pointerSize64 {
		return interfaceDetailSize64
	}

	return interfaceDetailSize32
}

// newInterfaceDetailBuffer returns room for requiredBytes of
// SP_DEVICE_INTERFACE_DETAIL_DATA_W, as SetupDiGetDeviceInterfaceDetailW said
// it needs, with the size field set, little endian, in its first two units.
func newInterfaceDetailBuffer(requiredBytes uint32, pointerSize uintptr) []uint16 {
	units := max((int(requiredBytes)+utf16UnitBytes-1)/utf16UnitBytes, interfaceDetailPathStart+1)
	detail := make([]uint16, units)
	size := interfaceDetailHeaderSize(pointerSize)
	detail[0] = uint16(size & lowWordMask)
	detail[1] = uint16(size >> wordBits)

	return detail
}

// pathFromInterfaceDetail reads the path from a filled
// SP_DEVICE_INTERFACE_DETAIL_DATA_W, up to its terminating zero. It is the
// interface's symbolic link, the same string hidapi's Windows backend lists
// (from CM_Get_Device_Interface_ListW) as the path, in UTF8 as hidapi gives it.
func pathFromInterfaceDetail(detail []uint16) string {
	if len(detail) <= interfaceDetailPathStart {
		return ""
	}

	path := detail[interfaceDetailPathStart:]

	for index, unit := range path {
		if unit == 0 {
			path = path[:index]

			break
		}
	}

	return string(utf16.Decode(path))
}

// Masks for the results of the system calls: a BOOL is a 32 bit int and hid.dll's
// BOOLEAN one byte, so only those bits of the returned register mean anything.
const (
	boolMask    = 0xFFFFFFFF
	booleanMask = 0xFF
)

// boolSucceeded reads a BOOL result: nonzero is success.
func boolSucceeded(result uintptr) bool { return result&boolMask != 0 }

// booleanSucceeded reads hid.dll's BOOLEAN result: nonzero is success.
func booleanSucceeded(result uintptr) bool { return result&booleanMask != 0 }

// ntStatusIs reports whether an NTSTATUS result, such as HidP_GetCaps
// returns, is status.
func ntStatusIs(result uintptr, status uint32) bool { return result&boolMask == uintptr(status) }

// hidpStatusSuccess is HIDP_STATUS_SUCCESS, from hidpi.h.
const hidpStatusSuccess = 0x00110000

// windowsGUID is a GUID as Windows lays it out: 16 bytes, aligned to four.
// The driver only passes GUIDs Windows fills, never reading their parts.
type windowsGUID struct {
	_ [4]uint32
}

// hidAttributes is HIDD_ATTRIBUTES, from hidsdi.h, its version number, which
// the driver does not read, left unnamed.
type hidAttributes struct {
	size      uint32
	vendorID  uint16
	productID uint16
	_         uint16
}

// hidCapabilities is HIDP_CAPS, from hidpi.h: the usage and usage page, the
// report lengths, then reserved space and counts the driver does not read.
type hidCapabilities struct {
	_                       [2]uint16
	inputReportByteLength   uint16
	outputReportByteLength  uint16
	featureReportByteLength uint16
	_                       [27]uint16
}

// deviceInterfaceData is SP_DEVICE_INTERFACE_DATA, from setupapi.h: its size,
// then the class GUID, flags and a reserved pointer, which only Windows reads.
type deviceInterfaceData struct {
	size uint32
	_    windowsGUID
	_    uint32
	_    uintptr
}

// reportLengths are a device's longest input, output and feature reports,
// each counting its ID byte, as HidP_GetCaps gives them.
type reportLengths struct {
	input   int
	output  int
	feature int
}

func (capabilities hidCapabilities) lengths() reportLengths {
	return reportLengths{
		input:   int(capabilities.inputReportByteLength),
		output:  int(capabilities.outputReportByteLength),
		feature: int(capabilities.featureReportByteLength),
	}
}

// errWinHIDUnavailable reports that hid.dll or setupapi.dll lacks a function
// the Windows driver calls, so the driver cannot run.
var errWinHIDUnavailable = errors.New("the Windows HID driver is unavailable")

// bindRecovering runs bind, which finds the system functions the driver
// calls, and reports any failure, a panic included, as the driver being
// unavailable rather than letting it end the daemon. Its result is named so
// the deferred recover can set it.
func bindRecovering(bind func() error) (bindError error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			bindError = fmt.Errorf("%w: %v", errWinHIDUnavailable, recovered)
		}
	}()

	findError := bind()
	if findError != nil {
		return fmt.Errorf("%w: %w", errWinHIDUnavailable, findError)
	}

	return nil
}

// winhidCalls are the Windows calls one open device makes. Each error that
// means the device has gone wraps errHandleGone (see callError).
type winhidCalls interface {
	// writeOutput sends report with WriteFile and waits for it to finish.
	writeOutput(report []byte) error
	// setFeature sends report with HidD_SetFeature.
	setFeature(report []byte) error
	// getFeature fills request, its first byte the report ID, with
	// HidD_GetFeature.
	getFeature(request []byte) error
	// startRead starts an overlapped ReadFile into the device's own buffer.
	startRead() error
	// readResult asks, without waiting, whether the read has finished, and
	// if so returns the bytes it read, which stay valid until the next
	// startRead.
	readResult() (report []byte, finished bool, resultError error)
	// cancelRead cancels the pending read with CancelIoEx and waits for it
	// to end, returning how it ended.
	cancelRead() error
	// release closes the device's events and handle.
	release() error
}

// winhidHandle is one device open through the Windows driver. Watcher
// serialises the calls on it, so it holds no lock.
type winhidHandle struct {
	calls   winhidCalls
	lengths reportLengths
	// pending says an overlapped read has been started and not yet
	// collected, so cancelRead must be called before closing.
	pending bool
	// gone is set by the first call that finds the device removed; every
	// later call returns errHandleGone.
	gone bool
}

func newWinHIDHandle(calls winhidCalls, lengths reportLengths) *winhidHandle {
	return &winhidHandle{calls: calls, lengths: lengths, pending: false, gone: false}
}

// noted returns callResult, first marking the handle gone if it says the
// device has gone.
func (opened *winhidHandle) noted(callResult error) error {
	if errors.Is(callResult, errHandleGone) {
		opened.gone = true
	}

	return callResult
}

func (opened *winhidHandle) write(report []byte) error {
	if opened.gone {
		return errHandleGone
	}

	return opened.noted(opened.calls.writeOutput(padReport(report, opened.lengths.output)))
}

func (opened *winhidHandle) sendFeature(report []byte) error {
	if opened.gone {
		return errHandleGone
	}

	return opened.noted(opened.calls.setFeature(padReport(report, opened.lengths.feature)))
}

func (opened *winhidHandle) getFeature(reportID byte, length int) ([]byte, error) {
	if opened.gone {
		return nil, errHandleGone
	}

	request, requestError := featureRequest(reportID, length, opened.lengths.feature)
	if requestError != nil {
		return nil, requestError
	}

	getError := opened.noted(opened.calls.getFeature(request))
	if getError != nil {
		return nil, getError
	}

	return request[:length], nil
}

// readWithoutWaiting keeps one overlapped read pending. If it has finished,
// the report is copied out and the next read started; if not, it returns 0
// without waiting, the read left pending.
func (opened *winhidHandle) readWithoutWaiting(buffer []byte) (int, error) {
	if opened.gone {
		return 0, errHandleGone
	}

	if opened.lengths.input == 0 {
		return 0, nil
	}

	if !opened.pending {
		startError := opened.noted(opened.calls.startRead())
		if startError != nil {
			return 0, startError
		}

		opened.pending = true
	}

	report, finished, resultError := opened.calls.readResult()
	if resultError != nil {
		opened.pending = false

		return 0, opened.noted(resultError)
	}

	if !finished {
		return 0, nil
	}

	opened.pending = false
	copied := copyInputReport(buffer, report)

	// The report is out of the device's buffer, so the next read can start.
	// If it will not start, the next call starts it again and reports why.
	if opened.calls.startRead() == nil {
		opened.pending = true
	}

	return copied, nil
}

// close cancels the pending read, waiting for it to end so Windows no longer
// writes into its buffer, then releases the device. A device found removed,
// by an earlier call or by how the pending read ended, gives errHandleGone
// once released.
func (opened *winhidHandle) close() error {
	if opened.pending {
		opened.pending = false
		_ = opened.noted(opened.calls.cancelRead())
	}

	releaseError := opened.calls.release()

	switch {
	case opened.gone && releaseError != nil:
		return fmt.Errorf("%w: releasing: %w", errHandleGone, releaseError)
	case opened.gone:
		return errHandleGone
	case releaseError != nil:
		return fmt.Errorf("releasing: %w", releaseError)
	default:
		return nil
	}
}
