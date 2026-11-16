package hidtransport

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// The macOS HID driver (iokit_darwin.go and iokit_input_darwin.go) keeps its
// IOKit calls thin. What it decides is here, in plain Go built on every
// system, so its tests run anywhere: the path format, the report ID byte,
// IOReturn codes, how many listings a device gives, the run loop's results,
// and the handle's state (removal, the input queue, closing).

// iokitPathPrefix starts every path the driver lists. hidapi's macOS backend
// writes a device's path as this prefix and the decimal registry entry ID of
// its IOHIDDevice service, so a path from either driver opens with either.
const iokitPathPrefix = "DevSrvsID:"

const decimalBase = 10

// iokitPath is the path for the service with the registry entry ID given,
// written as hidapi writes it ("DevSrvsID:%llu").
func iokitPath(entryID uint64) string {
	return iokitPathPrefix + strconv.FormatUint(entryID, decimalBase)
}

// errNotIOKitPath reports a path the driver does not list and cannot open.
var errNotIOKitPath = errors.New("not a DevSrvsID path")

// parseIOKitPath reads the registry entry ID from a path iokitPath wrote.
// hidapi also opens an IOService plane path, which no version of it lists on
// macOS; this driver refuses one, along with anything else not in the
// DevSrvsID form. Only decimal digits follow the prefix: hidapi's strtoull
// would also take leading spaces or a sign, which iokitPath never writes.
func parseIOKitPath(path string) (uint64, error) {
	digits, prefixed := strings.CutPrefix(path, iokitPathPrefix)
	if !prefixed {
		return 0, fmt.Errorf("%w: %q", errNotIOKitPath, path)
	}

	for _, character := range digits {
		if character < '0' || character > '9' {
			return 0, fmt.Errorf("%w: %q", errNotIOKitPath, path)
		}
	}

	entryID, parseError := strconv.ParseUint(digits, decimalBase, 64)
	if parseError != nil {
		return 0, fmt.Errorf("%w: %q: %w", errNotIOKitPath, path, parseError)
	}

	return entryID, nil
}

// The IOReturn codes the driver tells apart, from IOKit's IOReturn.h.
// kIOReturnError is the general error a write to the deck now and then
// meets while it is plainly attached (see retryTransient); kIOReturnNoDevice
// and kIOReturnNotAttached mean the device has gone.
const (
	ioReturnSuccess      = 0
	ioReturnGeneralError = 0xE00002BC
	ioReturnNoDevice     = 0xE00002C0
	ioReturnNotAttached  = 0xE00002D9
)

// ioReturnError is a failed IOKit call and the IOReturn code it gave.
type ioReturnError struct {
	call string
	code uint32
}

func (failure ioReturnError) Error() string {
	return fmt.Sprintf("%s failed: IOReturn 0x%08X", failure.call, failure.code)
}

// Is makes a code meaning the device has gone match errHandleGone, so the
// watcher reports it as disconnected.
func (failure ioReturnError) Is(target error) bool {
	return target == errHandleGone &&
		(failure.code == ioReturnNoDevice || failure.code == ioReturnNotAttached)
}

// ioReturnCode reads an IOReturn, a 32 bit kern_return_t, from a call's
// result, of which only the low 32 bits mean anything.
func ioReturnCode(result int32) uint32 {
	return uint32(result) //nolint:gosec // the same 32 bits, read as IOReturn.h writes its codes
}

// checkIOReturn is nil for kIOReturnSuccess and otherwise an
// ioReturnError carrying the code.
func checkIOReturn(call string, result int32) error {
	if result == ioReturnSuccess {
		return nil
	}

	return ioReturnError{call: call, code: ioReturnCode(result)}
}

// iokitReportType is IOHIDReportType, from IOHIDKeys.h.
type iokitReportType int32

const (
	reportTypeInput   iokitReportType = 0
	reportTypeOutput  iokitReportType = 1
	reportTypeFeature iokitReportType = 2
)

