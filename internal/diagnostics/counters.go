package diagnostics

import (
	"maps"
	"sync"
)

// Counters are the named counts and readings the log and the editor's
// diagnostics view both read (specification section 7): key presses by kind,
// brick errors by reason, pipeline polls by result, deck connections, rejected
// configurations and editor requests. Names carry their category as a dotted
// prefix; values never carry a payload, only a number.
type Counters struct {
	mutex  sync.Mutex
	values map[string]int64
}

// NewCounters returns empty counters.
func NewCounters() *Counters {
	return &Counters{mutex: sync.Mutex{}, values: map[string]int64{}}
}

// Increment adds one to a count.
func (counters *Counters) Increment(name string) {
	counters.mutex.Lock()
	defer counters.mutex.Unlock()

	counters.values[name]++
}

// Set records a reading, such as the remaining rate limit.
func (counters *Counters) Set(name string, value int64) {
	counters.mutex.Lock()
	defer counters.mutex.Unlock()

	counters.values[name] = value
}

// Snapshot returns a copy of every count and reading.
func (counters *Counters) Snapshot() map[string]int64 {
	counters.mutex.Lock()
	defer counters.mutex.Unlock()

	return maps.Clone(counters.values)
}
