package systemadapter

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// ApplicationBadges is the ApplicationBadges on Windows, which is parked
// (evaluation, deck testing): it reports badges unavailable.
type ApplicationBadges struct{}

// NewApplicationBadges returns this machine's ApplicationBadges.
func NewApplicationBadges() ApplicationBadges { return ApplicationBadges{} }

// Badges reports badges unavailable.
func (ApplicationBadges) Badges(context.Context) (map[string]string, error) {
	return nil, platform.ErrBadgesUnavailable
}
