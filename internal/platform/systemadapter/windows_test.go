package systemadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

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
	onRaise  func(title string)
}

func (fake *snapshotFake) Windows() []systemadapter.AppWindow { return fake.windows }

func (fake *snapshotFake) Raise(index int) error {
	for _, window := range fake.windows {
		if window.Index == index {
			fake.raised = append(fake.raised, window.Title)

			if fake.onRaise != nil {
				fake.onRaise(window.Title)
			}
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

	return &snapshotFake{windows: windows, raised: nil, released: false, onRaise: nil}
}

func cycleFrom(t *testing.T, fake *snapshotFake, trusted bool, step int) error {
	t.Helper()

	return cycleAsking(t, fake, trusted, step, func() {})
}

// cycleAsking cycles with a hook for the request for Accessibility a refusal
// makes.
func cycleAsking(t *testing.T, fake *snapshotFake, trusted bool, step int, ask func()) error {
	t.Helper()

	cycler := systemadapter.NewWindowCyclerWith(func() bool { return trusted },
		func(context.Context, string) (systemadapter.WindowSnapshot, error) { return fake, nil }, ask)

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

	// A refusal asks for Accessibility, as F13's does, so macOS shows its
	// prompt and the daemon restarts itself once it is granted: a running
	// daemon cannot see a grant, and the window keys stayed refused after the
	// author granted it (evaluation, deck testing).
	asked := 0

	refusedError := cycleAsking(t, fake, false, 1, func() { asked++ })
	if !errors.Is(refusedError, platform.ErrWindowsNotPermitted) || len(fake.raised) != 0 || asked != 1 {
		t.Errorf("want a refusal that asks for Accessibility and raises nothing, got %v, %v, asked %d",
			refusedError, fake.raised, asked)
	}

	empty := &snapshotFake{windows: nil, raised: nil, released: false, onRaise: nil}

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

// TestWindowCycler_AddsUpPressesMadeWhileOneIsUnderWay: a busy IDE can take a
// second to list its windows, and presses made meanwhile once ran side by
// side and landed in a burst (evaluation, app screens). Now they add up: the
// cycle under way finishes, then one more moves by every step pressed since,
// from the window the first one raised.
func TestWindowCycler_AddsUpPressesMadeWhileOneIsUnderWay(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		// In order: GoLand punchline, GoLand the-front-buttons, PyCharm
		// scripts, Rider Game. Each listing takes a second and sees the window
		// the last raise brought forward.
		focused, listings, raised := punchTitle, 0, []string(nil)

		list := func(context.Context, string) (systemadapter.WindowSnapshot, error) {
			listings++

			time.Sleep(time.Second)

			snapshot := openWindows(focused)
			snapshot.onRaise = func(title string) { focused = title; raised = append(raised, title) }

			return snapshot, nil
		}

		cycler := systemadapter.NewWindowCyclerWith(func() bool { return true }, list, func() {})
		results := make(chan error, 3)

		for range 3 {
			go func() { results <- cycler.CycleWindows(t.Context(), jetbrains(), 1) }()

			time.Sleep(100 * time.Millisecond)
		}

		for range 3 {
			cycleError := <-results
			if cycleError != nil {
				t.Errorf("want every press to succeed, got %v", cycleError)
			}
		}

		if listings != 2 || !slices.Equal(raised, []string{buttonsTitle, gameTitle}) {
			t.Fatalf("want one step then two, raising %s then %s, got %d listings raising %v",
				buttonsTitle, gameTitle, listings, raised)
		}
	})
}
