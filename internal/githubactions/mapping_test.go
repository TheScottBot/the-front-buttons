package githubactions_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/githubactions"
	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
)

// The run addresses the press target tests use.
const (
	newestRun   = "https://github.test/runs/9"
	previousRun = "https://github.test/runs/8"
)

func TestStateOf_MapsEveryGitHubStatusAndConclusionAsTD10Settled(t *testing.T) {
	t.Parallel()

	mappings := []struct {
		status     string
		conclusion string
		want       pipeline.State
	}{
		{completed, succeeded, pipeline.StateSuccess},
		{completed, failed, pipeline.StateFailure},
		{completed, "timed_out", pipeline.StateFailure},
		{completed, "cancelled", pipeline.StateCancelled},
		{completed, "neutral", pipeline.StateUnknown},
		{completed, "skipped", pipeline.StateUnknown},
		{completed, "stale", pipeline.StateUnknown},
		{completed, "startup_failure", pipeline.StateUnknown},
		{completed, "", pipeline.StateUnknown},
		{"in_progress", "", pipeline.StateInProgress},
		{"queued", "", pipeline.StateQueued},
		{"waiting", "", pipeline.StateQueued},
		{"requested", "", pipeline.StateQueued},
		{"pending", "", pipeline.StateQueued},
		{"action_required", "", pipeline.StateQueued},
		{completed, "action_required", pipeline.StateQueued},
		{"something_new", "", pipeline.StateUnknown},
	}

	for _, mapping := range mappings {
		if got := githubactions.StateOf(mapping.status, mapping.conclusion); got != mapping.want {
			t.Errorf("%s/%s: want %s, got %s", mapping.status, mapping.conclusion, mapping.want, got)
		}
	}
}

func TestPressTarget_OpensTheLastFailure_WhileAFixIsRunning(t *testing.T) {
	t.Parallel()

	targetCases := map[string]struct {
		runs []githubactions.RunSummary
		want string
	}{
		"latest completed": {
			[]githubactions.RunSummary{{State: pipeline.StateSuccess, Completed: true, URL: newestRun}},
			newestRun,
		},
		"running after a failure": {
			[]githubactions.RunSummary{
				{State: pipeline.StateInProgress, Completed: false, URL: newestRun},
				{State: pipeline.StateFailure, Completed: true, URL: previousRun},
			},
			previousRun,
		},
		"queued after a success": {
			[]githubactions.RunSummary{
				{State: pipeline.StateQueued, Completed: false, URL: newestRun},
				{State: pipeline.StateSuccess, Completed: true, URL: previousRun},
			},
			newestRun,
		},
		"latest failed": {
			[]githubactions.RunSummary{
				{State: pipeline.StateFailure, Completed: true, URL: newestRun},
				{State: pipeline.StateFailure, Completed: true, URL: previousRun},
			},
			newestRun,
		},
	}

	for caseName, targetCase := range targetCases {
		if got := githubactions.PressTarget(targetCase.runs); got != targetCase.want {
			t.Errorf("%s: want %s, got %s", caseName, targetCase.want, got)
		}
	}
}
