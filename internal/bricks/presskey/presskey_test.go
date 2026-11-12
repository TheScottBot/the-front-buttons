package presskey_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/presskey"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// errSomethingElse is a failure that is neither permission nor an unsupported
// key.
var errSomethingElse = errors.New("anything else")

// keyPresserTestDouble records the keys pressed and fails with a set error.
type keyPresserTestDouble struct {
	pressed *[]string
	failure error
}

func (presser keyPresserTestDouble) PressKey(_ context.Context, key string) error {
	*presser.pressed = append(*presser.pressed, key)

	return presser.failure
}

//nolint:ireturn // the brick is exercised through the interface the daemon uses
func constructWith(t *testing.T, presser platform.KeyPresser, key string) brick.Pressable {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := presskey.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(presskey.KindName)
	parameters := kind.Schema().NewParameters()
	keyField, _ := kind.Schema().Field("key")

	assignError := kind.Schema().Assign(parameters, keyField, key)
	if assignError != nil {
		t.Fatal(assignError)
	}

	constructed, constructError := kind.Construct(parameters, brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: presser,
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	pressable, _ := constructed.(brick.Pressable)

	return pressable
}

func TestPressKey_PressesItsKey(t *testing.T) {
	t.Parallel()

	var pressed []string

	pressError := constructWith(t, keyPresserTestDouble{pressed: &pressed, failure: nil}, "F13").Press(t.Context())
	if pressError != nil || len(pressed) != 1 || pressed[0] != "F13" {
		t.Fatalf("want F13 pressed once, got %v (%v)", pressed, pressError)
	}
}

// TestPressKey_ShowsWhyAPressFailed: macOS needs Accessibility permission to
// send a key and has no F21 to F24, so either failure is named on the key
// rather than a press silently doing nothing.
func TestPressKey_ShowsWhyAPressFailed(t *testing.T) {
	t.Parallel()

	for failure, want := range map[error]string{
		fmt.Errorf("posting: %w", platform.ErrKeyNotPermitted): "permission",
		fmt.Errorf("mapping: %w", platform.ErrKeyUnsupported):  "unsupported_key",
		errSomethingElse: "key_failed",
	} {
		var pressed []string

		pressError := constructWith(t, keyPresserTestDouble{pressed: &pressed, failure: failure}, "F21").Press(t.Context())
		if category := brick.ReasonCategoryOf(pressError); category != want {
			t.Errorf("%v: want %q, got %q", failure, want, category)
		}
	}
}

func TestPressKey_DeclaresARequiredFunctionKey(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := presskey.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(presskey.KindName)

	keyField, found := kind.Schema().Field("key")
	if !found || !keyField.Required || keyField.Format != brick.FormatFunctionKey {
		t.Fatalf("want a required function_key field, got %+v", keyField)
	}

	_, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
	})
	if !errors.Is(constructError, brick.ErrPortMissing) {
		t.Fatalf("want ErrPortMissing without a key presser, got %v", constructError)
	}
}
