//go:build cgo

package hidtransport //nolint:testpackage // drives the unexported hidapi driver and the run loop functions

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// firstUseVariable makes the test binary run the body of
// TestHIDAPI_FirstUse_LeavesTheCallersRunLoopEmpty in a process of its own,
// where it is hidapi's first use: hidapi initialises once per process.
const firstUseVariable = "THE_FRONT_BUTTONS_HIDAPI_FIRST_USE"

// TestHIDAPI_FirstUse_LeavesTheCallersRunLoopEmpty wants hidapi's first use
// to schedule nothing on the run loop of the calling thread. On macOS
// hid_init schedules hidapi's one HID manager on the run loop of whichever
// thread calls it first. A Go thread is shared by goroutines, so another
// goroutine later locked to that thread, such as an IOKit input thread,
// would run hidapi's manager on its run loop while another thread
// enumerates: that crashed the test process inside hid_enumerate.
func TestHIDAPI_FirstUse_LeavesTheCallersRunLoopEmpty(t *testing.T) {
	t.Parallel()

	if os.Getenv(firstUseVariable) == "1" {
		checkFirstUseLeavesRunLoopEmpty(t)

		return
	}

	child := exec.CommandContext(t.Context(), os.Args[0], //nolint:gosec // reruns this test binary
		"-test.run=^TestHIDAPI_FirstUse_LeavesTheCallersRunLoopEmpty$", "-test.count=1", "-test.v")

	child.Env = append(os.Environ(), firstUseVariable+"=1")

	output, runError := child.CombinedOutput()
	if runError != nil {
		t.Fatalf("the first use in a process of its own failed: %v\n%s", runError, output)
	}
}

// checkFirstUseLeavesRunLoopEmpty checks on a thread other than the main
// thread, whose run loop always holds the main dispatch queue's port.
// runLoopRunFinished, returned for a run of no time at all, means the run loop
// holds no source in the mode: nothing is scheduled on it.
func checkFirstUseLeavesRunLoopEmpty(t *testing.T) {
	t.Helper()

	functions, loadError := loadIOKit()
	if loadError != nil {
		t.Fatal(loadError)
	}

	coreFoundation, frameworkError := objcbridge.Framework(objcbridge.PathCoreFoundation)
	if frameworkError != nil {
		t.Fatal(frameworkError)
	}

	var mainRunLoop func() uintptr

	objcbridge.Function(&mainRunLoop, coreFoundation, "CFRunLoopGetMain")

	runOnce := func() int32 { return functions.runLoopRunInMode(functions.defaultMode, 0, true) }

	onOtherThread(t, func() bool { return functions.runLoopGetCurrent() != mainRunLoop() }, func() {
		if before := runOnce(); before != runLoopRunFinished {
			t.Errorf("want this thread's run loop empty before hidapi is used, got result %d", before)

			return
		}

		listError := withDriverLock(func() error {
			_, enumerateError := hidapiDriver{}.enumerate(elgatoVendorID)

			return enumerateError
		})
		if listError != nil {
			t.Errorf("want enumeration to succeed, got %v", listError)

			return
		}

		if after := runOnce(); after != runLoopRunFinished {
			t.Errorf("want hidapi to schedule nothing on the calling thread's run loop, got result %d", after)
		}
	})
}

// onOtherThread runs check on a goroutine locked to a thread for which
// suitable reports true. A goroutine landing on an unsuitable thread stays
// locked to it until check is over, so the next goroutine cannot land there.
func onOtherThread(t *testing.T, suitable func() bool, check func()) {
	t.Helper()

	release := make(chan struct{})
	defer close(release)

	for {
		landed := make(chan bool)
		done := make(chan struct{})

		go func() {
			runtime.LockOSThread()

			defer runtime.UnlockOSThread()

			if !suitable() {
				landed <- false

				<-release

				return
			}

			landed <- true

			defer close(done)

			check()
		}()

		if <-landed {
			<-done

			return
		}
	}
}

// TestHIDAPIThread_RunsEveryCallOnOneThread wants every call on hidapi's
// thread run on the same run loop, and so the same thread, whichever thread
// sends it. Each call is sent from a goroutine locked to a thread of its own,
// which then waits, so a goroutine not locked to its thread would be taken
// up by another thread between calls.
func TestHIDAPIThread_RunsEveryCallOnOneThread(t *testing.T) {
	t.Parallel()

	functions, loadError := loadIOKit()
	if loadError != nil {
		t.Fatal(loadError)
	}

	const callCount = 64

	runLoops := make(chan uintptr, callCount)

	for range callCount {
		sent := make(chan struct{})

		go func() {
			runtime.LockOSThread()

			defer runtime.UnlockOSThread()
			defer close(sent)

			onHIDAPIThread(func() { runLoops <- functions.runLoopGetCurrent() })
		}()

		<-sent
	}

	close(runLoops)

	first := <-runLoops

	for runLoop := range runLoops {
		if runLoop != first {
			t.Fatalf("want every call on one run loop, got %#x and %#x", first, runLoop)
		}
	}
}
