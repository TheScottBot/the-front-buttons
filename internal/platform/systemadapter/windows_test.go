package systemadapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// The JetBrains IDE identifiers the window tests use.
const (
	golandID  = "com.jetbrains.goland"
	pycharmID = "com.jetbrains.pycharm"
	riderID   = "com.jetbrains.rider"
	toolboxID = "com.jetbrains.toolbox"
)

// The window titles the window tests use: the author's projects, by IDE.
const (
	gameTitle    = "Game"
	buttonsTitle = "the-front-buttons"
	scriptsTitle = "scripts"
	punchTitle   = "punchline"
	toolboxTitle = "Toolbox"
)

// jetbrains is the group the author's keys cycle.
func jetbrains() platform.WindowGroup {
	return platform.WindowGroup{Prefix: "com.jetbrains.", Excluded: []string{toolboxID}}
}

// snapshotFake is a fixed list of windows that records which it raised and
// whether it was released.
type snapshotFake struct {
	windows  []systemadapter.AppWindow
	raised   []string
	released bool
}

func (fake *snapshotFake) Windows() []systemadapter.AppWindow { return fake.windows }

func (fake *snapshotFake) Raise(index int) error {
	for _, window := range fake.windows {
		if window.Index == index {
			fake.raised = append(fake.raised, window.Title)
		}
	}

	return nil
}

func (fake *snapshotFake) Release() { fake.released = true }

// openWindows are the author's IDEs, listed out of order as macOS might give
// them, with Toolbox's window too; focused names the window in front, or none.
func openWindows(focused string) *snapshotFake {
	windows := []systemadapter.AppWindow{
		{Application: "Rider", Identifier: riderID, Title: gameTitle, Focused: false, Index: 0},
		{Application: "GoLand", Identifier: golandID, Title: buttonsTitle, Focused: false, Index: 1},
		{Application: "PyCharm", Identifier: pycharmID, Title: scriptsTitle, Focused: false, Index: 2},
		{Application: "GoLand", Identifier: golandID, Title: punchTitle, Focused: false, Index: 3},
		{Application: "Toolbox", Identifier: toolboxID, Title: toolboxTitle, Focused: false, Index: 4},
	}

	for index := range windows {
		windows[index].Focused = windows[index].Title == focused
	}

	return &snapshotFake{windows: windows, raised: nil, released: false}
}

func cycleFrom(t *testing.T, fake *snapshotFake, trusted bool, step int) error {
	t.Helper()

	cycler := systemadapter.NewWindowCyclerWith(func() bool { return trusted },
		func(context.Context, string) (systemadapter.WindowSnapshot, error) { return fake, nil })

	// The test inspects the cycler's own error.
	return cycler.CycleWindows(t.Context(), jetbrains(), step) //nolint:wrapcheck // see above
}

// TestWindowCycler_MovesThroughTheIDEsInAStableOrder: the author asked for
// keys that cycle GoLand, PyCharm and Rider windows (evaluation, deck
// testing). Windows go in order of their IDE then their title, whatever order
// macOS lists them in, and wrap round; Toolbox is not an IDE window.
func TestWindowCycler_MovesThroughTheIDEsInAStableOrder(t *testing.T) {
	t.Parallel()

	// In order: GoLand punchline, GoLand the-front-buttons, PyCharm scripts,
	// Rider Game.
	cases := []struct {
		focused string
		step    int
		want    string
	}{
		{punchTitle, 1, buttonsTitle},
		{buttonsTitle, 1, scriptsTitle},
		{scriptsTitle, 1, gameTitle},
		{gameTitle, 1, punchTitle},
		{punchTitle, -1, gameTitle},
		{scriptsTitle, -1, buttonsTitle},
		{"", 1, punchTitle},
		{"", -1, gameTitle},
		{toolboxTitle, 1, punchTitle},
	}

	for _, cycleCase := range cases {
		fake := openWindows(cycleCase.focused)

		cycleError := cycleFrom(t, fake, true, cycleCase.step)
		if cycleError != nil || len(fake.raised) != 1 || fake.raised[0] != cycleCase.want {
			t.Errorf("from %q by %d: want %q raised, got %v (%v)",
				cycleCase.focused, cycleCase.step, cycleCase.want, fake.raised, cycleError)
		}

		if !fake.released {
			t.Errorf("from %q: want the windows released", cycleCase.focused)
		}
	}
}

// TestWindowCycler_RefusesWithoutPermissionOrWindows: without Accessibility
// nothing is read; with no IDE window open there is nothing to move to.
func TestWindowCycler_RefusesWithoutPermissionOrWindows(t *testing.T) {
	t.Parallel()

	fake := openWindows(punchTitle)

	refusedError := cycleFrom(t, fake, false, 1)
	if !errors.Is(refusedError, platform.ErrWindowsNotPermitted) || len(fake.raised) != 0 {
		t.Errorf("want a refusal without Accessibility and nothing raised, got %v, %v", refusedError, fake.raised)
	}

	empty := &snapshotFake{windows: nil, raised: nil, released: false}

	emptyError := cycleFrom(t, empty, true, 1)
	if !errors.Is(emptyError, platform.ErrNoWindow) {
		t.Errorf("want no window to move to, got %v", emptyError)
	}

	for failure, want := range map[error]string{
		platform.ErrWindowsNotPermitted: "permission",
		platform.ErrNoWindow:            "no_window",
		platform.ErrWindowsUnavailable:  "unavailable",
	} {
		if got := platform.WindowFailureCategory(failure); got != want {
			t.Errorf("%v: want category %q, got %q", failure, want, got)
		}
	}
}
