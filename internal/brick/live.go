package brick

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// LiveState is what a live brick last reported: its status, a reason category
// if it is in error, and the address a press opens, if any.
type LiveState struct {
	Status      render.Status
	ErrorReason string
	TargetURL   string
}

// LiveSource is a brick that publishes states over time until its context is
// cancelled (specification section 2). Bricks with the same identity watch the
// same source, so the live hub runs one of them and shares its states.
type LiveSource interface {
	LiveIdentity() string
	Watch(watchContext context.Context, publish func(state LiveState))
}

// LivePressable is a live brick whose press acts on its latest state, such as
// opening the run a pipeline key shows.
type LivePressable interface {
	PressWithState(pressContext context.Context, state LiveState) error
}
