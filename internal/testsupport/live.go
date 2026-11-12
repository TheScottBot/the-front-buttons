package testsupport

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// LiveNoRunReason is the fake live kind's press failure when no run is known,
// matching the real pipeline key's.
const LiveNoRunReason = "no_run"

var errLiveNoRun = errors.New("fake live key: no run to open yet")

// LiveKindName is a fake live kind whose states a test sets directly.
const LiveKindName = "test_live"

// LiveParameters names the fake source a key watches, and a credential so the
// daemon's missing credential check has a secret field to read.
type LiveParameters struct {
	brick.Appearance

	Source     string `brick:"source,required"   bounds:"1,32" description:"Name of the fake source the key watches."`
	Credential string `brick:"credential,secret" bounds:"1,64" format:"identifier"                                    description:"Name of a credential the fake source would use."`
}

// LiveController sets fake sources' states and records presses on live keys.
type LiveController struct {
	mutex    sync.Mutex
	states   map[string]brick.LiveState
	changed  chan struct{}
	pressed  []string
	watching map[string]int
}

// NewLiveController returns a controller with no states set.
func NewLiveController() *LiveController {
	return &LiveController{
		mutex:    sync.Mutex{},
		states:   map[string]brick.LiveState{},
		changed:  make(chan struct{}),
		pressed:  nil,
		watching: map[string]int{},
	}
}

// Set publishes a state for a source to every key watching it.
func (controller *LiveController) Set(source string, state brick.LiveState) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()

	controller.states[source] = state
	close(controller.changed)
	controller.changed = make(chan struct{})
}

// Pressed returns the targets live keys were pressed with.
func (controller *LiveController) Pressed() []string {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()

	return append([]string(nil), controller.pressed...)
}

// Watching counts the running watches of a source.
func (controller *LiveController) Watching(source string) int {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()

	return controller.watching[source]
}

func (controller *LiveController) current(source string) (brick.LiveState, bool, chan struct{}) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()

	state, set := controller.states[source]

	return state, set, controller.changed
}

// liveTestDouble is what the fake live kind constructs.
type liveTestDouble struct {
	source     string
	controller *LiveController
}

func (watcher liveTestDouble) LiveIdentity() string { return LiveKindName + ":" + watcher.source }

func (watcher liveTestDouble) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	watcher.controller.mutex.Lock()
	watcher.controller.watching[watcher.source]++
	watcher.controller.mutex.Unlock()

	defer func() {
		watcher.controller.mutex.Lock()
		watcher.controller.watching[watcher.source]--
		watcher.controller.mutex.Unlock()
	}()

	for {
		state, set, changed := watcher.controller.current(watcher.source)
		if set {
			publish(state)
		}

		select {
		case <-watchContext.Done():
			return
		case <-changed:
		}
	}
}

func (watcher liveTestDouble) PressWithState(_ context.Context, state brick.LiveState) error {
	watcher.controller.mutex.Lock()
	defer watcher.controller.mutex.Unlock()

	// Like the real pipeline key, a press with no run known yet fails.
	if state.TargetURL == "" {
		return fmt.Errorf("pressing the fake live key: %w", brick.Categorised(LiveNoRunReason, errLiveNoRun))
	}

	watcher.controller.pressed = append(watcher.controller.pressed, state.TargetURL)

	return nil
}

// RegisterLiveKind adds the fake live kind to a registry.
func RegisterLiveKind(registry *brick.Registry, controller *LiveController) error {
	registerError := brick.Register(registry, LiveKindName, func(parameters LiveParameters, _ brick.Ports) (any, error) {
		return liveTestDouble{source: parameters.Source, controller: controller}, nil
	})
	if registerError != nil {
		return fmt.Errorf("registering the fake live kind: %w", registerError)
	}

	return nil
}