// errIOKitEmptyReport reports a report with no bytes, not even its ID.
var errIOKitEmptyReport = errors.New("a report needs at least one byte, its ID")

// iokitReportPayload splits a report as hidapi's macOS backend does before
// IOHIDDeviceSetReport: the first byte is the report ID, and for ID zero,
// meaning the device numbers no reports, that byte is left out of what is
// sent. Any other ID stays in the bytes sent, as hidapi sends them.
func iokitReportPayload(report []byte) (byte, []byte, error) {
	if len(report) == 0 {
		return 0, nil, errIOKitEmptyReport
	}

	if report[0] == 0 {
		return 0, report[1:], nil
	}

	return report[0], report, nil
}

// iokitFeatureBuffer is the buffer for IOHIDDeviceGetReport, length bytes
// with the ID first, and the part of it IOKit fills: as hidapi does, the
// whole buffer for a numbered report, whose bytes from IOKit start with the
// ID, and all but the first byte for ID zero.
func iokitFeatureBuffer(reportID byte, length int) ([]byte, []byte, error) {
	if length < 1 {
		return nil, nil, fmt.Errorf("%w: %d asked for", errFeatureLength, length)
	}

	buffer := make([]byte, length)
	buffer[0] = reportID

	if reportID == 0 {
		return buffer, buffer[1:], nil
	}

	return buffer, buffer, nil
}

// iokitFeatureResult is the report read into buffer, given how many bytes of
// it IOKit filled: counting the ID byte left out for ID zero, as hidapi
// does, and never longer than the buffer.
func iokitFeatureResult(buffer []byte, reportID byte, filled int) []byte {
	if reportID == 0 {
		filled++
	}

	return buffer[:max(0, min(filled, len(buffer)))]
}

// usagePair is a HID usage page and usage, as a device's properties give
// them.
type usagePair struct {
	page  int64
	usage int64
}

// iokitListing is what the driver reads of one IOHIDDevice while listing.
type iokitListing struct {
	vendorID  uint16
	productID uint16
	// path is empty when the device's registry entry ID could not be read,
	// as hidapi leaves it.
	path string
	// primary is the PrimaryUsagePage and PrimaryUsage properties.
	primary usagePair
	// pairs are the well formed entries of DeviceUsagePairs, in order.
	pairs []usagePair
}

// connections are the listings hidapi's macOS backend gives for one device:
// one for its primary usage, then one for each usage pair other than the
// primary, all with the same path, vendor and product.
func (listing iokitListing) connections() []deck.Connection {
	connection := deck.Connection{Path: listing.path, VendorID: listing.vendorID, ProductID: listing.productID}
	found := []deck.Connection{connection}

	for _, pair := range listing.pairs {
		if pair != listing.primary {
			found = append(found, connection)
		}
	}

	return found
}

const sixteenBitMask = 0xFFFF

// lowSixteenBits keeps the low 16 bits of a property's value, as hidapi does
// when it reads a vendor or product ID as a 32 bit number into an unsigned
// short.
func lowSixteenBits(value int64) uint16 {
	return uint16(value & sixteenBitMask)
}

// iokitCalls are the IOKit calls one open device makes. Each returns the raw
// IOReturn, so what a code means is decided here, not in the calls.
type iokitCalls interface {
	// setReport sends payload with IOHIDDeviceSetReport.
	setReport(reportType iokitReportType, reportID byte, payload []byte) int32
	// getReport fills buffer with IOHIDDeviceGetReport and returns the
	// bytes IOKit filled.
	getReport(reportType iokitReportType, reportID byte, buffer []byte) (int, int32)
	// release stops the device's run loop thread, if it has one, then
	// closes and releases the device.
	release() int32
}

// iokitHandle is one device open through the macOS driver. Watcher
// serialises the calls on it, but the removal callback runs on the device's
// run loop thread, so removal is marked atomically.
type iokitHandle struct {
	calls   iokitCalls
	lengths reportLengths
	// queue holds the input reports the run loop thread has received.
	queue *reportQueue
	// gone is set by the removal callback, or by the first call whose
	// IOReturn says the device has gone; every later call returns
	// errHandleGone.
	gone atomic.Bool
}

