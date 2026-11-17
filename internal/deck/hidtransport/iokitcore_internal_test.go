package hidtransport

import (
	"bytes"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// The macOS driver's own logic, run on every system against fake IOKit
// calls. The IOKit calls themselves are checked on the Mac
// (iokit_darwin_test.go), and on the deck by the driver contract.

const (
	iokitTestEntryID      = 4294968334
	iokitTestPath         = "DevSrvsID:4294968334"
	iokitTestReportID     = 0x03
	iokitTestFeatureID    = 0x06
	iokitTestFeatureBytes = 32
	iokitTestXLProduct    = 0x006C
	iokitTestPage         = 0x0C
	iokitTestUsage        = 0x01
	iokitTestOtherPage    = 0x01
	iokitTestOtherUsage   = 0x06
)

// sentReport is one IOHIDDeviceSetReport call the fake received.
type sentReport struct {
	reportType iokitReportType
	reportID   byte
	payload    []byte
}

// fakeIOKitCalls stands in for one open device's IOKit calls.
type fakeIOKitCalls struct {
	sent          []sentReport
	setResult     int32
	gotType       iokitReportType
	gotID         byte
	gotLength     int
	getResult     int32
	getFill       []byte
	releaseResult int32
	releases      int
}

func newFakeIOKitCalls() *fakeIOKitCalls {
	return &fakeIOKitCalls{
		sent: nil, setResult: ioReturnSuccess, gotType: 0, gotID: 0, gotLength: 0,
		getResult: ioReturnSuccess, getFill: nil, releaseResult: ioReturnSuccess, releases: 0,
	}
}

func (fake *fakeIOKitCalls) setReport(reportType iokitReportType, reportID byte, payload []byte) int32 {
	fake.sent = append(fake.sent, sentReport{reportType: reportType, reportID: reportID, payload: bytes.Clone(payload)})

	return fake.setResult
}

func (fake *fakeIOKitCalls) getReport(reportType iokitReportType, reportID byte, buffer []byte) (int, int32) {
	fake.gotType, fake.gotID, fake.gotLength = reportType, reportID, len(buffer)

	return copy(buffer, fake.getFill), fake.getResult
}

func (fake *fakeIOKitCalls) release() int32 {
	fake.releases++

	return fake.releaseResult
}

// asIOReturn gives a code as the 32 bit result IOKit returns.
func asIOReturn(code uint32) int32 {
	return int32(code) //nolint:gosec // the same 32 bits, as IOKit returns them
}

func newTestIOKitHandle(calls *fakeIOKitCalls) *iokitHandle {
	return newIOKitHandle(calls, reportLengths{input: 512, output: 1024, feature: iokitTestFeatureBytes}, nil)
}

func TestIOKitPath_IsWrittenAsHIDAPIWritesIt(t *testing.T) {
	t.Parallel()

	if got := iokitPath(iokitTestEntryID); got != iokitTestPath {
		t.Errorf("want %q, got %q", iokitTestPath, got)
	}

	if got := iokitPath(^uint64(0)); got != "DevSrvsID:18446744073709551615" {
		t.Errorf("want the largest ID in decimal, got %q", got)
	}
}

func TestIOKitPath_ReadsBackTheEntryID(t *testing.T) {
	t.Parallel()

	for _, entryID := range []uint64{0, 1, iokitTestEntryID, ^uint64(0)} {
		got, parseError := parseIOKitPath(iokitPath(entryID))
		if parseError != nil || got != entryID {
			t.Errorf("entry %d: got %d (%v)", entryID, got, parseError)
		}
	}
}

func TestIOKitPath_RefusesAnyOtherForm(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"", "DevSrvsID:", "DevSrvsID:12ab", "DevSrvsID: 12", "DevSrvsID:+12", "DevSrvsID:-1",
		"devsrvsid:12", "DevSrvsID:0x1F", "DevSrvsID:18446744073709551616",
		"IOService:/AppleACPIPlatformExpert/XHC1@14", `\\?\hid#vid_0fd9&pid_006c`,
	} {
		_, parseError := parseIOKitPath(path)
		if !errors.Is(parseError, errNotIOKitPath) {
			t.Errorf("%q: want errNotIOKitPath, got %v", path, parseError)
		}
	}
}

func TestIOKitHandle_WritesAnOutputReportWithItsID(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	report := []byte{iokitTestReportID, 0x07, 0x08}

	writeError := newTestIOKitHandle(calls).write(report)
	if writeError != nil {
		t.Fatal(writeError)
	}

	want := sentReport{reportType: reportTypeOutput, reportID: iokitTestReportID, payload: report}
	if len(calls.sent) != 1 || !sameSent(calls.sent[0], want) {
		t.Errorf("want %+v, got %+v", want, calls.sent)
	}
}

