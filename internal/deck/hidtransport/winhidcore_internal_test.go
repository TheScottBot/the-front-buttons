package hidtransport

import (
	"bytes"
	"errors"
	"testing"
	"unicode/utf16"
	"unsafe"
)

// The Windows driver's own logic, run on every system against fake Windows
// calls. The Windows calls themselves are checked on Windows with a deck
// attached (winhid_windows_test.go).

const (
	fakeInputLength   = 512
	fakeOutputLength  = 1024
	fakeFeatureLength = 32
	fakeReportID      = 0x06
	fakeInputID       = 0x01
	codeAccessDenied  = 5
	codeIOIncomplete  = 996
	elgatoTestVendor  = 0x0FD9
	otherTestVendor   = 0x046D
	xlProductID       = 0x006C
	keyboardProductID = 0xC31C
)

var (
	errFakeWindowsCall = errors.New("fake Windows error")
	errFakeMissingProc = errors.New("procedure not found")
)

// fakeWinHIDCalls records what the handle asks Windows for and answers as
// told.
type fakeWinHIDCalls struct {
	calls        []string
	written      [][]byte
	reads        [][]byte
	readFinished bool
	failWith     error
	cancelResult error
}

func newFakeWinHIDCalls() *fakeWinHIDCalls {
	return &fakeWinHIDCalls{
		calls: nil, written: nil, reads: nil, readFinished: false, failWith: nil, cancelResult: nil,
	}
}

func (fake *fakeWinHIDCalls) record(call string, report []byte) error {
	fake.calls = append(fake.calls, call)
	fake.written = append(fake.written, append([]byte(nil), report...))

	return fake.failWith
}

func (fake *fakeWinHIDCalls) writeOutput(report []byte) error {
	return fake.record("WriteFile", report)
}

func (fake *fakeWinHIDCalls) setFeature(report []byte) error {
	return fake.record("HidD_SetFeature", report)
}

func (fake *fakeWinHIDCalls) getFeature(request []byte) error {
	recordError := fake.record("HidD_GetFeature", request)
	if recordError == nil {
		for index := 1; index < len(request); index++ {
			request[index] = byte(index)
		}
	}

	return recordError
}

func (fake *fakeWinHIDCalls) startRead() error { return fake.record("ReadFile", nil) }

func (fake *fakeWinHIDCalls) readResult() ([]byte, bool, error) {
	fake.calls = append(fake.calls, "GetOverlappedResult")

	if fake.failWith != nil {
		return nil, false, fake.failWith
	}

	if !fake.readFinished || len(fake.reads) == 0 {
		return nil, false, nil
	}

	next := fake.reads[0]
	fake.reads = fake.reads[1:]

	return next, true, nil
}

func (fake *fakeWinHIDCalls) cancelRead() error {
	fake.calls = append(fake.calls, "CancelIoEx")

	return fake.cancelResult
}

func (fake *fakeWinHIDCalls) release() error {
	fake.calls = append(fake.calls, "CloseHandle")

	return nil
}

func (fake *fakeWinHIDCalls) count(call string) int {
	found := 0

	for _, made := range fake.calls {
		if made == call {
			found++
		}
	}

	return found
}

func fakeLengths() reportLengths {
	return reportLengths{input: fakeInputLength, output: fakeOutputLength, feature: fakeFeatureLength}
}

func removalError() error {
	return callError("WriteFile", windowsErrorDeviceNotConnected, errFakeWindowsCall)
}

func TestWinHID_WriteSendsAnOutputReportPaddedToTheOutputLength(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	writeError := opened.write([]byte{0x02, 0x07, 0x01})
	if writeError != nil {
		t.Fatal(writeError)
	}

	if len(fake.calls) != 1 || fake.calls[0] != "WriteFile" {
		t.Fatalf("want one WriteFile, got %v", fake.calls)
	}

	sent := fake.written[0]
	if len(sent) != fakeOutputLength || !bytes.Equal(sent[:3], []byte{0x02, 0x07, 0x01}) || sent[fakeOutputLength-1] != 0 {
		t.Errorf("want the report, its ID first, zero padded to %d bytes; got %d bytes starting %v",
			fakeOutputLength, len(sent), sent[:min(len(sent), 4)])
	}
}

