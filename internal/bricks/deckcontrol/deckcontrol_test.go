package deckcontrol_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/deckcontrol"
)

// noPorts are the ports a deck control needs: none, since the daemon carries
// out its request on the deck it owns.
func noPorts() brick.Ports {
	return brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
	}
}

// construct builds a kind with its level set, or left out when level is zero.
func construct(t *testing.T, kindName string, level int) (any, error) {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := deckcontrol.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, found := registry.Kind(kindName)
	if !found {
		t.Fatalf("want the %s kind registered", kindName)
	}

	parameters := kind.Schema().NewParameters()

	if level != 0 {
		levelField, _ := kind.Schema().Field("level")

		assignError := kind.Schema().Assign(parameters, levelField, int64(level))
		if assignError != nil {
			return nil, assignError //nolint:wrapcheck // the test inspects the schema's own error
		}
	}

	return kind.Construct(parameters, noPorts()) //nolint:wrapcheck // the test inspects the construction's own error
}

func requestOf(t *testing.T, constructed any) brick.DeckControlRequest {
	t.Helper()

	control, isControl := constructed.(brick.DeckControl)
	if !isControl {
		t.Fatalf("want a deck control, got %T", constructed)
	}

	return control.DeckControl()
}

// TestBrightness_AsksForItsLevel: the author asked for a key that toggles the
// deck between its usual brightness and a set percentage.
func TestBrightness_AsksForItsLevel(t *testing.T) {
	t.Parallel()

	constructed, constructError := construct(t, deckcontrol.BrightnessKindName, 40)
	if constructError != nil {
		t.Fatal(constructError)
	}

	want := brick.DeckControlRequest{Action: brick.DeckControlBrightness, Percent: 40}
	if got := requestOf(t, constructed); got != want {
		t.Fatalf("want %+v, got %+v", want, got)
	}
}

// TestOff_AsksToBlankTheOtherKeys_AtALowLevel: the author asked for a key that
// turns every other key off and turns the deck down low; ten percent unless
// set.
func TestOff_AsksToBlankTheOtherKeys_AtALowLevel(t *testing.T) {
	t.Parallel()

	for level, want := range map[int]int{0: 10, 25: 25} {
		constructed, constructError := construct(t, deckcontrol.OffKindName, level)
		if constructError != nil {
			t.Fatal(constructError)
		}

		wantRequest := brick.DeckControlRequest{Action: brick.DeckControlOff, Percent: want}
		if got := requestOf(t, constructed); got != wantRequest {
			t.Errorf("level %d: want %+v, got %+v", level, wantRequest, got)
		}
	}
}

// TestDeckControls_RefuseALevelOffTheScale: one to a hundred percent. Nought
// would leave a deck too dark to find the key that brings it back.
func TestDeckControls_RefuseALevelOffTheScale(t *testing.T) {
	t.Parallel()

	for _, kindName := range []string{deckcontrol.BrightnessKindName, deckcontrol.OffKindName} {
		for _, level := range []int{-1, 101} {
			_, constructError := construct(t, kindName, level)
			fieldError, isFieldError := errors.AsType[*brick.FieldError](constructError)

			if !isFieldError || fieldError.Reason != brick.ReasonOutOfBounds {
				t.Errorf("%s level %d: want refused as out of bounds, got %v", kindName, level, constructError)
			}
		}
	}

	_, missingError := construct(t, deckcontrol.BrightnessKindName, 0)
	if missingError == nil {
		t.Error("want a brightness key without a level refused")
	}
}
