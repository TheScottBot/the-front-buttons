package systemadapter_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// stopWait bounds each wait for a context to end; the event is set in the same
// process, so it ends at once or not at all.
const stopWait = 5 * time.Second

// testStopRequest is a stop request on an event named for this test alone.
func testStopRequest(t *testing.T) systemadapter.StopRequest {
	t.Helper()

	return systemadapter.NewStopRequestNamed(`Local\the-front-buttons-test.` +
		strings.ReplaceAll(t.Name(), "/", ".") + "." + time.Now().Format("150405.000000000"))
}

func requireEnds(stopped context.Context, t *testing.T) {
	t.Helper()

	select {
	case <-stopped.Done():
	case <-time.After(stopWait):
		t.Fatal("want the context ended, still running")
	}
}

func TestStopRequest_ReportsNotRunning_WhenNothingListens(t *testing.T) {
	t.Parallel()

	requestError := testStopRequest(t).Request()

	if !errors.Is(requestError, systemadapter.ErrNotRunning) {
		t.Fatalf("want ErrNotRunning, got %v", requestError)
	}
}

func TestStopRequest_EndsTheListeningContext_WithStopRequestedAsTheCause(t *testing.T) {
	t.Parallel()

	request := testStopRequest(t)

	stopped, release, notifyError := request.Notify(t.Context())
	if notifyError != nil {
		t.Fatalf("notify: %v", notifyError)
	}
	defer release()

	requestError := request.Request()
	if requestError != nil {
		t.Fatalf("request: %v", requestError)
	}

	requireEnds(stopped, t)

	cause := context.Cause(stopped)
	if !errors.Is(cause, systemadapter.ErrStopRequested) {
		t.Fatalf("want ErrStopRequested as the cause, got %v", cause)
	}
}

func TestStopRequest_StopsListening_WhenReleased(t *testing.T) {
	t.Parallel()

	request := testStopRequest(t)

	stopped, release, notifyError := request.Notify(t.Context())
	if notifyError != nil {
		t.Fatalf("notify: %v", notifyError)
	}

	release()
	requireEnds(stopped, t)

	cause := context.Cause(stopped)
	if errors.Is(cause, systemadapter.ErrStopRequested) {
		t.Fatal("want a release, not a stop request, as the cause")
	}

	requestError := request.Request()
	if !errors.Is(requestError, systemadapter.ErrNotRunning) {
		t.Fatalf("want ErrNotRunning once released, got %v", requestError)
	}
}

func TestStopRequest_EndsWithItsParent(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithCancel(t.Context())

	stopped, release, notifyError := testStopRequest(t).Notify(parent)
	if notifyError != nil {
		t.Fatalf("notify: %v", notifyError)
	}

	cancelParent()
	requireEnds(stopped, t)

	released := make(chan struct{})

	go func() {
		release()
		close(released)
	}()

	select {
	case <-released:
	case <-time.After(stopWait):
		t.Fatal("want release to return once the parent ended, still waiting")
	}
}
