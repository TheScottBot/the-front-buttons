package testsupport

import (
	"context"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
)

// PipelineProviderTestDouble answers polls from a script, repeating the last
// answer once the script runs out, and records every query.
type PipelineProviderTestDouble struct {
	mutex   sync.Mutex
	answers []PipelineAnswer
	queries []pipeline.Query
}

// PipelineAnswer is one scripted poll result.
type PipelineAnswer struct {
	Result  pipeline.Result
	Failure error
}

// NewPipelineProviderTestDouble returns a provider that gives the answers in order.
func NewPipelineProviderTestDouble(answers ...PipelineAnswer) *PipelineProviderTestDouble {
	return &PipelineProviderTestDouble{mutex: sync.Mutex{}, answers: answers, queries: nil}
}

// LatestRun returns the next scripted answer.
func (provider *PipelineProviderTestDouble) LatestRun(
	_ context.Context,
	query pipeline.Query,
) (pipeline.Result, error) {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()

	provider.queries = append(provider.queries, query)

	answer := provider.answers[0]
	if len(provider.answers) > 1 {
		provider.answers = provider.answers[1:]
	}

	return answer.Result, answer.Failure
}

// Queries returns every query received.
func (provider *PipelineProviderTestDouble) Queries() []pipeline.Query {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()

	return append([]pipeline.Query(nil), provider.queries...)
}