func TestWinHID_SendFeatureSendsAFeatureReportPaddedToTheFeatureLength(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	sendError := opened.sendFeature([]byte{0x03, 0x08, 70})
	if sendError != nil {
		t.Fatal(sendError)
	}

	if len(fake.calls) != 1 || fake.calls[0] != "HidD_SetFeature" {
		t.Fatalf("want one HidD_SetFeature, got %v", fake.calls)
	}

	sent := fake.written[0]
	if len(sent) != fakeFeatureLength || !bytes.Equal(sent[:3], []byte{0x03, 0x08, 70}) {
		t.Errorf("want the report, its ID first, zero padded to %d bytes; got %v", fakeFeatureLength, sent)
	}
}

func TestWinHID_GetFeatureAsksWithTheReportIDFirstAndReturnsTheLengthAskedFor(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	report, getError := opened.getFeature(fakeReportID, fakeFeatureLength)
	if getError != nil {
		t.Fatal(getError)
	}

	if len(fake.calls) != 1 || fake.calls[0] != "HidD_GetFeature" {
		t.Fatalf("want one HidD_GetFeature, got %v", fake.calls)
	}

	if asked := fake.written[0]; len(asked) != fakeFeatureLength || asked[0] != fakeReportID {
		t.Errorf("want a %d byte request starting with ID %#x, got %v", fakeFeatureLength, fakeReportID, asked)
	}

	if len(report) != fakeFeatureLength || report[0] != fakeReportID || report[1] != 1 {
		t.Errorf("want %d bytes starting with the ID then what Windows filled in, got %v", fakeFeatureLength, report)
	}
}

func TestWinHID_GetFeatureShorterThanTheDeviceAsksForTheWholeReport(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	const shorter = 17

	report, getError := opened.getFeature(fakeInputID, shorter)
	if getError != nil || len(report) != shorter || report[0] != fakeInputID {
		t.Fatalf("want %d bytes starting with the ID, got %v (%v)", shorter, report, getError)
	}

	if len(fake.written[0]) != fakeFeatureLength {
		t.Errorf("want Windows given room for the device's %d byte report, got %d", fakeFeatureLength, len(fake.written[0]))
	}
}

func TestWinHID_GetFeatureRefusesALengthWithNoRoomForTheID(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	_, getError := opened.getFeature(fakeReportID, 0)
	if !errors.Is(getError, errFeatureLength) || len(fake.calls) != 0 {
		t.Errorf("want errFeatureLength and no call, got %v after %v", getError, fake.calls)
	}
}

func TestWinHID_ReadWithoutWaiting_LeavesAnUnfinishedReadPending(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	for range 3 {
		readLength, readError := opened.readWithoutWaiting(make([]byte, fakeInputLength))
		if readLength != 0 || readError != nil {
			t.Fatalf("want 0 and no error, got %d (%v)", readLength, readError)
		}
	}

	if started := fake.count("ReadFile"); started != 1 {
		t.Errorf("want one read started and kept pending, got %d started", started)
	}
}

func TestWinHID_ReadWithoutWaiting_CopiesAFinishedReportWithItsIDAndStartsTheNext(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	fake.readFinished = true
	fake.reads = [][]byte{{fakeInputID, 0x00, 0x00, 0x01}}
	opened := newWinHIDHandle(fake, fakeLengths())

	buffer := make([]byte, fakeInputLength)

	readLength, readError := opened.readWithoutWaiting(buffer)
	if readError != nil || !bytes.Equal(buffer[:readLength], []byte{fakeInputID, 0x00, 0x00, 0x01}) {
		t.Fatalf("want the report with its ID, got %v (%v)", buffer[:readLength], readError)
	}

	if started := fake.count("ReadFile"); started != 2 || !opened.pending {
		t.Errorf("want the next read started at once, got %d started, pending %v", started, opened.pending)
	}
}

