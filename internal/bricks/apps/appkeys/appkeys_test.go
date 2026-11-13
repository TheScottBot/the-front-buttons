package appkeys_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// sampleActions is a small table: build, pressed differently on each
// platform, and comment.
func sampleActions() []appkeys.Action {
	return []appkeys.Action{
		{
			Name: "build", Label: "Build", Mac: appkeys.Keys("F9", appkeys.Command),
			Windows: appkeys.Keys("F9", appkeys.Control),
		},
		{
			Name: "comment", Label: "Comment", Mac: appkeys.Keys("Slash", appkeys.Command),
			Windows: appkeys.Keys("Slash", appkeys.Control),
		},
	}
}

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

// TestCheck_RefusesABadTable: a table with a repeated or empty name, a
// missing label or a key no shortcut may name is refused, and registers
// nothing.
func TestCheck_RefusesABadTable(t *testing.T) {
	t.Parallel()

	checkError := appkeys.Check(sampleActions())
	if checkError != nil {
		t.Fatalf("want the sample accepted, got %v", checkError)
	}

	for name, change := range map[string]func(rows []appkeys.Action){
		"a repeated name": func(rows []appkeys.Action) { rows[1].Name = rows[0].Name },
		"an empty name":   func(rows []appkeys.Action) { rows[0].Name = "" },
		"no label":        func(rows []appkeys.Action) { rows[0].Label = "" },
		"a bad Mac key":   func(rows []appkeys.Action) { rows[0].Mac.Key = "Hyper" },
		"a bad Windows key": func(rows []appkeys.Action) {
			rows[1].Windows.Key = ""
		},
	} {
		rows := sampleActions()
		change(rows)

		rowsError := appkeys.Check(rows)
		if !errors.Is(rowsError, appkeys.ErrBadTable) {
			t.Errorf("%s: want the table refused, got %v", name, rowsError)
		}

		registry := brick.NewRegistry()
		if appkeys.Register(registry, "ides", "ides_", rows) == nil || len(registry.KindNames()) != 0 {
			t.Errorf("%s: want nothing registered", name)
		}
	}
}

// TestRegister_MakesAScopedKindForEachAction: an application's set registers
// each action as its own kind, scoped to the set and named with its prefix;
// pressing one sends the shortcut for the platform it runs on, and the key
// shows the action's label until given its own.
func TestRegister_MakesAScopedKindForEachAction(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := appkeys.Register(registry, "ides", "ides_", sampleActions())
	if registerError != nil {
		t.Fatal(registerError)
	}

	if len(registry.KindNamesIn("ides")) != len(sampleActions()) || len(registry.KindNamesIn("")) != 0 {
		t.Fatalf("want one scoped kind per action, got %v", registry.KindNames())
	}

	kind, found := registry.Kind("ides_build")
	if !found || kind.Scope() != "ides" {
		t.Fatalf("want ides_build scoped to ides, got %+v", kind)
	}

	build := actionNamed(t, "build")

	for _, onWindows := range []bool{false, true} {
		assertPressesBuild(t, kind, build, onWindows)
	}
}

// assertPressesBuild constructs the build key on one platform and checks what
// a press sends and how the key describes itself.
func assertPressesBuild(t *testing.T, kind brick.Kind, build appkeys.Action, onWindows bool) {
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
func actionNamed(t *testing.T, name string) appkeys.Action {
	t.Helper()

	index := slices.IndexFunc(sampleActions(), func(action appkeys.Action) bool { return action.Name == name })
	if index < 0 {
		t.Fatalf("want the %s action", name)
	}

	return sampleActions()[index]
}