func TestIOKitHandle_SendsAFeatureReportWithItsID(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	report := []byte{iokitTestFeatureID, 0x46}

	sendError := newTestIOKitHandle(calls).sendFeature(report)
	if sendError != nil {
		t.Fatal(sendError)
	}

	want := sentReport{reportType: reportTypeFeature, reportID: iokitTestFeatureID, payload: report}
	if len(calls.sent) != 1 || !sameSent(calls.sent[0], want) {
		t.Errorf("want %+v, got %+v", want, calls.sent)
	}
}

// TestIOKitHandle_LeavesOutReportIDZero follows hidapi: ID zero means the
// device numbers no reports, so that byte is not sent.
func TestIOKitHandle_LeavesOutReportIDZero(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	opened := newTestIOKitHandle(calls)

	writeError := opened.write([]byte{0, 0x0A, 0x0B})
	if writeError != nil {
		t.Fatal(writeError)
	}

	sendError := opened.sendFeature([]byte{0})
	if sendError != nil {
		t.Fatal(sendError)
	}

	wants := []sentReport{
		{reportType: reportTypeOutput, reportID: 0, payload: []byte{0x0A, 0x0B}},
		{reportType: reportTypeFeature, reportID: 0, payload: []byte{}},
	}
	if len(calls.sent) != len(wants) {
		t.Fatalf("want %d calls, got %+v", len(wants), calls.sent)
	}

	for index, want := range wants {
		if !sameSent(calls.sent[index], want) {
			t.Errorf("call %d: want %+v, got %+v", index, want, calls.sent[index])
		}
	}
}

func TestIOKitHandle_RefusesAnEmptyReport(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	opened := newTestIOKitHandle(calls)

	writeError := opened.write(nil)
	if !errors.Is(writeError, errIOKitEmptyReport) {
		t.Errorf("write: want errIOKitEmptyReport, got %v", writeError)
	}

	sendError := opened.sendFeature([]byte{})
	if !errors.Is(sendError, errIOKitEmptyReport) {
		t.Errorf("send feature: want errIOKitEmptyReport, got %v", sendError)
	}

	if len(calls.sent) != 0 {
		t.Errorf("want nothing sent, got %+v", calls.sent)
	}
}

func TestIOKitHandle_GetsANumberedFeatureReport(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	calls.getFill = []byte{iokitTestFeatureID, 0x0C, 'A', 'B'}

	report, getError := newTestIOKitHandle(calls).getFeature(iokitTestFeatureID, iokitTestFeatureBytes)
	if getError != nil {
		t.Fatal(getError)
	}

	asked := calls.gotType == reportTypeFeature && calls.gotID == iokitTestFeatureID
	if !asked || calls.gotLength != iokitTestFeatureBytes {
		t.Errorf("want a feature report %#x into %d bytes, got type %d ID %#x into %d",
			iokitTestFeatureID, iokitTestFeatureBytes, calls.gotType, calls.gotID, calls.gotLength)
	}

	if !bytes.Equal(report, calls.getFill) {
		t.Errorf("want %v, got %v", calls.getFill, report)
	}
}

// TestIOKitHandle_GetsFeatureReportZeroAfterItsID follows hidapi: for ID zero
// IOKit fills the buffer after the ID byte, which is counted in the result.
func TestIOKitHandle_GetsFeatureReportZeroAfterItsID(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	calls.getFill = []byte{0x11, 0x22}

	report, getError := newTestIOKitHandle(calls).getFeature(0, 4)
	if getError != nil {
		t.Fatal(getError)
	}

	if calls.gotLength != 3 || !bytes.Equal(report, []byte{0, 0x11, 0x22}) {
		t.Errorf("want 3 bytes asked for and [0 17 34], got %d and %v", calls.gotLength, report)
	}
}

func TestIOKitHandle_RefusesAFeatureReportWithNoRoomForItsID(t *testing.T) {
	t.Parallel()

	_, getError := newTestIOKitHandle(newFakeIOKitCalls()).getFeature(1, 0)
	if !errors.Is(getError, errFeatureLength) {
		t.Errorf("want errFeatureLength, got %v", getError)
	}
}

