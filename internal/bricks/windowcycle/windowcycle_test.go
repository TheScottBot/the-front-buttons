package windowcycle_test

import (
	"context"
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/windowcycle"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// cyclerFake records each cycle asked of it.
type cyclerFake struct {
	groups  []platform.WindowGroup
	steps   []int
	failure error
}

func (fake *cyclerFake) CycleWindows(_ context.Context, group platform.WindowGroup, step int) error {
	fake.groups = append(fake.groups, group)
	fake.steps = append(fake.steps, step)

	return fake.failure
}

func press(t *testing.T, kindName string, cycler platform.WindowCycler) error {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := windowcycle.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, found := registry.Kind(kindName)
	if !found {
		t.Fatalf("want the %s kind", kindName)
	}

	constructed, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
		MediaPlayer: nil, WindowCycler: cycler,
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	pressable, _ := constructed.(brick.Pressable)

	return pressable.Press(t.Context()) //nolint:wrapcheck // the test inspects the key's own error
}

// TestWindowCycle_KeysMoveThroughTheJetBrainsIDEs: the author asked for next
// and previous window keys across GoLand, PyCharm and Rider: every JetBrains
// application but Toolbox.
func TestWindowCycle_KeysMoveThroughTheJetBrainsIDEs(t *testing.T) {
	t.Parallel()

	for kindName, wantStep := range map[string]int{
		windowcycle.NextKindName:     1,
		windowcycle.PreviousKindName: -1,
	} {
		cycler := &cyclerFake{groups: nil, steps: nil, failure: nil}

		pressError := press(t, kindName, cycler)
		if pressError != nil || len(cycler.steps) != 1 || cycler.steps[0] != wantStep {
			t.Fatalf("%s: want one step of %d, got %v (%v)", kindName, wantStep, cycler.steps, pressError)
		}

		group := cycler.groups[0]
		if group.Prefix != "com.jetbrains." || !slices.Contains(group.Excluded, "com.jetbrains.toolbox") {
			t.Errorf("%s: want the JetBrains IDEs without Toolbox, got %+v", kindName, group)
		}
	}

	refused := &cyclerFake{groups: nil, steps: nil, failure: platform.ErrWindowsNotPermitted}
	if reason := brick.ReasonCategoryOf(press(t, windowcycle.NextKindName, refused)); reason != "permission" {
		t.Errorf("want a refusal shown as permission, got %q", reason)
	}
}
