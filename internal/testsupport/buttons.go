package testsupport

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The fake pressable kinds for daemon tests: a plain button, and one that
// counts as sending key presses, as press_shortcut does.
const (
	ButtonKindName          = "test_button"
	KeystrokeButtonKindName = "test_keystroke_button"
)

// errButtonFailedOnPurpose is what a button configured to fail returns.
var errButtonFailedOnPurpose = errors.New("fake button failed on purpose")

// ButtonParameters is the fake button's parameters: a name to record presses
// under, and whether a press fails.
type ButtonParameters struct {
	brick.Appearance

	Name  string `brick:"name,required" bounds:"1,32"                                         description:"Name the press is recorded under."`
	Fails bool   `brick:"fails"         description:"Whether pressing the fake button fails."`
}

// PressRecorder records which fake buttons were pressed, in order.
type PressRecorder struct {
	mutex   sync.Mutex
	presses []string
}

// NewPressRecorder returns an empty recorder.
func NewPressRecorder() *PressRecorder {
	return &PressRecorder{mutex: sync.Mutex{}, presses: nil}
}

// Presses returns the names pressed so far.
func (recorder *PressRecorder) Presses() []string {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	return append([]string(nil), recorder.presses...)
}

func (recorder *PressRecorder) record(name string) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	recorder.presses = append(recorder.presses, name)
}

// buttonTestDouble is what the fake button kind constructs.
type buttonTestDouble struct {
	parameters ButtonParameters
	recorder   *PressRecorder
}

// Press records the press, and fails if the button was configured to.
func (button *buttonTestDouble) Press(context.Context) error {
	button.recorder.record(button.parameters.Name)

	if button.parameters.Fails {
		return errButtonFailedOnPurpose
	}

	return nil
}

// keystrokeButtonTestDouble is a fake button that sends key presses.
type keystrokeButtonTestDouble struct {
	buttonTestDouble
}

// SendsKeystrokes marks the button as sending key presses.
func (*keystrokeButtonTestDouble) SendsKeystrokes() {}

// RegisterButtonKind adds the fake button kinds to a registry, recording
// presses into the recorder.
func RegisterButtonKind(registry *brick.Registry, recorder *PressRecorder) error {
	registerError := errors.Join(
		brick.Register(registry, ButtonKindName,
			func(parameters ButtonParameters, _ brick.Ports) (any, error) {
				return &buttonTestDouble{parameters: parameters, recorder: recorder}, nil
			}),
		brick.Register(registry, KeystrokeButtonKindName,
			func(parameters ButtonParameters, _ brick.Ports) (any, error) {
				return &keystrokeButtonTestDouble{parameters: parameters, recorder: recorder}, nil
			}),
	)
	if registerError != nil {
		return fmt.Errorf("registering the fake button kinds: %w", registerError)
	}

	return nil
}

// ActivatorTestDouble brings forward only the applications it is told are
// running, recording each into the press recorder as "forward <identifier>",
// so a test sees it came before the press. Those it is told are slow take
// SlowActivation to come forward.
type ActivatorTestDouble struct {
	Recorder *PressRecorder
	Running  []string
	Slow     []string
}

// SlowActivation is how long a slow application takes to come forward.
const SlowActivation = 300 * time.Millisecond

// ActivateApplication records the application, or fails if it is not running.
func (activator ActivatorTestDouble) ActivateApplication(_ context.Context, identifier string) error {
	if slices.Contains(activator.Slow, identifier) {
		time.Sleep(SlowActivation)
	}

	if !slices.Contains(activator.Running, identifier) {
		return fmt.Errorf("%w: %s", platform.ErrApplicationNotRunning, identifier)
	}

	activator.Recorder.record("forward " + identifier)

	return nil
}