func TestIOKitFeatureResult_NeverRunsPastTheBuffer(t *testing.T) {
	t.Parallel()

	buffer := []byte{0, 1, 2}

	if got := iokitFeatureResult(buffer, 0, len(buffer)); len(got) != len(buffer) {
		t.Errorf("want at most %d bytes, got %v", len(buffer), got)
	}

	if got := iokitFeatureResult(buffer, 1, -1); len(got) != 0 {
		t.Errorf("want nothing for a negative length, got %v", got)
	}
}

// TestIOKitHandle_KeepsTheIOReturnCode checks the general error a write to the
// deck now and then meets reaches the watcher's retry with its code.
func TestIOKitHandle_KeepsTheIOReturnCode(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	calls.setResult = asIOReturn(ioReturnGeneralError)
	calls.getResult = asIOReturn(ioReturnGeneralError)
	opened := newTestIOKitHandle(calls)

	_, getError := opened.getFeature(iokitTestFeatureID, iokitTestFeatureBytes)

	for call, callError := range map[string]error{
		"output report":      opened.write([]byte{iokitTestReportID}),
		"feature report out": opened.sendFeature([]byte{iokitTestFeatureID}),
		"feature report in":  getError,
	} {
		failure, isFailure := errors.AsType[ioReturnError](callError)
		if !isFailure || failure.code != ioReturnGeneralError {
			t.Errorf("%s: want IOReturn 0xE00002BC, got %v", call, callError)
		}

		if errors.Is(callError, errHandleGone) {
			t.Errorf("%s: the general error is not a removal: %v", call, callError)
		}
	}

	if opened.gone.Load() {
		t.Error("want the handle still usable after a general error")
	}
}

func TestIOKitHandle_ReportsTheDeviceGoneFromItsIOReturn(t *testing.T) {
	t.Parallel()

	for _, code := range []uint32{ioReturnNoDevice, ioReturnNotAttached} {
		calls := newFakeIOKitCalls()
		calls.setResult = asIOReturn(code)
		opened := newTestIOKitHandle(calls)

		writeError := opened.write([]byte{iokitTestReportID})

		failure, isFailure := errors.AsType[ioReturnError](writeError)
		if !isFailure || failure.code != code {
			t.Errorf("%#x: want the code kept, got %v", code, writeError)
		}

		calls.setResult = ioReturnSuccess
		checkEveryCallGone(t, opened, calls)
	}
}

func TestIOKitHandle_ReportsEveryCallGoneOnceRemoved(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	opened := newTestIOKitHandle(calls)
	opened.markGone()

	checkEveryCallGone(t, opened, calls)
}

func checkEveryCallGone(t *testing.T, opened *iokitHandle, calls *fakeIOKitCalls) {
	t.Helper()

	sentBefore := len(calls.sent)
	_, getError := opened.getFeature(iokitTestFeatureID, iokitTestFeatureBytes)
	_, readError := opened.readWithoutWaiting(make([]byte, 8))

	for call, callError := range map[string]error{
		"output report":      opened.write([]byte{iokitTestReportID}),
		"feature report out": opened.sendFeature([]byte{iokitTestFeatureID}),
		"feature report in":  getError,
		"input report":       readError,
		"closing":            opened.close(),
	} {
		if !errors.Is(callError, errHandleGone) {
			t.Errorf("%s: want errHandleGone, got %v", call, callError)
		}
	}

	if len(calls.sent) != sentBefore || calls.releases != 1 {
		t.Errorf("want nothing sent once gone and one release, got %d sent and %d releases",
			len(calls.sent)-sentBefore, calls.releases)
	}
}

func TestIOKitHandle_ReadsNothingWithNoQueueAndClosesOnce(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	opened := newTestIOKitHandle(calls)

	readLength, readError := opened.readWithoutWaiting(make([]byte, 8))
	if readLength != 0 || readError != nil {
		t.Errorf("want 0 and no error, got %d (%v)", readLength, readError)
	}

	closeError := opened.close()
	if closeError != nil || calls.releases != 1 {
		t.Errorf("want one release and no error, got %d (%v)", calls.releases, closeError)
	}
}

func TestIOKitHandle_ReportsAFailedClose(t *testing.T) {
	t.Parallel()

	calls := newFakeIOKitCalls()
	calls.releaseResult = asIOReturn(ioReturnGeneralError)

	closeError := newTestIOKitHandle(calls).close()

	failure, isFailure := errors.AsType[ioReturnError](closeError)
	if !isFailure || failure.code != ioReturnGeneralError {
		t.Errorf("want IOReturn 0xE00002BC, got %v", closeError)
	}
}

