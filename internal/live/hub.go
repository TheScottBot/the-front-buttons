package live

import (
	"context"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// watch is one running source.
type watch struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Hub runs every live brick in the configuration, whatever page it is on and
// whether or not a deck is connected (specification section 2). Bricks with the
// same identity share one running source, so two keys on one workflow make one
// request per interval. The latest state of each source is cached, so a page
// switch shows it at once without waiting on any brick.
type Hub struct {
	notify func(identity string)

	mutex   sync.Mutex
	watches map[string]watch
	states  map[string]brick.LiveState
}

// NewHub returns a hub that calls notify, from the source's goroutine, each time
// a source publishes a state.
func NewHub(notify func(identity string)) *Hub {
	return &Hub{
		notify:  notify,
		mutex:   sync.Mutex{},
		watches: map[string]watch{},
		states:  map[string]brick.LiveState{},
	}
}

// Update makes the running sources match the given ones. A source whose identity
// is already running is left running with its state, which is how a reload
// restarts only the bricks whose definition changed: a changed definition is a
// new identity. Sources no longer present are stopped and their states dropped.
func (hub *Hub) Update(updateContext context.Context, sources []brick.LiveSource) {
	wanted := make(map[string]brick.LiveSource, len(sources))
	for _, source := range sources {
		if _, seen := wanted[source.LiveIdentity()]; !seen {
			wanted[source.LiveIdentity()] = source
		}
	}

	hub.mutex.Lock()

	var stopping []watch

	for identity, running := range hub.watches {
		if _, stillWanted := wanted[identity]; !stillWanted {
			running.cancel()
			stopping = append(stopping, running)

			delete(hub.watches, identity)
			delete(hub.states, identity)
		}
	}

	for identity, source := range wanted {
		if _, alreadyRunning := hub.watches[identity]; !alreadyRunning {
			hub.watches[identity] = hub.start(updateContext, identity, source)
		}
	}

	hub.mutex.Unlock()

	for _, stopped := range stopping {
		<-stopped.done
	}
}

// State returns the latest state a source published.
func (hub *Hub) State(identity string) (brick.LiveState, bool) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	state, found := hub.states[identity]

	return state, found
}

// Stop stops every source and waits for each to end.
func (hub *Hub) Stop() {
	hub.mutex.Lock()

	stopping := make([]watch, 0, len(hub.watches))
	for identity, running := range hub.watches {
		running.cancel()
		stopping = append(stopping, running)

		delete(hub.watches, identity)
		delete(hub.states, identity)
	}

	hub.mutex.Unlock()

	for _, stopped := range stopping {
		<-stopped.done
	}
}

func (hub *Hub) start(updateContext context.Context, identity string, source brick.LiveSource) watch {
	watchContext, cancel := context.WithCancel(updateContext)
	done := make(chan struct{})

	go func() {
		defer close(done)

		source.Watch(watchContext, func(state brick.LiveState) {
			hub.mutex.Lock()
			_, stillRunning := hub.watches[identity]

			if stillRunning {
				hub.states[identity] = state
			}
			hub.mutex.Unlock()

			if stillRunning {
				hub.notify(identity)
			}
		})
	}()

	return watch{cancel: cancel, done: done}
}