func TestWinHID_ReadWithoutWaiting_LeavesOutAZeroReportID(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	fake.readFinished = true
	fake.reads = [][]byte{{0x00, 0x05, 0x06}}
	opened := newWinHIDHandle(fake, fakeLengths())

	buffer := make([]byte, fakeInputLength)

	readLength, readError := opened.readWithoutWaiting(buffer)
	if readError != nil || !bytes.Equal(buffer[:readLength], []byte{0x05, 0x06}) {
		t.Errorf("want the report less Windows' zero ID, got %v (%v)", buffer[:readLength], readError)
	}
}

func TestWinHID_CopyInputReport_KeepsWhatFitsTheBuffer(t *testing.T) {
	t.Parallel()

	buffer := make([]byte, 2)

	copied := copyInputReport(buffer, []byte{fakeInputID, 0x09, 0x0A})
	if copied != 2 || !bytes.Equal(buffer, []byte{fakeInputID, 0x09}) {
		t.Errorf("want the first two bytes, got %d: %v", copied, buffer)
	}

	if copyInputReport(buffer, nil) != 0 {
		t.Error("want nothing copied from an empty read")
	}
}

func TestWinHID_RemovalCodesMapToErrHandleGone(t *testing.T) {
	t.Parallel()

	for _, code := range []uintptr{windowsErrorDeviceNotConnected, windowsErrorGenFailure, windowsErrorInvalidHandle} {
		mapped := callError("WriteFile", code, errFakeWindowsCall)
		if !errors.Is(mapped, errHandleGone) || !errors.Is(mapped, errFakeWindowsCall) {
			t.Errorf("code %d: want errHandleGone wrapping the cause, got %v", code, mapped)
		}
	}

	for _, code := range []uintptr{0, codeAccessDenied, codeIOIncomplete} {
		mapped := callError("WriteFile", code, errFakeWindowsCall)
		if errors.Is(mapped, errHandleGone) || !errors.Is(mapped, errFakeWindowsCall) {
			t.Errorf("code %d: want the cause and not errHandleGone, got %v", code, mapped)
		}
	}
}

func TestWinHID_EveryCallAfterARemovalReturnsErrHandleGone(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	fake.failWith = removalError()
	opened := newWinHIDHandle(fake, fakeLengths())

	firstError := opened.write([]byte{0x02})
	if !errors.Is(firstError, errHandleGone) {
		t.Fatalf("want errHandleGone from the call that met the removal, got %v", firstError)
	}

	fake.failWith = nil
	callsBefore := len(fake.calls)

	_, getError := opened.getFeature(fakeReportID, fakeFeatureLength)
	_, readError := opened.readWithoutWaiting(make([]byte, fakeInputLength))

	later := map[string]error{
		"write":        opened.write([]byte{0x02}),
		"send feature": opened.sendFeature([]byte{0x03}),
		"get feature":  getError,
		"read":         readError,
	}

	for call, callResult := range later {
		if !errors.Is(callResult, errHandleGone) {
			t.Errorf("%s after removal: want errHandleGone, got %v", call, callResult)
		}
	}

	if len(fake.calls) != callsBefore {
		t.Errorf("want no Windows call once the device is gone, got %v", fake.calls[callsBefore:])
	}

	closeError := opened.close()
	if !errors.Is(closeError, errHandleGone) || fake.count("CloseHandle") != 1 {
		t.Errorf("want the handle released and errHandleGone, got %v after %v", closeError, fake.calls)
	}
}

func TestWinHID_ARemovedPendingReadMarksTheHandleGone(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	_, _ = opened.readWithoutWaiting(make([]byte, fakeInputLength))

	fake.failWith = callError("GetOverlappedResult", windowsErrorDeviceNotConnected, errFakeWindowsCall)

	_, readError := opened.readWithoutWaiting(make([]byte, fakeInputLength))
	if !errors.Is(readError, errHandleGone) {
		t.Fatalf("want errHandleGone from the read, got %v", readError)
	}

	fake.failWith = nil

	writeError := opened.write([]byte{0x02})
	if !errors.Is(writeError, errHandleGone) {
		t.Errorf("want errHandleGone from a write after, got %v", writeError)
	}
}

