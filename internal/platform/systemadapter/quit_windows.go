package systemadapter

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// ApplicationQuitter is the ApplicationQuitter on Windows, not built yet: it
// reports quitting unavailable.
type ApplicationQuitter struct{}

// NewApplicationQuitter returns this machine's ApplicationQuitter.
func NewApplicationQuitter() ApplicationQuitter { return ApplicationQuitter{} }

// QuitApplication reports quitting unavailable.
func (ApplicationQuitter) QuitApplication(context.Context, string) error {
	return platform.ErrQuitUnavailable
}
