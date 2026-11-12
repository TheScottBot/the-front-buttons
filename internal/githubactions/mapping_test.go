package githubactions_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/githubactions"
	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
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
		{inProgress, "", pipeline.StateInProgress},
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
