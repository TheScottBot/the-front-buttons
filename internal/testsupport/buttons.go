package testsupport

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// ButtonKindName is a fake pressable kind for daemon tests.
const ButtonKindName = "test_button"

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

// RegisterButtonKind adds the fake button kind to a registry, recording presses
// into the recorder.
func RegisterButtonKind(registry *brick.Registry, recorder *PressRecorder) error {
	registerError := brick.Register(registry, ButtonKindName,
		func(parameters ButtonParameters, _ brick.Ports) (any, error) {
			return &buttonTestDouble{parameters: parameters, recorder: recorder}, nil
		})
	if registerError != nil {
		return fmt.Errorf("registering the fake button kind: %w", registerError)
	}

	return nil
}
