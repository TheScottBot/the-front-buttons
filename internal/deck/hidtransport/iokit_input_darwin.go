package hidtransport

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// Input reports and removal for the macOS driver, as hidapi's macOS backend
// does them: each open device is scheduled on a CFRunLoop of its own, run by
// a goroutine locked to its thread (inputThread in iokitcore.go), with
// IOHIDDeviceRegisterInputReportCallback filling the device's queue and
// IOHIDDeviceRegisterRemovalCallback marking it gone.

// runLoopSlice is how long one CFRunLoopRunInMode call runs, in seconds,
// before the thread looks again at whether it is asked to stop. close stops
// the run loop at once with CFRunLoopStop; this bounds the wait should that
// ever be missed.
const runLoopSlice = 1.0

// inputs finds the device each callback is for.
//
//nolint:gochecknoglobals // purego callbacks are process wide, so what they reach is too
var inputs = newInputRegistry()

// runLoopSourceContext is CFRunLoopSourceContext, version 0, from
// CFRunLoop.h: a version, an info pointer and eight callbacks, ten words in
// all, every one zero here. The driver's source has no callbacks, as it is
// never signalled: it only keeps the run loop from finishing for want of
// sources.
type runLoopSourceContext struct {
	_ [10]uintptr
}

// inputFunctions are the functions and values input reports and removal
// need, found with the rest in loadIOKit.
type inputFunctions struct {
	deviceSchedule      func(device cfRef, runLoop uintptr, mode cfRef)
	deviceUnschedule    func(device cfRef, runLoop uintptr, mode cfRef)
	registerInput       func(device cfRef, report *byte, length int, callback, context uintptr)
	registerRemoval     func(device cfRef, callback, context uintptr)
	runLoopGetCurrent   func() uintptr
	runLoopRunInMode    func(mode cfRef, seconds float64, returnAfterSourceHandled bool) int32
	runLoopStop         func(runLoop uintptr)
	runLoopWakeUp       func(runLoop uintptr)
	runLoopSourceCreate func(allocator uintptr, order int, context *runLoopSourceContext) cfRef
	runLoopAddSource    func(runLoop uintptr, source, mode cfRef)
	runLoopRemoveSource func(runLoop uintptr, source, mode cfRef)

	// defaultMode is kCFRunLoopDefaultMode. Each device has a thread and
	// run loop of its own, so it needs no mode of its own, as hidapi's
	// shared threads did.
	defaultMode cfRef
	// inputCallback and removalCallback are made once per process.
	inputCallback   uintptr
	removalCallback uintptr
}

func (functions *inputFunctions) inputBindings(iokit, coreFoundation uintptr) []binding {
	return []binding{
		{&functions.deviceSchedule, iokit, "IOHIDDeviceScheduleWithRunLoop"},
		{&functions.deviceUnschedule, iokit, "IOHIDDeviceUnscheduleFromRunLoop"},
		{&functions.registerInput, iokit, "IOHIDDeviceRegisterInputReportCallback"},
		{&functions.registerRemoval, iokit, "IOHIDDeviceRegisterRemovalCallback"},
		{&functions.runLoopGetCurrent, coreFoundation, "CFRunLoopGetCurrent"},
		{&functions.runLoopRunInMode, coreFoundation, "CFRunLoopRunInMode"},
		{&functions.runLoopStop, coreFoundation, "CFRunLoopStop"},
		{&functions.runLoopWakeUp, coreFoundation, "CFRunLoopWakeUp"},
		{&functions.runLoopSourceCreate, coreFoundation, "CFRunLoopSourceCreate"},
		{&functions.runLoopAddSource, coreFoundation, "CFRunLoopAddSource"},
		{&functions.runLoopRemoveSource, coreFoundation, "CFRunLoopRemoveSource"},
	}
}

// findInputValues reads kCFRunLoopDefaultMode and makes the two callbacks.
// It runs once, inside loadIOKit, so purego's limit of 2000 callbacks is
// never approached and a failure is recovered there.
func (functions *inputFunctions) findInputValues(coreFoundation uintptr) error {
	mode, modeError := objcbridge.Constant(coreFoundation, "kCFRunLoopDefaultMode")
	if modeError != nil {
		return fmt.Errorf("reading the run loop mode: %w", modeError)
	}

	functions.defaultMode = mode
	functions.inputCallback = purego.NewCallback(inputReportReceived)
	functions.removalCallback = purego.NewCallback(deviceRemoved)

	return nil
}