func TestIOReturnFailure_NamesTheCallAndCode(t *testing.T) {
	t.Parallel()

	failure := checkIOReturn("IOHIDDeviceSetReport", asIOReturn(ioReturnGeneralError))
	if failure == nil || failure.Error() != "IOHIDDeviceSetReport failed: IOReturn 0xE00002BC" {
		t.Errorf("got %v", failure)
	}

	if checkIOReturn("IOHIDDeviceSetReport", ioReturnSuccess) != nil {
		t.Error("want no error for kIOReturnSuccess")
	}
}

func TestIOKitListing_GivesOneConnectionPerUsageAsHIDAPIDoes(t *testing.T) {
	t.Parallel()

	primary := usagePair{page: iokitTestPage, usage: iokitTestUsage}
	other := usagePair{page: iokitTestOtherPage, usage: iokitTestOtherUsage}
	connection := deck.Connection{Path: iokitTestPath, VendorID: elgatoVendorID, ProductID: iokitTestXLProduct}

	for _, example := range []struct {
		name  string
		pairs []usagePair
		want  int
	}{
		{"no usage pairs", nil, 1},
		{"only the primary", []usagePair{primary}, 1},
		{"the primary and another", []usagePair{primary, other}, 2},
		{"another repeated", []usagePair{other, primary, other}, 3},
	} {
		listing := iokitListing{
			vendorID: elgatoVendorID, productID: iokitTestXLProduct, path: iokitTestPath,
			primary: primary, pairs: example.pairs,
		}

		got := listing.connections()
		if len(got) != example.want {
			t.Errorf("%s: want %d connections, got %v", example.name, example.want, got)
		}

		for _, each := range got {
			if each != connection {
				t.Errorf("%s: want %+v, got %+v", example.name, connection, each)
			}
		}
	}
}

func TestLowSixteenBits_TruncatesAsHIDAPIDoes(t *testing.T) {
	t.Parallel()

	for value, want := range map[int64]uint16{0x0FD9: 0x0FD9, 0x10FD9: 0x0FD9, -1: 0xFFFF, 0: 0} {
		if got := lowSixteenBits(value); got != want {
			t.Errorf("%#x: want %#x, got %#x", value, want, got)
		}
	}
}

func sameSent(got, want sentReport) bool {
	return got.reportType == want.reportType && got.reportID == want.reportID && bytes.Equal(got.payload, want.payload)
}

func TestRunLoopOutcome_FollowsHIDAPIsReadThread(t *testing.T) {
	t.Parallel()

	for _, example := range []struct {
		result   int32
		stopping bool
		over     bool
		gone     bool
	}{
		{runLoopRunTimedOut, false, false, false},
		{runLoopRunHandledSource, false, false, false},
		{runLoopRunStopped, false, true, true},
		{runLoopRunFinished, false, true, true},
		{0, false, true, true},
		{runLoopRunStopped, true, true, false},
		{runLoopRunTimedOut, true, true, false},
	} {
		over, gone := runLoopOutcome(example.result, example.stopping)
		if over != example.over || gone != example.gone {
			t.Errorf("result %d, stopping %v: want over %v gone %v, got %v %v",
				example.result, example.stopping, example.over, example.gone, over, gone)
		}
	}
}

func TestInputRegistry_QueuesEachReportForItsDevice(t *testing.T) {
	t.Parallel()

	registry := newInputRegistry()
	first, second := newReportQueue(), newReportQueue()
	firstContext := registry.add(inputTarget{queue: first, removed: nil})
	secondContext := registry.add(inputTarget{queue: second, removed: nil})

	if firstContext == 0 || firstContext == secondContext {
		t.Fatalf("want distinct contexts other than zero, got %d and %d", firstContext, secondContext)
	}

	registry.deliverReport(secondContext, []byte{0x01, 0x02})
	registry.deliverReport(secondContext+1, []byte{0x09})

	buffer := make([]byte, queueTestBuffer)
	if first.length() != 0 || second.take(buffer) != 2 || !bytes.Equal(buffer[:2], []byte{0x01, 0x02}) {
		t.Errorf("want the report in the second queue only, got %d first and %v", first.length(), buffer[:2])
	}

	registry.remove(secondContext)
	registry.deliverReport(secondContext, []byte{0x03})

	if second.length() != 0 {
		t.Errorf("want reports for a removed device ignored, got %d", second.length())
	}
}

func TestInputRegistry_DeliversRemoval(t *testing.T) {
	t.Parallel()

	registry := newInputRegistry()
	removals := 0
	context := registry.add(inputTarget{queue: nil, removed: func() { removals++ }})

	registry.deliverRemoval(context)
	registry.deliverRemoval(context + 1)

	if removals != 1 {
		t.Errorf("want one removal, got %d", removals)
	}
}

