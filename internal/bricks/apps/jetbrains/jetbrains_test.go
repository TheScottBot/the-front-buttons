package jetbrains_test

import (
	"context"
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/jetbrains"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// presserFake records what it was asked to press, and answers for the
// platform it pretends to be.
type presserFake struct {
	windows bool
	pressed []platform.Shortcut
}

func (fake *presserFake) PressShortcut(_ context.Context, shortcut platform.Shortcut) error {
	fake.pressed = append(fake.pressed, shortcut)

	return nil
}

func (*presserFake) DescribeShortcut(shortcut platform.Shortcut) string { return shortcut.Key }

func (fake *presserFake) ForThisPlatform(pair platform.ShortcutPair) platform.Shortcut {
	if fake.windows {
		return pair.Windows
	}

	return pair.Mac
}

// TestActions_AreAllPressableOnBothPlatforms: every JetBrains action has a
// friendly label and a shortcut on macOS and on Windows whose key a shortcut
// may name, and no name repeats (evaluation, app screens).
func TestActions_AreAllPressableOnBothPlatforms(t *testing.T) {
	t.Parallel()

	names := make([]string, 0, len(jetbrains.Actions()))

	for _, action := range jetbrains.Actions() {
		if slices.Contains(names, action.Name) {
			t.Errorf("%s: the name repeats", action.Name)
		}

		names = append(names, action.Name)

		if action.Label == "" {
			t.Errorf("%s: want a label", action.Name)
		}

		for platformName, shortcut := range map[string]platform.Shortcut{"macOS": action.Mac, "Windows": action.Windows} {
			if !brick.CheckFormat(brick.FormatShortcutKey, shortcut.Key) {
				t.Errorf("%s on %s: %q is not a key a shortcut may name", action.Name, platformName, shortcut.Key)
			}
		}
	}

	for _, wanted := range []string{"build", "run", "debug", "step_over", "comment_line", "terminal", "commit"} {
		if !slices.Contains(names, wanted) {
			t.Errorf("want the %s action", wanted)
		}
	}
}

// TestRegisterActions_MakesAScopedKindForEachAction: an IDE's set registers
// each action as its own kind, scoped to the set and named with its prefix;
// pressing one sends the shortcut for the platform it runs on, and the key
// shows the action's label until given its own.
func TestRegisterActions_MakesAScopedKindForEachAction(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := jetbrains.RegisterActions(registry, "goland", "goland_")
	if registerError != nil {
		t.Fatal(registerError)
	}

	if len(registry.KindNamesIn("goland")) != len(jetbrains.Actions()) || len(registry.KindNamesIn("")) != 0 {
		t.Fatalf("want one scoped kind per action, got %v", registry.KindNames())
	}

	kind, found := registry.Kind("goland_build")
	if !found || kind.Scope() != "goland" {
		t.Fatalf("want goland_build scoped to goland, got %+v", kind)
	}

	build := actionNamed(t, "build")

	for _, onWindows := range []bool{false, true} {
		assertPressesBuild(t, kind, build, onWindows)
	}
}

// assertPressesBuild constructs the build key on one platform and checks what
// a press sends and how the key describes itself.
func assertPressesBuild(t *testing.T, kind brick.Kind, build jetbrains.Action, onWindows bool) {
	t.Helper()

	presser := &presserFake{windows: onWindows, pressed: nil}

	constructed, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
		MediaPlayer: nil, WindowCycler: nil, ShortcutPresser: presser,
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	pressable, _ := constructed.(brick.Pressable)

	pressError := pressable.Press(t.Context())

	want := build.Mac
	if onWindows {
		want = build.Windows
	}

	if pressError != nil || len(presser.pressed) != 1 || presser.pressed[0] != want {
		t.Errorf("windows %v: want %+v pressed, got %+v (%v)", onWindows, want, presser.pressed, pressError)
	}

	labelled, _ := constructed.(brick.DefaultLabelled)
	if labelled == nil || labelled.DefaultLabel() != build.Label {
		t.Error("want the key to show the action's label")
	}

	if _, sends := constructed.(brick.KeystrokeSender); !sends {
		t.Error("want the key marked as sending keystrokes, so its application comes forward first")
	}
}

// actionNamed finds one action in the table.
func actionNamed(t *testing.T, name string) jetbrains.Action {
	t.Helper()

	index := slices.IndexFunc(jetbrains.Actions(), func(action jetbrains.Action) bool { return action.Name == name })
	if index < 0 {
		t.Fatalf("want the %s action", name)
	}

	return jetbrains.Actions()[index]
}
