package systemadapter_test

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// The interval the watcher asks at in these tests.
const askEvery = 2 * time.Second

// answers is a stand in for asking a fresh copy of the daemon: it says no
// until the given call, then yes, and counts the calls.
type answers struct {
	yesFrom int32
	calls   atomic.Int32
}

func (stand *answers) check(context.Context) bool {
	return stand.calls.Add(1) >= stand.yesFrom
}

// TestAccessibilityWatcher_SaysOnceWhenPermissionArrives: a running daemon
// does not see Accessibility granted until it restarts, which the author had
// to do by hand (evaluation, deck testing). The watcher asks a fresh copy
// every interval and says so once when the answer turns to yes; starting it
// again while it runs starts nothing more.
func TestAccessibilityWatcher_SaysOnceWhenPermissionArrives(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		stand := &answers{yesFrom: 3, calls: atomic.Int32{}}

		var told atomic.Int32

		watcher := systemadapter.NewAccessibilityWatcher(t.Context(), stand.check, askEvery, func() { told.Add(1) })
		watcher.Start()
		watcher.Start()

		time.Sleep(10 * askEvery)
		synctest.Wait()
		watcher.Start()
		synctest.Wait()

		if told.Load() != 1 || stand.calls.Load() != 3 {
			t.Fatalf("want one yes after three asks, got %d told after %d asks", told.Load(), stand.calls.Load())
		}
	})
}

func TestAccessibilityWatcher_StopsWithTheDaemon(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		stand := &answers{yesFrom: 100, calls: atomic.Int32{}}

		var told atomic.Int32

		watchContext, stop := context.WithCancel(t.Context())
		watcher := systemadapter.NewAccessibilityWatcher(watchContext, stand.check, askEvery, func() { told.Add(1) })
		watcher.Start()

		time.Sleep(3 * askEvery)
		stop()
		synctest.Wait()

		asked := stand.calls.Load()

		time.Sleep(10 * askEvery)
		synctest.Wait()

		if told.Load() != 0 || stand.calls.Load() != asked {
			t.Fatalf("want no asking once stopped, got %d more asks", stand.calls.Load()-asked)
		}
	})
}
