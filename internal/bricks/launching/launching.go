// Package launching is the press behaviour open_url and open_application share:
// hand one target to a platform port and show any failure on the key by its
// reason category. It exists because the duplicate code check found the two
// bricks repeating it (TE7).
package launching

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// Launch is a platform port's method that opens or launches one target.
type Launch func(launchContext context.Context, target string) error

// Brick presses by launching one target.
type Brick struct {
	target      string
	launch      Launch
	description string
}

// New returns a brick that launches target through launch. The description says
// what a press does, for the error's text; the target itself is never put in an
// error, since a URL can hold colleagues' addresses (specification section 7).
func New(target string, launch Launch, description string) Brick {
	return Brick{target: target, launch: launch, description: description}
}

// Press launches the target.
func (button Brick) Press(pressContext context.Context) error {
	launchError := button.launch(pressContext, button.target)
	if launchError != nil {
		return fmt.Errorf("%s: %w", button.description,
			brick.Categorised(platform.LaunchFailureCategory(launchError), launchError))
	}

	return nil
}
