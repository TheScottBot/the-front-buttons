package diagnostics_test

import (
	"sync"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
)

func TestCounters_IncrementAndSet_AreSafeFromManyGoroutines(t *testing.T) {
	t.Parallel()

	counters := diagnostics.NewCounters()

	var waitGroup sync.WaitGroup

	for range 50 {
		waitGroup.Go(func() { counters.Increment("key_presses.open_url") })
	}

	waitGroup.Wait()
	counters.Set("github.rate_limit_remaining", 4990)

	snapshot := counters.Snapshot()
	if snapshot["key_presses.open_url"] != 50 || snapshot["github.rate_limit_remaining"] != 4990 {
		t.Fatalf("want 50 presses and 4990 remaining, got %v", snapshot)
	}
}

func TestCounters_SnapshotIsACopy(t *testing.T) {
	t.Parallel()

	counters := diagnostics.NewCounters()
	counters.Increment("deck.connects")

	snapshot := counters.Snapshot()
	snapshot["deck.connects"] = 99

	if counters.Snapshot()["deck.connects"] != 1 {
		t.Fatal("want the snapshot detached from the counters")
	}
}
