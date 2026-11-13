// Package pipelinestatus is the pipeline_status brick: a key that shows a
// workflow's latest run, keeps updating while its page is not shown, and opens
// the run when pressed (specification section 2, TD10).
package pipelinestatus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/launching"
	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// KindName is the kind's name in configuration.
const KindName = "pipeline_status"

// The defaults for the two optional fields, both settled by the author: a
// configurable host defaulting to github.com (TD7), and a minute between polls
// (TD8), which the interval field's bounds keep at 30 seconds or more.
const (
	defaultAPIBaseURL = "https://api.github.com"
	defaultInterval   = time.Minute
	// activeInterval is how often a key polls while its run is queued or in
	// progress, whatever its interval: a minute left a finished or failing run
	// showing as running for too long on the real deck, and the author chose
	// 15 seconds (evaluation, deck testing). GitHub's own longer waits still win.
	activeInterval = 15 * time.Second
)

// noRunCategory is shown when a press finds no run to open yet.
const noRunCategory = "no_run"

// errNoRun reports a press before any run is known.
var errNoRun = errors.New("no run to open yet")

// identitySeparator joins the definition's parts into an identity; it cannot
// appear in any of them, since every part has a format that excludes it.
const identitySeparator = "\x1f"

// Parameters is the kind's one declaration of its parameters.
type Parameters struct {
	brick.Appearance

	Repository string        `brick:"repository,required"        bounds:"3,140"  description:"The GitHub repository, written owner/name."`
	Workflow   string        `brick:"workflow,required"          bounds:"1,255"  format:"token"                                                                                                                                                                description:"The workflow's file name, such as ci.yml, or its numeric ID."`
	Branch     string        `brick:"branch,required"            bounds:"1,255"  format:"branch"                                                                                                                                                               description:"The branch whose runs the key shows."`
	Credential string        `brick:"credential,required,secret" bounds:"1,64"   format:"identifier"                                                                                                                                                           description:"The name of the stored GitHub token, a fine grained token with read access to Actions."`
	Interval   time.Duration `brick:"interval"                   bounds:"30s,1h" description:"How often to poll when no run is active; a minute if left out. Every 15 seconds while a run is queued or in progress. GitHub may ask for longer, and is obeyed."`
	APIBaseURL string        `brick:"api_base_url"               bounds:"9,2048" format:"https_url"                                                                                                                                                            description:"The GitHub API address for GitHub Enterprise; api.github.com if left out."`
}

// Register adds the kind to a registry.
func Register(registry *brick.Registry) error {
	return brick.Register(registry, KindName, construct) //nolint:wrapcheck // Register already names the kind
}

func construct(parameters Parameters, ports brick.Ports) (any, error) {
	if ports.PipelineProvider == nil || ports.URLOpener == nil {
		return nil, fmt.Errorf("%w: %s needs a pipeline provider and a URL opener", brick.ErrPortMissing, KindName)
	}

	query := pipeline.Query{
		APIBaseURL:     parameters.APIBaseURL,
		Repository:     parameters.Repository,
		Workflow:       parameters.Workflow,
		Branch:         parameters.Branch,
		CredentialName: parameters.Credential,
	}
	if query.APIBaseURL == "" {
		query.APIBaseURL = defaultAPIBaseURL
	}

	interval := parameters.Interval
	if interval == 0 {
		interval = defaultInterval
	}

	return pipelineBrick{query: query, interval: interval, provider: ports.PipelineProvider, opener: ports.URLOpener}, nil
}

// pipelineBrick watches one workflow.
type pipelineBrick struct {
	query    pipeline.Query
	interval time.Duration
	provider pipeline.Provider
	opener   interface {
		OpenURL(openContext context.Context, address string) error
	}
}

// LiveIdentity is every part of the definition that changes what is polled.
// The appearance is not part of it, so relabelling a key does not restart its
// poller.
func (watcher pipelineBrick) LiveIdentity() string {
	return strings.Join([]string{
		KindName, watcher.query.APIBaseURL, watcher.query.Repository, watcher.query.Workflow,
		watcher.query.Branch, watcher.query.CredentialName, watcher.interval.String(),
	}, identitySeparator)
}

// Watch polls until its context ends, publishing each result. It waits the
// configured interval between polls, or longer when GitHub asks.
func (watcher pipelineBrick) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	for {
		result, pollError := watcher.provider.LatestRun(watchContext, watcher.query)
		if watchContext.Err() != nil {
			return
		}

		wait := watcher.interval

		if pollError != nil {
			publish(brick.LiveState{
				Status: render.StatusNone, ErrorReason: brick.ReasonCategoryOf(pollError), TargetURL: "",
				Picture: nil, Playing: false,
			})

			if failure, isFailure := errors.AsType[*pipeline.PollError](pollError); isFailure {
				wait = max(wait, failure.RetryAfter)
			}
		} else {
			publish(brick.LiveState{
				Status: statusOf(result.Run.State), ErrorReason: "", TargetURL: result.Run.TargetURL, Picture: nil, Playing: false,
			})

			if result.Run.State == pipeline.StateQueued || result.Run.State == pipeline.StateInProgress {
				wait = min(wait, activeInterval)
			}

			wait = max(wait, result.PollAfter)
		}

		timer := time.NewTimer(wait)

		select {
		case <-watchContext.Done():
			timer.Stop()

			return
		case <-timer.C:
		}
	}
}

// PressWithState opens the run the latest state names (TD10 part b).
func (watcher pipelineBrick) PressWithState(pressContext context.Context, state brick.LiveState) error {
	if state.TargetURL == "" {
		return fmt.Errorf("pressing a pipeline key: %w", brick.Categorised(noRunCategory, errNoRun))
	}

	runOpener := launching.New(state.TargetURL, watcher.opener.OpenURL, "opening the run")

	return runOpener.Press(pressContext) //nolint:wrapcheck // launching names the action
}

func statusOf(state pipeline.State) render.Status {
	switch state {
	case pipeline.StateSuccess:
		return render.StatusSuccess
	case pipeline.StateFailure:
		return render.StatusFailure
	case pipeline.StateInProgress:
		return render.StatusInProgress
	case pipeline.StateQueued:
		return render.StatusQueued
	case pipeline.StateCancelled:
		return render.StatusCancelled
	case pipeline.StateUnknown:
		return render.StatusUnknown
	default:
		return render.StatusUnknown
	}
}