func newIOKitHandle(calls iokitCalls, lengths reportLengths, queue *reportQueue) *iokitHandle {
	return &iokitHandle{calls: calls, lengths: lengths, queue: queue, gone: atomic.Bool{}}
}

// markGone records that the device has been removed.
func (opened *iokitHandle) markGone() {
	opened.gone.Store(true)
}

// noted returns callResult, first marking the handle gone if it says the
// device has gone.
func (opened *iokitHandle) noted(callResult error) error {
	if errors.Is(callResult, errHandleGone) {
		opened.markGone()
	}

	return callResult
}

func (opened *iokitHandle) write(report []byte) error {
	return opened.setReport(reportTypeOutput, report)
}

func (opened *iokitHandle) sendFeature(report []byte) error {
	return opened.setReport(reportTypeFeature, report)
}

func (opened *iokitHandle) setReport(reportType iokitReportType, report []byte) error {
	if opened.gone.Load() {
		return errHandleGone
	}

	reportID, payload, payloadError := iokitReportPayload(report)
	if payloadError != nil {
		return payloadError
	}

	return opened.noted(checkIOReturn("IOHIDDeviceSetReport", opened.calls.setReport(reportType, reportID, payload)))
}

func (opened *iokitHandle) getFeature(reportID byte, length int) ([]byte, error) {
	if opened.gone.Load() {
		return nil, errHandleGone
	}

	buffer, filledPart, bufferError := iokitFeatureBuffer(reportID, length)
	if bufferError != nil {
		return nil, bufferError
	}

	filled, result := opened.calls.getReport(reportTypeFeature, reportID, filledPart)

	getError := opened.noted(checkIOReturn("IOHIDDeviceGetReport", result))
	if getError != nil {
		return nil, getError
	}

	return iokitFeatureResult(buffer, reportID, filled), nil
}

// readWithoutWaiting takes the oldest queued input report. As hidapi does,
// reports already queued are still given after removal; only then does the
// handle report the device gone.
func (opened *iokitHandle) readWithoutWaiting(buffer []byte) (int, error) {
	if opened.queue != nil {
		if taken := opened.queue.take(buffer); taken > 0 {
			return taken, nil
		}
	}

	if opened.gone.Load() {
		return 0, errHandleGone
	}

	return 0, nil
}

// close releases the device. A device found removed gives errHandleGone once
// released, as the watcher expects.
func (opened *iokitHandle) close() error {
	releaseError := opened.noted(checkIOReturn("IOHIDDeviceClose", opened.calls.release()))

	switch {
	case opened.gone.Load() && releaseError != nil:
		return fmt.Errorf("%w: %w", errHandleGone, releaseError)
	case opened.gone.Load():
		return errHandleGone
	default:
		return releaseError
	}
}

// The results of CFRunLoopRunInMode, from CFRunLoop.h.
const (
	runLoopRunFinished      = 1
	runLoopRunStopped       = 2
	runLoopRunTimedOut      = 3
	runLoopRunHandledSource = 4
)

// runLoopOutcome reads one CFRunLoopRunInMode result on a device's run loop
// thread: whether the loop is over, and whether that means the device has
// gone. Asked to stop by close, it is over and nothing is wrong. Otherwise,
// as in hidapi's read thread, a timeout or a handled source runs it again;
// a run loop stopped by the removal callback, or finished for want of
// sources, means the device has gone; and any other result ends it too,
// with the device treated as gone, as hidapi's reads then fail.
func runLoopOutcome(result int32, stopping bool) (bool, bool) {
	switch {
	case stopping:
		return true, false
	case result == runLoopRunTimedOut || result == runLoopRunHandledSource:
		return false, false
	default:
		return true, true
	}
}

