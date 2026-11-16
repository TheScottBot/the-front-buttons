package hidtransport //nolint:testpackage // the queue is unexported, used only by the macOS driver

import (
	"bytes"
	"sync"
	"testing"
)

const (
	queueTestOverflow = 10
	queueTestAdders   = 8
	queueTestEach     = 200
	queueTestBuffer   = 16
)

func TestReportQueue_GivesNothingWhenEmpty(t *testing.T) {
	t.Parallel()

	if taken := newReportQueue().take(make([]byte, queueTestBuffer)); taken != 0 {
		t.Errorf("want 0, got %d", taken)
	}
}

func TestReportQueue_GivesReportsOldestFirst(t *testing.T) {
	t.Parallel()

	queue := newReportQueue()
	queue.add([]byte{1, 1})
	queue.add([]byte{2, 2, 2})

	buffer := make([]byte, queueTestBuffer)

	for _, want := range [][]byte{{1, 1}, {2, 2, 2}} {
		taken := queue.take(buffer)
		if !bytes.Equal(buffer[:taken], want) {
			t.Errorf("want %v, got %v", want, buffer[:taken])
		}
	}

	if queue.length() != 0 {
		t.Errorf("want the queue empty, got %d", queue.length())
	}
}

// TestReportQueue_KeepsACopy matters because IOKit reuses one buffer for every
// input report it hands the callback.
func TestReportQueue_KeepsACopy(t *testing.T) {
	t.Parallel()

	queue := newReportQueue()
	reused := []byte{1, 2, 3}
	queue.add(reused)
	reused[0] = 9

	buffer := make([]byte, queueTestBuffer)

	taken := queue.take(buffer)
	if !bytes.Equal(buffer[:taken], []byte{1, 2, 3}) {
		t.Errorf("want the report as added, got %v", buffer[:taken])
	}
}

func TestReportQueue_DropsTheOldestPastItsLimit(t *testing.T) {
	t.Parallel()

	queue := newReportQueue()

	for index := range reportQueueLimit + queueTestOverflow {
		queue.add([]byte{byte(index)})
	}

	if queue.length() != reportQueueLimit {
		t.Fatalf("want %d reports kept, got %d", reportQueueLimit, queue.length())
	}

	buffer := make([]byte, 1)

	for index := queueTestOverflow; index < reportQueueLimit+queueTestOverflow; index++ {
		if taken := queue.take(buffer); taken != 1 || buffer[0] != byte(index) {
			t.Fatalf("want report %d, got %v", index, buffer[:taken])
		}
	}
}

func TestReportQueue_LimitIsHIDAPIs(t *testing.T) {
	t.Parallel()

	const hidapiLimit = 30

	if reportQueueLimit != hidapiLimit {
		t.Errorf("want %d, got %d", hidapiLimit, reportQueueLimit)
	}
}

func TestReportQueue_CutsALongReportToTheBuffer(t *testing.T) {
	t.Parallel()

	queue := newReportQueue()
	queue.add([]byte{1, 2, 3, 4})

	buffer := make([]byte, 2)

	if taken := queue.take(buffer); taken != 2 || !bytes.Equal(buffer, []byte{1, 2}) {
		t.Errorf("want [1 2], got %v", buffer[:taken])
	}

	if queue.length() != 0 {
		t.Errorf("want the report gone once taken, got %d left", queue.length())
	}
}

// TestReportQueue_IsSafeAcrossGoroutines runs adders against a reader, as
// the run loop thread runs against the watcher; the race detector checks it.
func TestReportQueue_IsSafeAcrossGoroutines(t *testing.T) {
	t.Parallel()

	queue := newReportQueue()

	var adders sync.WaitGroup

	for range queueTestAdders {
		adders.Go(func() {
			for range queueTestEach {
				queue.add([]byte{1})
			}
		})
	}

	taken := 0
	finished := make(chan struct{})

	go func() {
		adders.Wait()
		close(finished)
	}()

	buffer := make([]byte, 1)

	for {
		taken += queue.take(buffer)

		select {
		case <-finished:
			taken += drain(queue, buffer)

			if taken == 0 || taken > queueTestAdders*queueTestEach || queue.length() != 0 {
				t.Errorf("took %d of %d reports, %d left", taken, queueTestAdders*queueTestEach, queue.length())
			}

			return
		default:
		}
	}
}

func drain(queue *reportQueue, buffer []byte) int {
	total := 0

	for {
		taken := queue.take(buffer)
		if taken == 0 {
			return total
		}

		total += taken
	}
}
