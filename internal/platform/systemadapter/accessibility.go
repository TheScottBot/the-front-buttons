package systemadapter

import (
	"context"
	"sync"
	"time"
)

// AccessibilityCheckFlag makes the daemon's binary answer one question and
// exit: zero if macOS trusts it for Accessibility, one if not. A running
// daemon does not see the permission granted until it restarts (evaluation,
// deck testing), but a fresh process does, so the daemon asks one.
const AccessibilityCheckFlag = "accessibility-check"

// AccessibilityWatcher waits for Accessibility to be granted after a key press
// was refused, asking a fresh process every interval, and says so once, so the
// daemon can restart itself to use it rather than the author restarting it by
// hand (the author's request, evaluation, deck testing).
type AccessibilityWatcher struct {
	watchContext context.Context //nolint:containedctx // the watcher outlives the press that starts it
	check        func(checkContext context.Context) bool
	interval     time.Duration
	granted      func()
	once         *sync.Once
}

// NewAccessibilityWatcher returns a watcher that asks check every interval
// until it says yes or watchContext ends, then calls granted once.
func NewAccessibilityWatcher(
	watchContext context.Context,
	check func(checkContext context.Context) bool,
	interval time.Duration,
	granted func(),
) *AccessibilityWatcher {
	return &AccessibilityWatcher{
		watchContext: watchContext, check: check, interval: interval, granted: granted, once: &sync.Once{},
	}
}

// Start begins waiting, the first time it is called; later calls do nothing.
func (watcher *AccessibilityWatcher) Start() {
	watcher.once.Do(func() { go watcher.wait() })
}

func (watcher *AccessibilityWatcher) wait() {
	ticker := time.NewTicker(watcher.interval)
	defer ticker.Stop()

	for {
		select {
		case <-watcher.watchContext.Done():
			return
		case <-ticker.C:
			if watcher.check(watcher.watchContext) {
				watcher.granted()

				return
			}
		}
	}
}
