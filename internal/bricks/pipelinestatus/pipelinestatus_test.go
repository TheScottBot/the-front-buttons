package pipelinestatus_test

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/pipelinestatus"
	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const failedRunURL = "https://github.com/octo-org/hello-world/actions/runs/8"

func ranWith(state pipeline.State, targetURL string, pollAfter time.Duration) testsupport.PipelineAnswer {
	return testsupport.PipelineAnswer{
		Result: pipeline.Result{
			Run: pipeline.Run{State: state, TargetURL: targetURL}, PollAfter: pollAfter, FromCache: false,
		},
		Failure: nil,
	}
}

func failedWith(category string, retryAfter time.Duration) testsupport.PipelineAnswer {
	return testsupport.PipelineAnswer{
		Result:  pipeline.Result{Run: pipeline.Run{State: "", TargetURL: ""}, PollAfter: 0, FromCache: false},
		Failure: &pipeline.PollError{Category: category, RetryAfter: retryAfter, Cause: nil},
	}
}

func baseParameters() map[string]any {
	return map[string]any{
		"repository": "octo-org/hello-world",
		"workflow":   "ci.yml",
		"branch":     "main",
		"credential": "github-work",
	}
}

func construct(
	t *testing.T,
	values map[string]any,
	provider pipeline.Provider,
	recorder *testsupport.LaunchRecorder,
) any {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := pipelinestatus.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(pipelinestatus.KindName)
	parameters := kind.Schema().NewParameters()

	for fieldName, value := range values {
		field, found := kind.Schema().Field(fieldName)
		if !found {
			t.Fatalf("no field %s", fieldName)
		}

		assignError := kind.Schema().Assign(parameters, field, value)
		if assignError != nil {
			t.Fatalf("assigning %s: %v", fieldName, assignError)
		}
	}

	constructed, constructError := kind.Construct(parameters, brick.Ports{
		URLOpener:           testsupport.URLOpenerTestDouble{Recorder: recorder},
		ApplicationLauncher: nil,
		PipelineProvider:    provider,
	})
	if constructError != nil {
		t.Fatalf("constructing: %v", constructError)
	}

	return constructed
}

// publishedStates collects what a source publishes.
type publishedStates struct {
	mutex  sync.Mutex
	states []brick.LiveState
}

func (published *publishedStates) publish(state brick.LiveState) {
	published.mutex.Lock()
	defer published.mutex.Unlock()

	published.states = append(published.states, state)
}

func (published *publishedStates) all() []brick.LiveState {
	published.mutex.Lock()
	defer published.mutex.Unlock()

	return append([]brick.LiveState(nil), published.states...)
}

func watchFor(t *testing.T, source brick.LiveSource, duration time.Duration) []brick.LiveState {
	t.Helper()

	published := &publishedStates{mutex: sync.Mutex{}, states: nil}
	watchContext, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		source.Watch(watchContext, published.publish)
		close(done)
	}()

	time.Sleep(duration)
	synctest.Wait()
	cancel()
	<-done

	return published.all()
}

func TestPipelineStatus_IsALiveSourceAndALivePressable(t *testing.T) {
	t.Parallel()

	constructed := construct(t, baseParameters(), testsupport.NewPipelineProviderTestDouble(
		ranWith(pipeline.StateSuccess, failedRunURL, 0)), testsupport.NewLaunchRecorder())

	_, isSource := constructed.(brick.LiveSource)
	_, isPressable := constructed.(brick.LivePressable)

	if !isSource || !isPressable {
		t.Fatalf("want a live source and live pressable, got %T", constructed)
	}
}

func TestPipelineStatus_PublishesEachRunStateAsAKeyStatus(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		provider := testsupport.NewPipelineProviderTestDouble(
			ranWith(pipeline.StateInProgress, "", 0),
			ranWith(pipeline.StateFailure, failedRunURL, 0),
		)
		source, _ := construct(t, baseParameters(), provider, testsupport.NewLaunchRecorder()).(brick.LiveSource)

		states := watchFor(t, source, 61*time.Second)

		if len(states) != 2 || states[0].Status != render.StatusInProgress || states[1].Status != render.StatusFailure ||
			states[1].TargetURL != failedRunURL {
			t.Fatalf("want in progress then failure with its run, got %+v", states)
		}
	})
}

func TestPipelineStatus_PollsAtTheDefaultInterval_AgainstTheDefaultHost(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		provider := testsupport.NewPipelineProviderTestDouble(ranWith(pipeline.StateSuccess, "", 0))
		source, _ := construct(t, baseParameters(), provider, testsupport.NewLaunchRecorder()).(brick.LiveSource)

		watchFor(t, source, 3*time.Minute)

		queries := provider.Queries()
		if len(queries) != 4 || queries[0].APIBaseURL != "https://api.github.com" {
			t.Fatalf("want polls at 0, 60, 120 and 180 seconds against api.github.com, got %d: %+v",
				len(queries), queries)
		}
	})
}

