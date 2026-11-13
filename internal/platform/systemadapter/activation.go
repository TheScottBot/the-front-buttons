package systemadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// FrontState is what the system says of an application being in front: it
// is, it is not yet, or the system will not say.
type FrontState int

// The front states.
const (
	FrontUnknown FrontState = iota
	FrontYes
	FrontNotYet
)

// WaitUntilInFront checks an application until it is in front, as key presses
// sent before then would reach the window it replaced. A system that will not
// say, as macOS before Accessibility is granted, is taken at its word that the
// application was asked: the key presser then reports the missing permission
// itself.
func WaitUntilInFront(
	waitContext context.Context,
	identifier string,
	inFront func() FrontState,
	wait time.Duration,
	check time.Duration,
) error {
	deadline := time.Now().Add(wait)

	for {
		if inFront() != FrontNotYet {
			return nil
		}

		if !time.Now().Before(deadline) {
			return fmt.Errorf("%w: %s did not come to the front within %s",
				platform.ErrApplicationNotInFront, identifier, wait)
		}

		select {
		case <-waitContext.Done():
			return fmt.Errorf("waiting for %s to come to the front: %w", identifier, waitContext.Err())
		case <-time.After(check):
		}
	}
}