func TestWinHID_CloseCancelsThePendingReadBeforeReleasing(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	_, _ = opened.readWithoutWaiting(make([]byte, fakeInputLength))

	closeError := opened.close()
	if closeError != nil {
		t.Fatal(closeError)
	}

	last := fake.calls[len(fake.calls)-2:]
	if last[0] != "CancelIoEx" || last[1] != "CloseHandle" {
		t.Errorf("want CancelIoEx then CloseHandle, got %v", fake.calls)
	}
}

func TestWinHID_CloseWithNothingPendingOnlyReleases(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	closeError := opened.close()
	if closeError != nil || len(fake.calls) != 1 || fake.calls[0] != "CloseHandle" {
		t.Errorf("want only CloseHandle, got %v (%v)", fake.calls, closeError)
	}
}

func TestWinHID_CloseReportsADeviceRemovedUnderThePendingRead(t *testing.T) {
	t.Parallel()

	fake := newFakeWinHIDCalls()
	opened := newWinHIDHandle(fake, fakeLengths())

	_, _ = opened.readWithoutWaiting(make([]byte, fakeInputLength))

	fake.cancelResult = callError("GetOverlappedResult", windowsErrorDeviceNotConnected, errFakeWindowsCall)

	closeError := opened.close()
	if !errors.Is(closeError, errHandleGone) || fake.count("CloseHandle") != 1 {
		t.Errorf("want the handle released and errHandleGone, got %v after %v", closeError, fake.calls)
	}
}

func TestWinHID_PadReport(t *testing.T) {
	t.Parallel()

	long := []byte{1, 2, 3, 4}
	if padded := padReport(long, 3); !bytes.Equal(padded, long) {
		t.Errorf("want a long enough report unchanged, got %v", padded)
	}

	padded := padReport([]byte{9, 8}, 4)
	if !bytes.Equal(padded, []byte{9, 8, 0, 0}) {
		t.Errorf("want zero padding after the report, got %v", padded)
	}
}

func TestWinHID_ConnectionsOfVendor_KeepsOnlyThatVendor(t *testing.T) {
	t.Parallel()

	found := []hidInterface{
		{path: `\\?\hid#vid_046d&pid_c31c#keyboard`, vendorID: otherTestVendor, productID: keyboardProductID},
		{path: `\\?\hid#vid_0fd9&pid_006c#deck`, vendorID: elgatoTestVendor, productID: xlProductID},
	}

	kept := connectionsOfVendor(found, elgatoTestVendor)
	if len(kept) != 1 || kept[0].Path != found[1].path || kept[0].VendorID != elgatoTestVendor ||
		kept[0].ProductID != xlProductID {
		t.Errorf("want only the deck, its path, vendor and product, got %+v", kept)
	}

	if len(connectionsOfVendor(found, absentTestVendor)) != 0 {
		t.Error("want nothing for a vendor with nothing attached")
	}

	if len(connectionsOfVendor(found, 0)) != len(found) {
		t.Error("want vendor zero to keep every interface, as in hidapi")
	}
}

const absentTestVendor = 0xFFFE

func TestWinHID_InterfaceDetail_SizeFieldAndPath(t *testing.T) {
	t.Parallel()

	const path = `\\?\hid#vid_0fd9&pid_006c#7&1a2b3c4d&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`

	encoded := utf16.Encode([]rune(path))
	required := uint32(len(path)+interfaceDetailPathStart+1) * utf16UnitBytes

	for pointerSize, wantSize := range map[uintptr]uint32{8: 8, 4: 6} {
		detail := newInterfaceDetailBuffer(required, pointerSize)

		if got := uint32(detail[0]) | uint32(detail[1])<<wordBits; got != wantSize {
			t.Errorf("pointer size %d: want the size field %d, got %d", pointerSize, wantSize, got)
		}

		if len(detail)*utf16UnitBytes < int(required) {
			t.Fatalf("pointer size %d: want room for %d bytes, got %d", pointerSize, required, len(detail)*utf16UnitBytes)
		}

		copy(detail[interfaceDetailPathStart:], encoded)

		if got := pathFromInterfaceDetail(detail); got != path {
			t.Errorf("pointer size %d: want the path %q, got %q", pointerSize, path, got)
		}
	}

	if pathFromInterfaceDetail(nil) != "" {
		t.Error("want no path from an empty detail")
	}
}