// runLoopCalls are the Core Foundation and IOKit calls one device's run loop
// thread makes.
type runLoopCalls interface {
	// prepare runs on the thread before the loop: it takes the thread's run
	// loop, keeps it from running dry, schedules the device on it and
	// registers the input report and removal callbacks.
	prepare()
	// run runs the run loop once, for a short while, and returns
	// CFRunLoopRunInMode's result.
	run() int32
	// interrupt runs on the closing goroutine: it unregisters the callbacks,
	// unschedules the device and stops the run loop.
	interrupt()
	// finish runs on the thread after the loop, undoing prepare.
	finish()
}

// inputThread is one open device's run loop thread: a goroutine locked to
// its operating system thread for its whole life, as a CFRunLoop belongs to
// a thread. The goroutine never unlocks, so Go ends the thread with it,
// taking the run loop with it.
type inputThread struct {
	calls    runLoopCalls
	markGone func()
	stopping atomic.Bool
	// release lets the thread finish once the loop is over; closed by stop.
	release chan struct{}
	// done is closed when the goroutine has returned.
	done chan struct{}
}

// startInputThread starts the thread and returns once prepare has run, so
// the device is scheduled and its callbacks registered before the handle is
// used.
func startInputThread(calls runLoopCalls, markGone func()) *inputThread {
	thread := &inputThread{
		calls: calls, markGone: markGone, stopping: atomic.Bool{},
		release: make(chan struct{}), done: make(chan struct{}),
	}
	ready := make(chan struct{})

	go thread.loop(ready)

	<-ready

	return thread
}

func (thread *inputThread) loop(ready chan<- struct{}) {
	runtime.LockOSThread()

	defer close(thread.done)

	thread.calls.prepare()
	close(ready)

	for {
		over, gone := runLoopOutcome(thread.calls.run(), thread.stopping.Load())
		if gone {
			thread.markGone()
		}

		if over {
			break
		}
	}

	// The device stays scheduled on this run loop until stop has
	// unscheduled it, so the thread, and its run loop, outlive the loop
	// until then, as hidapi's read thread waits for hid_close.
	<-thread.release
	thread.calls.finish()
}

// stop unschedules the device, stops the run loop and waits for the thread
// to end. It is called once, by the device's close.
func (thread *inputThread) stop() {
	thread.stopping.Store(true)
	thread.calls.interrupt()
	close(thread.release)
	<-thread.done
}

// inputTarget is where one device's callbacks deliver: its queue, and what
// removal does.
type inputTarget struct {
	queue *reportQueue
	// removed marks the handle gone and stops the device's run loop.
	removed func()
}

// inputRegistry finds a device from the context pointer IOKit hands its
// callbacks. purego callbacks are never freed and a process may make only
// 2000, so the driver makes each callback once and passes every device a
// number, never a Go pointer, as the context.
type inputRegistry struct {
	mutex   sync.Mutex
	next    uintptr
	targets map[uintptr]inputTarget
}

func newInputRegistry() *inputRegistry {
	return &inputRegistry{mutex: sync.Mutex{}, next: 0, targets: map[uintptr]inputTarget{}}
}

// add registers a device and returns its context number, never zero.
func (registry *inputRegistry) add(target inputTarget) uintptr {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	registry.next++
	registry.targets[registry.next] = target

	return registry.next
}

// remove forgets a device; callbacks for it are ignored from then on.
func (registry *inputRegistry) remove(context uintptr) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	delete(registry.targets, context)
}

func (registry *inputRegistry) target(context uintptr) (inputTarget, bool) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	target, known := registry.targets[context]

	return target, known
}

// deliverReport queues a copy of one input report for the device, as
// hidapi's report callback does whatever the result IOKit gives.
func (registry *inputRegistry) deliverReport(context uintptr, report []byte) {
	if target, known := registry.target(context); known && target.queue != nil {
		target.queue.add(report)
	}
}

// deliverRemoval marks the device gone and stops its run loop, as hidapi's
// removal callback does.
func (registry *inputRegistry) deliverRemoval(context uintptr) {
	if target, known := registry.target(context); known && target.removed != nil {
		target.removed()
	}
}
