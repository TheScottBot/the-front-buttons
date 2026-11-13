package live_test

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/live"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// buildIdentity is the identity most hub tests watch.
const buildIdentity = "build"

// sourceTestDouble publishes a status once a second and counts how many times
// it was started and how many of its runs are still going.
type sourceTestDouble struct {
	identity string
	status   render.Status
	starts   *countTestDouble
}

type countTestDouble struct {
	mutex   sync.Mutex
	started map[string]int
	running map[string]int
}

func newCount() *countTestDouble {
	return &countTestDouble{mutex: sync.Mutex{}, started: map[string]int{}, running: map[string]int{}}
}

func (count *countTestDouble) change(identity string, startedDelta int, runningDelta int) {
	count.mutex.Lock()
	defer count.mutex.Unlock()

	count.started[identity] += startedDelta
	count.running[identity] += runningDelta
}

func (count *countTestDouble) read(identity string) (int, int) {
	count.mutex.Lock()
	defer count.mutex.Unlock()

	return count.started[identity], count.running[identity]
}

func (source sourceTestDouble) LiveIdentity() string { return source.identity }

func (source sourceTestDouble) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	source.starts.change(source.identity, 1, 1)
	defer source.starts.change(source.identity, 0, -1)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		publish(brick.LiveState{Status: source.status, ErrorReason: "", TargetURL: "", Picture: nil, Playing: false})

		select {
		case <-watchContext.Done():
			return
		case <-ticker.C:
		}
	}
}

func TestHub_RunsOneWatchPerIdentity_SoTwoKeysOnOneWorkflowMakeOnePoll(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		count := newCount()
		hub := live.NewHub(func(string) {})

		hub.Update(t.Context(), []brick.LiveSource{
			sourceTestDouble{identity: buildIdentity, status: render.StatusSuccess, starts: count},
			sourceTestDouble{identity: buildIdentity, status: render.StatusSuccess, starts: count},
		})
		time.Sleep(time.Second)
		synctest.Wait()

		if started, running := count.read(buildIdentity); started != 1 || running != 1 {
			t.Fatalf("want one watch for two keys, started %d running %d", started, running)
		}

		hub.Stop()
	})
}

func TestHub_KeepsTheLatestStatePerIdentity(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		count := newCount()
		hub := live.NewHub(func(string) {})

		hub.Update(t.Context(), []brick.LiveSource{
			sourceTestDouble{identity: buildIdentity, status: render.StatusFailure, starts: count},
		})
		time.Sleep(time.Second)
		synctest.Wait()

		state, found := hub.State(buildIdentity)
		if !found || state.Status != render.StatusFailure {
			t.Fatalf("want the failure cached, got %+v (%v)", state, found)
		}

		hub.Stop()
	})
}

func TestHub_RestartsOnlyChangedSources_AndStopsRemovedOnes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		count := newCount()
		hub := live.NewHub(func(string) {})

		hub.Update(t.Context(), []brick.LiveSource{
			sourceTestDouble{identity: "kept", status: render.StatusSuccess, starts: count},
			sourceTestDouble{identity: "removed", status: render.StatusSuccess, starts: count},
		})
		time.Sleep(time.Second)
		synctest.Wait()

		hub.Update(t.Context(), []brick.LiveSource{
			sourceTestDouble{identity: "kept", status: render.StatusSuccess, starts: count},
			sourceTestDouble{identity: "added", status: render.StatusQueued, starts: count},
		})
		time.Sleep(time.Second)
		synctest.Wait()

		keptStarts, keptRunning := count.read("kept")
		_, removedRunning := count.read("removed")
		addedStarts, addedRunning := count.read("added")

		if keptStarts != 1 || keptRunning != 1 {
			t.Fatalf("want the unchanged source left running, started %d running %d", keptStarts, keptRunning)
		}

		if removedRunning != 0 || addedStarts != 1 || addedRunning != 1 {
			t.Fatalf("want removed stopped and added started, removed running %d, added %d/%d",
				removedRunning, addedStarts, addedRunning)
		}

		if _, found := hub.State("removed"); found {
			t.Fatal("want a removed source's state forgotten")
		}

		hub.Stop()
	})
}

func TestHub_NotifiesEachPublishedState_AndStopWaitsForEveryWatch(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		count := newCount()

		var notified sync.Map

		hub := live.NewHub(func(identity string) { notified.Store(identity, true) })

		hub.Update(t.Context(), []brick.LiveSource{
			sourceTestDouble{identity: buildIdentity, status: render.StatusSuccess, starts: count},
		})
		time.Sleep(time.Second)
		synctest.Wait()

		if _, wasNotified := notified.Load(buildIdentity); !wasNotified {
			t.Fatal("want a notification for the published state")
		}

		hub.Stop()

		if _, running := count.read(buildIdentity); running != 0 {
			t.Fatalf("want every watch stopped once Stop returns, %d running", running)
		}
	})
}
