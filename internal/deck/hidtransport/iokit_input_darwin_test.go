package hidtransport //nolint:testpackage // drives the driver's run loop and callbacks, which are unexported

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// The macOS driver's run loop and callbacks, on the Mac, with no device: a
// real CFRunLoop on its own thread, and the callbacks called through purego
// as IOKit calls them. Scheduling a device and its reports arriving are
// checked on the deck by the driver contract and the experiments.

// bareRunLoop is a device's run loop with no device on it.
type bareRunLoop struct {
	cfRunLoop
}

func (loop *bareRunLoop) interrupt() { loop.stopLoop() }

func newBareRunLoop(t *testing.T) *bareRunLoop {
	t.Helper()

	functions, loadError := loadIOKit()
	if loadError != nil {
		t.Fatal(loadError)
	}

	return &bareRunLoop{functions: functions, runLoop: 0, source: 0}
}

func TestIOKitRunLoop_RunsOnItsThreadUntilClosed(t *testing.T) {
	t.Parallel()

	loop := newBareRunLoop(t)
	gone := atomic.Int32{}
	thread := startInputThread(loop, func() { gone.Add(1) })

	if loop.runLoop == 0 || loop.source == 0 {
		t.Fatalf("want a run loop and source once started, got %#x and %#x", loop.runLoop, loop.source)
	}

	time.Sleep(50 * time.Millisecond)

	if isClosed(thread.done) || gone.Load() != 0 {
		t.Fatal("want the run loop kept running by its source, not finished or taken for removal")
	}

	started := time.Now()

	thread.stop()

	if !isClosed(thread.done) || gone.Load() != 0 {
		t.Errorf("want the thread ended by close, not taken for removal, got ended %v gone %d",
			isClosed(thread.done), gone.Load())
	}

	if waited := time.Since(started); waited > runLoopSlice*time.Second/2 {
		t.Errorf("want CFRunLoopStop to end the loop at once, waited %v", waited)
	}
}

func TestIOKitRunLoop_StoppedAloneMarksTheDeviceGone(t *testing.T) {
	t.Parallel()

	loop := newBareRunLoop(t)
	gone := make(chan struct{})
	thread := startInputThread(loop, func() { close(gone) })

	// The removal callback stops the run loop like this, on its thread;
	// CFRunLoopStop may be called from any thread.
	loop.stopLoop()

	select {
	case <-gone:
	case <-time.After(threadWait):
		t.Fatal("want the device marked gone")
	}

	thread.stop()

	if !isClosed(thread.done) {
		t.Error("want the thread ended once closed")
	}
}

func TestIOKitCallbacks_AreMadeOncePerProcess(t *testing.T) {
	t.Parallel()

	first, firstError := loadIOKit()
	second, secondError := loadIOKit()

	if firstError != nil || secondError != nil {
		t.Fatalf("loading: %v, %v", firstError, secondError)
	}

	if first.inputCallback == 0 || first.removalCallback == 0 || first.inputCallback == first.removalCallback {
		t.Errorf("want two distinct callbacks, got %#x and %#x", first.inputCallback, first.removalCallback)
	}

	if second.inputCallback != first.inputCallback || second.removalCallback != first.removalCallback {
		t.Error("want the same callbacks on every load")
	}
}

// TestIOKitCallbacks_DeliverThroughTheCABI calls each callback through purego
// with the arguments IOKit passes, so a wrong argument list shows here rather
// than on the deck.
func TestIOKitCallbacks_DeliverThroughTheCABI(t *testing.T) {
	t.Parallel()

	functions, loadError := loadIOKit()
	if loadError != nil {
		t.Fatal(loadError)
	}

	queue := newReportQueue()
	removals := atomic.Int32{}
	context := inputs.add(inputTarget{queue: queue, removed: func() { removals.Add(1) }})

	defer inputs.remove(context)

	report := []byte{0x01, 0x00, 0x20, 0x00, 0x01}
	sender := uintptr(0x1234)

	purego.SyscallN(functions.inputCallback, context, ioReturnSuccess, sender, uintptr(reportTypeInput), 0,
		uintptr(unsafe.Pointer(&report[0])), uintptr(len(report))) //nolint:gosec // the report's address, as IOKit passes it
	purego.SyscallN(functions.removalCallback, context, ioReturnSuccess, sender)

	buffer := make([]byte, queueTestBuffer)

	taken := queue.take(buffer)
	if !bytes.Equal(buffer[:taken], report) {
		t.Errorf("want the report queued, got %v", buffer[:taken])
	}

	if removals.Load() != 1 {
		t.Errorf("want one removal, got %d", removals.Load())
	}
}
