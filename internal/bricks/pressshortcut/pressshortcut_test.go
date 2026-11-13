package pressshortcut_test

import (
	"context"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/pressshortcut"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// presserFake records the shortcuts pressed and names them as a Mac would.
type presserFake struct {
	pressed []platform.Shortcut
	failure error
}

func (fake *presserFake) PressShortcut(_ context.Context, shortcut platform.Shortcut) error {
	fake.pressed = append(fake.pressed, shortcut)

	return fake.failure
}

func (*presserFake) ForThisPlatform(pair platform.ShortcutPair) platform.Shortcut { return pair.Mac }

func (*presserFake) DescribeShortcut(shortcut platform.Shortcut) string {
	text := ""
	if shortcut.Command {
		text = "⌘"
	}

	return text + shortcut.Key
}

func construct(t *testing.T, presser platform.ShortcutPresser, assign map[string]any) any {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := pressshortcut.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(pressshortcut.KindName)
	parameters := kind.Schema().NewParameters()

	for name, value := range assign {
		field, _ := kind.Schema().Field(name)

		assignError := kind.Schema().Assign(parameters, field, value)
		if assignError != nil {
			t.Fatal(assignError)
		}
	}

	constructed, constructError := kind.Construct(parameters, brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
		MediaPlayer: nil, WindowCycler: nil, ShortcutPresser: presser,
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	return constructed
}

// TestPressShortcut_PressesTheKeyWithItsModifiers: the author asked for a key
// that sends any shortcut, its modifiers in any combination (evaluation, deck
// testing).
func TestPressShortcut_PressesTheKeyWithItsModifiers(t *testing.T) {
	t.Parallel()

	presser := &presserFake{pressed: nil, failure: nil}
	constructed := construct(t, presser, map[string]any{"key": "C", "command": true, "shift": true})

	pressable, _ := constructed.(brick.Pressable)

	pressError := pressable.Press(t.Context())
	want := platform.Shortcut{Key: "C", Control: false, Alt: false, Shift: true, Command: true}

	if pressError != nil || len(presser.pressed) != 1 || presser.pressed[0] != want {
		t.Fatalf("want %+v pressed, got %+v (%v)", want, presser.pressed, pressError)
	}

	labelled, isLabelled := constructed.(brick.DefaultLabelled)
	if !isLabelled || labelled.DefaultLabel() != "⌘C" {
		t.Error("want the key to show its shortcut as the platform names it, when it has no label")
	}

	refused := &presserFake{pressed: nil, failure: platform.ErrKeyNotPermitted}
	refusedKey, _ := construct(t, refused, map[string]any{"key": "V"}).(brick.Pressable)

	if reason := brick.ReasonCategoryOf(refusedKey.Press(t.Context())); reason != "permission" {
		t.Errorf("want a refusal shown as permission, got %q", reason)
	}
}
