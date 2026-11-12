// Package pipeline is the PipelineProvider port: the latest run of a workflow,
// in states that belong to no one CI provider, so another provider can be added
// behind the same port later (specification section 1).
package pipeline

import (
	"context"
	"time"
)

// State is a run's state as a key shows it (TD10).
type State string

// The run states.
const (
	StateSuccess    State = "success"
	StateFailure    State = "failure"
	StateInProgress State = "in_progress"
	StateQueued     State = "queued"
	StateCancelled  State = "cancelled"
	StateUnknown    State = "unknown"
)

// Query names one workflow on one branch, and the credential to read it with.
type Query struct {
	APIBaseURL     string
	Repository     string
	Workflow       string
	Branch         string
	CredentialName string
}

// Run is a workflow's latest state and the run a press opens (TD10).
type Run struct {
	State     State
	TargetURL string
}

// Result is one poll's answer. PollAfter is the shortest wait the provider asked
// for before the next poll, zero if it asked for none.
type Result struct {
	Run       Run
	PollAfter time.Duration
	FromCache bool
}

// PollError is a poll that failed, with the reason category its key shows and,
// when the provider asked for one, how long to wait before trying again. It
// satisfies brick.ReasonCategorised.
type PollError struct {
	Category   string
	RetryAfter time.Duration
	Cause      error
}

func (failure *PollError) Error() string {
	if failure.Cause == nil {
		return "pipeline poll failed: " + failure.Category
	}

	return "pipeline poll failed: " + failure.Category + ": " + failure.Cause.Error()
}

func (failure *PollError) Unwrap() error { return failure.Cause }

// ReasonCategory is the short reason the key shows.
func (failure *PollError) ReasonCategory() string { return failure.Category }

// Provider returns a workflow's latest run.
type Provider interface {
	LatestRun(pollContext context.Context, query Query) (Result, error)
}