// TestIOKitHandle_ReadsQueuedReportsThenReportsRemoval follows hidapi: reports
// already queued are given before the removal is.
func TestIOKitHandle_ReadsQueuedReportsThenReportsRemoval(t *testing.T) {
	t.Parallel()

	queue := newReportQueue()
	opened := newIOKitHandle(newFakeIOKitCalls(), reportLengths{input: 512, output: 1024, feature: 32}, queue)
	registry := newInputRegistry()
	context := registry.add(inputTarget{queue: queue, removed: opened.markGone})
	buffer := make([]byte, queueTestBuffer)

	readLength, readError := opened.readWithoutWaiting(buffer)
	if readLength != 0 || readError != nil {
		t.Errorf("empty: want 0 and no error, got %d (%v)", readLength, readError)
	}

	registry.deliverReport(context, []byte{0x01, 0x00, 0x08})
	registry.deliverRemoval(context)

	readLength, readError = opened.readWithoutWaiting(buffer)
	if readError != nil || !bytes.Equal(buffer[:readLength], []byte{0x01, 0x00, 0x08}) {
		t.Errorf("queued: want the report, got %v (%v)", buffer[:readLength], readError)
	}

	_, readError = opened.readWithoutWaiting(buffer)
	if !errors.Is(readError, errHandleGone) {
		t.Errorf("after removal: want errHandleGone, got %v", readError)
	}

	writeError := opened.write([]byte{iokitTestReportID})
	if !errors.Is(writeError, errHandleGone) {
		t.Errorf("write after removal: want errHandleGone, got %v", writeError)
	}
}

// fakeRunLoop stands in for a device's run loop: each run waits until the
// test takes a slice from slices, when it times out, as a run loop with a
// device scheduled does, or until the loop is stopped.
type fakeRunLoop struct {
	prepared    atomic.Bool
	interrupted atomic.Bool
	finished    atomic.Bool
	stopped     chan struct{}
	slices      chan struct{}
}

func newFakeRunLoop() *fakeRunLoop {
	return &fakeRunLoop{
		prepared: atomic.Bool{}, interrupted: atomic.Bool{}, finished: atomic.Bool{},
		stopped: make(chan struct{}), slices: make(chan struct{}),
	}
}

func (loop *fakeRunLoop) prepare() { loop.prepared.Store(true) }

func (loop *fakeRunLoop) run() int32 {
	select {
	case <-loop.stopped:
		return runLoopRunStopped
	case loop.slices <- struct{}{}:
		return runLoopRunTimedOut
	}
}

func (loop *fakeRunLoop) interrupt() {
	loop.interrupted.Store(true)
	loop.stopLoop()
}

func (loop *fakeRunLoop) stopLoop() {
	select {
	case <-loop.stopped:
	default:
		close(loop.stopped)
	}
}

func (loop *fakeRunLoop) finish() { loop.finished.Store(true) }

func isClosed(channel <-chan struct{}) bool {
	select {
	case <-channel:
		return true
	default:
		return false
	}
}

func TestInputThread_RunsUntilStoppedThenEnds(t *testing.T) {
	t.Parallel()

	loop := newFakeRunLoop()
	gone := atomic.Int32{}
	thread := startInputThread(loop, func() { gone.Add(1) })

	if !loop.prepared.Load() {
		t.Error("want the device prepared before the thread is used")
	}

	// Two slices taken means the run loop timed out once and the thread ran
	// it again rather than ending.
	<-loop.slices
	<-loop.slices

	if isClosed(thread.done) {
		t.Fatal("want the thread running until stopped")
	}

	thread.stop()

	if !isClosed(thread.done) || !loop.interrupted.Load() || !loop.finished.Load() {
		t.Errorf("want the thread ended, interrupted and finished once stop returns, got %v %v %v",
			isClosed(thread.done), loop.interrupted.Load(), loop.finished.Load())
	}

	if gone.Load() != 0 {
		t.Errorf("want a stop by close not taken for removal, got %d", gone.Load())
	}
}

func TestInputThread_MarksTheDeviceGoneWhenItsRunLoopStopsAlone(t *testing.T) {
	t.Parallel()

	loop := newFakeRunLoop()
	gone := make(chan struct{})
	thread := startInputThread(loop, func() { close(gone) })

	loop.stopLoop()

	<-gone

	if isClosed(thread.done) || loop.finished.Load() {
		t.Error("want the thread kept until close, as the device is still scheduled on its run loop")
	}

	thread.stop()

	if !isClosed(thread.done) || !loop.finished.Load() {
		t.Error("want the thread ended once stopped")
	}
}
