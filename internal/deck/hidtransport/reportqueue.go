package hidtransport

import "sync"

// reportQueueLimit is how many input reports a queue keeps. As hidapi's macOS
// backend does, a device that nobody reads from does not grow its queue for
// ever: past this many, each new report pushes out the oldest.
const reportQueueLimit = 30

// reportQueue holds the input reports a device has sent and nobody has read
// yet, oldest first. The run loop thread adds to it and the watcher's reads
// take from it, so every method is safe from several goroutines at once.
type reportQueue struct {
	mutex   sync.Mutex
	reports [][]byte
}

func newReportQueue() *reportQueue {
	return &reportQueue{mutex: sync.Mutex{}, reports: nil}
}

// add keeps a copy of report, dropping the oldest report when the queue is
// already full. The caller may reuse report as soon as add returns.
func (queue *reportQueue) add(report []byte) {
	kept := append([]byte(nil), report...)

	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	if len(queue.reports) >= reportQueueLimit {
		queue.reports[0] = nil
		queue.reports = queue.reports[1:]
	}

	queue.reports = append(queue.reports, kept)
}

// take copies the oldest report into buffer and removes it, returning the
// bytes copied. A report longer than buffer is cut short, as hidapi does. It
// returns 0 when no report waits.
func (queue *reportQueue) take(buffer []byte) int {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	if len(queue.reports) == 0 {
		return 0
	}

	oldest := queue.reports[0]
	queue.reports[0] = nil
	queue.reports = queue.reports[1:]

	return copy(buffer, oldest)
}

// length is how many reports wait.
func (queue *reportQueue) length() int {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	return len(queue.reports)
}