// inputReportReceived is IOHIDReportCallback: it queues the report for the
// device the context names. IOKit reuses its buffer, so the queue keeps a
// copy.
func inputReportReceived(context uintptr, _ int32, _ uintptr, _ int32, _ uint32, report *byte, length int) {
	if report == nil || length <= 0 {
		return
	}

	inputs.deliverReport(context, unsafe.Slice(report, length)) //nolint:gosec // IOKit's buffer, length bytes long
}

// deviceRemoved is IOHIDCallback for removal: it marks the device the
// context names gone and stops its run loop.
func deviceRemoved(context uintptr, _ int32, _ uintptr) {
	inputs.deliverRemoval(context)
}

// cfRunLoop is the part of a device's run loop that needs no device: the
// thread's run loop and a source that keeps it running. The driver's
// deviceRunLoop adds the device to it.
type cfRunLoop struct {
	functions *iokitFunctions
	runLoop   uintptr
	source    cfRef
}

func (loop *cfRunLoop) prepare() {
	functions := loop.functions
	loop.runLoop = functions.runLoopGetCurrent()
	loop.source = functions.runLoopSourceCreate(allocatorDefault, 0, new(runLoopSourceContext))
	functions.runLoopAddSource(loop.runLoop, loop.source, functions.defaultMode)
}

func (loop *cfRunLoop) run() int32 {
	return loop.functions.runLoopRunInMode(loop.functions.defaultMode, runLoopSlice, false)
}

// stopLoop stops the run loop from any thread.
func (loop *cfRunLoop) stopLoop() {
	loop.functions.runLoopStop(loop.runLoop)
	loop.functions.runLoopWakeUp(loop.runLoop)
}

func (loop *cfRunLoop) finish() {
	loop.functions.runLoopRemoveSource(loop.runLoop, loop.source, loop.functions.defaultMode)
	objcbridge.Release(loop.source)
}

// deviceRunLoop is one open device's run loop: the device scheduled on it,
// its input report buffer and its callbacks.
type deviceRunLoop struct {
	cfRunLoop

	device  cfRef
	context uintptr
	// buffer is where IOKit writes each input report, MaxInputReportSize
	// long, pinned so it stays put while IOKit holds it.
	buffer []byte
	pinner runtime.Pinner
}

func (loop *deviceRunLoop) prepare() {
	loop.cfRunLoop.prepare()

	functions := loop.functions

	if len(loop.buffer) > 0 {
		loop.pinner.Pin(&loop.buffer[0])
		functions.registerInput(loop.device, &loop.buffer[0], len(loop.buffer), functions.inputCallback, loop.context)
	}

	functions.registerRemoval(loop.device, functions.removalCallback, loop.context)
	functions.deviceSchedule(loop.device, loop.runLoop, functions.defaultMode)
}

// interrupt unregisters the callbacks with null ones and unschedules the
// device, as hidapi's hid_close does, then stops the run loop.
func (loop *deviceRunLoop) interrupt() {
	functions := loop.functions

	if len(loop.buffer) > 0 {
		functions.registerInput(loop.device, &loop.buffer[0], len(loop.buffer), 0, loop.context)
	}

	functions.registerRemoval(loop.device, 0, loop.context)
	functions.deviceUnschedule(loop.device, loop.runLoop, functions.defaultMode)
	loop.stopLoop()
}

func (loop *deviceRunLoop) finish() {
	loop.cfRunLoop.finish()
	loop.pinner.Unpin()
}

// startDevice gives an opened device its queue and run loop thread and
// returns its handle.
func (functions *iokitFunctions) startDevice(device cfRef, lengths reportLengths) *iokitHandle {
	opened := &iokitDevice{functions: functions, device: device, handle: nil, context: 0, input: nil}
	queue := newReportQueue()
	opened.handle = newIOKitHandle(opened, lengths, queue)

	loop := &deviceRunLoop{
		functions: functions,
		runLoop:   0,
		source:    0,
		device:    device,
		context:   0,
		buffer:    make([]byte, max(lengths.input, 0)),
		pinner:    runtime.Pinner{},
	}

	loop.context = inputs.add(inputTarget{queue: queue, removed: func() {
		opened.handle.markGone()
		loop.stopLoop()
	}})
	opened.context = loop.context
	opened.input = startInputThread(loop, opened.handle.markGone)

	return opened.handle
}

// stopInput ends the device's run loop thread and forgets its callbacks'
// context, before the device is closed.
func (opened *iokitDevice) stopInput() {
	if opened.input != nil {
		opened.input.stop()
		opened.input = nil
	}

	inputs.remove(opened.context)
}
