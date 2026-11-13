package testsupport

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// errLaunchTestDoubleFailure is what a launch double returns when told to fail.
var errLaunchTestDoubleFailure = errors.New("fake launch failure")

// LaunchRecorder records what a fake opener or launcher was asked to open, and
// can be told to fail the next request, as a missing application would.
type LaunchRecorder struct {
	mutex    sync.Mutex
	launched []string
	failNext bool
}

// NewLaunchRecorder returns an empty recorder.
func NewLaunchRecorder() *LaunchRecorder {
	return &LaunchRecorder{mutex: sync.Mutex{}, launched: nil, failNext: false}
}

// FailNext makes the next request fail.
func (recorder *LaunchRecorder) FailNext() {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	recorder.failNext = true
}

// Launched returns what has been opened or launched, in order.
func (recorder *LaunchRecorder) Launched() []string {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	return append([]string(nil), recorder.launched...)
}

func (recorder *LaunchRecorder) launch(target string) error {
	checkError := platform.CheckLaunchTarget(target)
	if checkError != nil {
		return fmt.Errorf("fake launch: %w", checkError)
	}

	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	if recorder.failNext {
		recorder.failNext = false

		return fmt.Errorf("%w: %w", platform.ErrLaunchFailed, errLaunchTestDoubleFailure)
	}

	recorder.launched = append(recorder.launched, target)

	return nil
}

// URLOpenerTestDouble is a fake URLOpener over a LaunchRecorder.
type URLOpenerTestDouble struct {
	Recorder *LaunchRecorder
}

// OpenURL records the URL.
func (opener URLOpenerTestDouble) OpenURL(_ context.Context, address string) error {
	return opener.Recorder.launch(address)
}

// ApplicationLauncherTestDouble is a fake ApplicationLauncher over a
// LaunchRecorder.
type ApplicationLauncherTestDouble struct {
	Recorder *LaunchRecorder
}

// LaunchApplication records the identifier.
func (launcher ApplicationLauncherTestDouble) LaunchApplication(_ context.Context, identifier string) error {
	return launcher.Recorder.launch(identifier)
}

// BadgesTestDouble is a fake ApplicationBadges: the badges it is given, or the
// failure, counting each read.
type BadgesTestDouble struct {
	mutex   sync.Mutex
	badges  map[string]string
	failure error
	reads   int
}

// NewBadgesTestDouble returns a double showing no badges.
func NewBadgesTestDouble() *BadgesTestDouble {
	return &BadgesTestDouble{mutex: sync.Mutex{}, badges: map[string]string{}, failure: nil, reads: 0}
}

// Show sets the badges every later read returns.
func (double *BadgesTestDouble) Show(badges map[string]string) {
	double.mutex.Lock()
	defer double.mutex.Unlock()

	double.badges = maps.Clone(badges)
}

// Fail makes every later read fail.
func (double *BadgesTestDouble) Fail(failure error) {
	double.mutex.Lock()
	defer double.mutex.Unlock()

	double.failure = failure
}

// Reads is how many times the badges have been read.
func (double *BadgesTestDouble) Reads() int {
	double.mutex.Lock()
	defer double.mutex.Unlock()

	return double.reads
}

// Badges returns the badges set, or the failure.
func (double *BadgesTestDouble) Badges(context.Context) (map[string]string, error) {
	double.mutex.Lock()
	defer double.mutex.Unlock()

	double.reads++

	if double.failure != nil {
		return nil, double.failure
	}

	return maps.Clone(double.badges), nil
}
