package openapplication

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The timings of a triple press: each press comes within pressGap of the one
// before, as quick taps on a key do, and after the quit the key waits until
// it has gone quietAfterClosing without a press, so the presses of an
// overeager hand after the third never open the application again.
const (
	pressGap          = 500 * time.Millisecond
	quietAfterClosing = 1500 * time.Millisecond
	pressesToClose    = 3
)

// quitFailedCategory is what the key shows when the application would not
// quit.
const quitFailedCategory = "quit_failed"

// closing is an open_application key with Triple press to close ticked: its
// first press opens the application, and the third quick one quits it.
type closing struct {
	application string
	opening     brick.Pressable
	quitter     platform.ApplicationQuitter

	mutex     sync.Mutex
	presses   int
	lastPress time.Time
	quiet     bool
}

func newClosing(application string, opening brick.Pressable, quitter platform.ApplicationQuitter) *closing {
	return &closing{
		application: application, opening: opening, quitter: quitter,
		mutex: sync.Mutex{}, presses: 0, lastPress: time.Time{}, quiet: false,
	}
}

// Press opens the application on the first of a run of quick presses, quits
// it on the third, and does nothing on the second or on any press while the
// key keeps quiet after a quit.
func (key *closing) Press(pressContext context.Context) error {
	switch key.count(time.Now()) {
	case 1:
		return key.opening.Press(pressContext) //nolint:wrapcheck // the opening brick already says what failed
	case pressesToClose:
		quitError := key.quitter.QuitApplication(pressContext, key.application)
		if quitError != nil {
			return fmt.Errorf("quitting the key's application: %w", brick.Categorised(quitFailedCategory, quitError))
		}
	}

	return nil
}

// count notes a press at now and returns its place in the run of quick
// presses, or 0 for a press while the key keeps quiet after a quit, which
// starts the quiet again.
func (key *closing) count(now time.Time) int {
	key.mutex.Lock()
	defer key.mutex.Unlock()

	since := now.Sub(key.lastPress)
	key.lastPress = now

	if key.quiet && since < quietAfterClosing {
		return 0
	}

	key.quiet = false

	if key.presses > 0 && since <= pressGap {
		key.presses++
	} else {
		key.presses = 1
	}

	if key.presses == pressesToClose {
		key.presses, key.quiet = 0, true

		return pressesToClose
	}

	return key.presses
}
