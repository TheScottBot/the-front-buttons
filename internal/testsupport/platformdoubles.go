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

// QuittedPrefix marks an application a quitter double was asked to quit, in
// its recorder's list, so launches and quits read in the order they came.
const QuittedPrefix = "quit "

// ApplicationQuitterTestDouble is a fake ApplicationQuitter over a
// LaunchRecorder, recording each quit as QuittedPrefix and the identifier.
type ApplicationQuitterTestDouble struct {
	Recorder *LaunchRecorder
}

// QuitApplication records the identifier, or fails when told to.
func (quitter ApplicationQuitterTestDouble) QuitApplication(_ context.Context, identifier string) error {
	return quitter.Recorder.launch(QuittedPrefix + identifier)
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

// ClipboardTestDouble is a fake ClipboardWriter: it records each text written,
// or fails once when told to.
type ClipboardTestDouble struct {
	mutex    sync.Mutex
	written  []string
	failNext error
}

// NewClipboardTestDouble returns a clipboard nothing has been written to.
func NewClipboardTestDouble() *ClipboardTestDouble {
	return &ClipboardTestDouble{mutex: sync.Mutex{}, written: nil, failNext: nil}
}

// FailNext makes the next write fail with the error given.
func (clipboard *ClipboardTestDouble) FailNext(failure error) {
	clipboard.mutex.Lock()
	defer clipboard.mutex.Unlock()

	clipboard.failNext = failure
}

// WriteClipboard records the text, or fails when told to.
func (clipboard *ClipboardTestDouble) WriteClipboard(_ context.Context, text string) error {
	clipboard.mutex.Lock()
	defer clipboard.mutex.Unlock()

	failure := clipboard.failNext
	if failure != nil {
		clipboard.failNext = nil

		return failure
	}

	clipboard.written = append(clipboard.written, text)

	return nil
}

// Written returns every text written, in order.
func (clipboard *ClipboardTestDouble) Written() []string {
	clipboard.mutex.Lock()
	defer clipboard.mutex.Unlock()

	return append([]string(nil), clipboard.written...)
}
