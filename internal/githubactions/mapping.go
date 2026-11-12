package githubactions

import "github.com/TheScottBot/the-front-buttons/internal/pipeline"

// GitHub's run status and conclusion values (evaluation section 5).
const (
	statusCompleted = "completed"

	conclusionSuccess        = "success"
	conclusionFailure        = "failure"
	conclusionTimedOut       = "timed_out"
	conclusionCancelled      = "cancelled"
	conclusionActionRequired = "action_required"
)

// StateOf maps GitHub's status and conclusion to a key state as TD10 settled:
// success; failure or timed out is failure; cancelled; in progress; queued,
// waiting, requested, pending and action required are queued; neutral, skipped,
// stale, no conclusion and anything unrecognised are unknown.
func StateOf(status string, conclusion string) pipeline.State {
	if status == statusCompleted {
		switch conclusion {
		case conclusionSuccess:
			return pipeline.StateSuccess
		case conclusionFailure, conclusionTimedOut:
			return pipeline.StateFailure
		case conclusionCancelled:
			return pipeline.StateCancelled
		case conclusionActionRequired:
			return pipeline.StateQueued
		default:
			return pipeline.StateUnknown
		}
	}

	switch status {
	case "in_progress":
		return pipeline.StateInProgress
	case "queued", "waiting", "requested", "pending", conclusionActionRequired:
		return pipeline.StateQueued
	default:
		return pipeline.StateUnknown
	}
}

// RunSummary is the part of a run the press target needs.
type RunSummary struct {
	State     pipeline.State
	Completed bool
	URL       string
}

// PressTarget picks the run a press opens (TD10 part b): while the newest run is
// still queued or in progress and the most recent completed run failed, the
// failure, so a press finds what broke while the fix runs; otherwise the newest
// run. Runs arrive newest first.
func PressTarget(runs []RunSummary) string {
	if len(runs) == 0 {
		return ""
	}

	newest := runs[0]
	if newest.Completed {
		return newest.URL
	}

	for _, run := range runs[1:] {
		if !run.Completed {
			continue
		}

		if run.State == pipeline.StateFailure {
			return run.URL
		}

		break
	}

	return newest.URL
}