func TestWinHID_ResultMasks(t *testing.T) {
	t.Parallel()

	const highGarbage = 0xDEAD_BEEF_0000_0000

	if booleanSucceeded(highGarbage|0x100) || !booleanSucceeded(highGarbage|1) {
		t.Error("want a BOOLEAN read from its low byte only")
	}

	if boolSucceeded(uintptr(highGarbage)) || !boolSucceeded(highGarbage|1) {
		t.Error("want a BOOL read from its low 32 bits only")
	}

	if !ntStatusIs(highGarbage|hidpStatusSuccess, hidpStatusSuccess) || ntStatusIs(0, hidpStatusSuccess) {
		t.Error("want an NTSTATUS read from its low 32 bits")
	}
}

// TestWinHID_StructureLayouts checks the structures Windows fills against the
// sizes and offsets the Windows headers give on 64 bit Windows, which a 64
// bit test system shares.
func TestWinHID_StructureLayouts(t *testing.T) {
	t.Parallel()

	if unsafe.Sizeof(uintptr(0)) != pointerSize64 {
		t.Skip("the sizes checked are 64 bit Windows'")
	}

	var (
		guid       windowsGUID
		attributes hidAttributes
		interfaces deviceInterfaceData
	)

	capabilities := hidCapabilities{inputReportByteLength: 512, outputReportByteLength: 1024, featureReportByteLength: 32}

	layouts := map[string][2]uintptr{
		"GUID size":                         {unsafe.Sizeof(guid), 16},
		"HIDD_ATTRIBUTES size":              {unsafe.Sizeof(attributes), 12},
		"HIDD_ATTRIBUTES.Size":              {unsafe.Offsetof(attributes.size), 0},
		"HIDD_ATTRIBUTES.VendorID":          {unsafe.Offsetof(attributes.vendorID), 4},
		"HIDD_ATTRIBUTES.ProductID":         {unsafe.Offsetof(attributes.productID), 6},
		"HIDP_CAPS size":                    {unsafe.Sizeof(capabilities), 64},
		"HIDP_CAPS.InputReportByteLength":   {unsafe.Offsetof(capabilities.inputReportByteLength), 4},
		"HIDP_CAPS.OutputReportByteLength":  {unsafe.Offsetof(capabilities.outputReportByteLength), 6},
		"HIDP_CAPS.FeatureReportByteLength": {unsafe.Offsetof(capabilities.featureReportByteLength), 8},
		"SP_DEVICE_INTERFACE_DATA size":     {unsafe.Sizeof(interfaces), 32},
		"SP_DEVICE_INTERFACE_DATA.cbSize":   {unsafe.Offsetof(interfaces.size), 0},
	}

	for name, layout := range layouts {
		if layout[0] != layout[1] {
			t.Errorf("%s: want %d bytes, got %d", name, layout[1], layout[0])
		}
	}

	if lengths := capabilities.lengths(); lengths != (reportLengths{input: 512, output: 1024, feature: 32}) {
		t.Errorf("want the three lengths read, got %+v", lengths)
	}
}

func TestWinHID_BindRecovering(t *testing.T) {
	t.Parallel()

	bindError := bindRecovering(func() error { return nil })
	if bindError != nil {
		t.Errorf("want nil when every function is found, got %v", bindError)
	}

	missing := bindRecovering(func() error { return errFakeMissingProc })
	if !errors.Is(missing, errWinHIDUnavailable) || !errors.Is(missing, errFakeMissingProc) {
		t.Errorf("want errWinHIDUnavailable wrapping the cause, got %v", missing)
	}

	panicked := bindRecovering(func() error { panic("Failed to find HidD_GetHidGuid procedure in hid.dll") })
	if !errors.Is(panicked, errWinHIDUnavailable) {
		t.Errorf("want a panic reported as errWinHIDUnavailable, got %v", panicked)
	}
}
