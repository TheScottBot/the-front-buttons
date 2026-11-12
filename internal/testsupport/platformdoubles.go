package testsupport

import (
	"context"
	"errors"
	"fmt"
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