func TestPipelineStatus_WaitsLongerWhenGitHubAsks(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		provider := testsupport.NewPipelineProviderTestDouble(
			failedWith("rate_limited", 5*time.Minute),
			ranWith(pipeline.StateSuccess, "", 2*time.Minute),
			ranWith(pipeline.StateSuccess, "", 0),
		)
		source, _ := construct(t, baseParameters(), provider, testsupport.NewLaunchRecorder()).(brick.LiveSource)

		states := watchFor(t, source, 6*time.Minute+30*time.Second)

		if queries := provider.Queries(); len(queries) != 2 {
			t.Fatalf("want polls at 0 and 5 minutes only, got %d", len(queries))
		}

		if states[0].ErrorReason != "rate_limited" {
			t.Fatalf("want the rate limit shown on the key, got %+v", states[0])
		}
	})
}

func TestPipelineStatus_PressOpensTheRunFromItsState(t *testing.T) {
	t.Parallel()

	recorder := testsupport.NewLaunchRecorder()
	pressable, _ := construct(t, baseParameters(), testsupport.NewPipelineProviderTestDouble(
		ranWith(pipeline.StateSuccess, "", 0)), recorder).(brick.LivePressable)

	pressError := pressable.PressWithState(t.Context(),
		brick.LiveState{Status: render.StatusFailure, ErrorReason: "", TargetURL: failedRunURL})

	if pressError != nil || len(recorder.Launched()) != 1 || recorder.Launched()[0] != failedRunURL {
		t.Fatalf("want the failing run opened, got %v (%v)", recorder.Launched(), pressError)
	}

	noRunError := pressable.PressWithState(t.Context(),
		brick.LiveState{Status: render.StatusUnknown, ErrorReason: "", TargetURL: ""})
	if brick.ReasonCategoryOf(noRunError) != "no_run" {
		t.Fatalf("want no_run when there is nothing to open, got %v", noRunError)
	}
}

func TestPipelineStatus_IdentityChangesWithAnyPartOfTheDefinition(t *testing.T) {
	t.Parallel()

	identityOf := func(change func(values map[string]any)) string {
		values := baseParameters()
		change(values)

		source, _ := construct(t, values, testsupport.NewPipelineProviderTestDouble(
			ranWith(pipeline.StateSuccess, "", 0)), testsupport.NewLaunchRecorder()).(brick.LiveSource)

		return source.LiveIdentity()
	}

	same := identityOf(func(map[string]any) {})
	relabelled := identityOf(func(values map[string]any) { values["label"] = "Build" })

	if same != relabelled {
		t.Fatal("want a label change to keep the identity, so the poller is not restarted")
	}

	for fieldName, value := range map[string]any{
		"repository": "octo-org/other", "workflow": "release.yml", "branch": "develop",
		"credential": "github-home", "interval": 2 * time.Minute, "api_base_url": "https://github.example.test/api/v3",
	} {
		changed := identityOf(func(values map[string]any) { values[fieldName] = value })
		if changed == same {
			t.Errorf("want a change to %s to change the identity", fieldName)
		}
	}
}

// TestPipelineStatus_PollsEveryFifteenSeconds_WhileARunIsActive: a minute
// between polls left a finished or failing run showing as running for too long
// on the real deck; the author chose 15 seconds while a run is queued or in
// progress, and the configured interval otherwise (evaluation, deck testing).
func TestPipelineStatus_PollsEveryFifteenSeconds_WhileARunIsActive(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		provider := testsupport.NewPipelineProviderTestDouble(
			ranWith(pipeline.StateQueued, failedRunURL, 0),
			ranWith(pipeline.StateInProgress, failedRunURL, 0),
			ranWith(pipeline.StateInProgress, failedRunURL, 0),
			ranWith(pipeline.StateSuccess, failedRunURL, 0),
		)
		source, _ := construct(t, baseParameters(), provider, testsupport.NewLaunchRecorder()).(brick.LiveSource)

		watchFor(t, source, 2*time.Minute)

		// Polls at 0, 15, 30 and 45 seconds while active, then a minute after
		// the success: 1:45, which is inside two minutes.
		if queries := provider.Queries(); len(queries) != 5 {
			t.Fatalf("want polls at 0, 15, 30, 45 seconds and 1:45, got %d", len(queries))
		}
	})
}
