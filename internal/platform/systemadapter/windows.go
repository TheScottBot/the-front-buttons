package systemadapter

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// AppWindow is one open window: its application's name and identifier, its
// title, whether it is the window in front, and its place in the snapshot it
// came from, by which it is raised.
type AppWindow struct {
	Application string
	Identifier  string
	Title       string
	Focused     bool
	Index       int
}

// WindowSnapshot is the windows open at one moment, held until released so
// one of them can be raised.
type WindowSnapshot interface {
	Windows() []AppWindow
	Raise(index int) error
	Release()
}

// WindowLister snapshots the standard windows of every running application
// whose identifier starts with a prefix.
type WindowLister func(listContext context.Context, prefix string) (WindowSnapshot, error)

// WindowCycler moves between a group's windows: it reads them, puts them in
// order of their application then their title, so the order holds however
// macOS lists them, and raises the one a step from the window in front.
type WindowCycler struct {
	trusted func() bool
	list    WindowLister
	ask     func()
}

// NewWindowCyclerWith returns a WindowCycler that asks trusted whether it may
// read windows, reads them with list, and calls ask when refused, which asks
// macOS for Accessibility and has the daemon restart once it is granted, as
// a refused F13 does.
func NewWindowCyclerWith(trusted func() bool, list WindowLister, ask func()) WindowCycler {
	return WindowCycler{trusted: trusted, list: list, ask: ask}
}

// CycleWindows raises the window a step from the one in front, wrapping
// round. When no window of the group is in front, a step forward goes to the
// first and a step back to the last.
func (cycler WindowCycler) CycleWindows(cycleContext context.Context, group platform.WindowGroup, step int) error {
	if !cycler.trusted() {
		cycler.ask()

		return fmt.Errorf("%w: grant The Front Buttons Accessibility in System Settings, Privacy and Security, "+
			"and it restarts itself to use it", platform.ErrWindowsNotPermitted)
	}

	snapshot, listError := cycler.list(cycleContext, group.Prefix)
	if listError != nil {
		return fmt.Errorf("reading windows: %w", listError)
	}

	defer snapshot.Release()

	windows := slices.DeleteFunc(slices.Clone(snapshot.Windows()), func(window AppWindow) bool {
		return slices.Contains(group.Excluded, window.Identifier)
	})
	if len(windows) == 0 {
		return platform.ErrNoWindow
	}

	slices.SortStableFunc(windows, func(first AppWindow, second AppWindow) int {
		return cmp.Or(cmp.Compare(first.Application, second.Application), cmp.Compare(first.Title, second.Title),
			cmp.Compare(first.Index, second.Index))
	})

	target := 0
	if step < 0 {
		target = len(windows) - 1
	}

	if current := slices.IndexFunc(windows, func(window AppWindow) bool { return window.Focused }); current >= 0 {
		target = ((current+step)%len(windows) + len(windows)) % len(windows)
	}

	raiseError := snapshot.Raise(windows[target].Index)
	if raiseError != nil {
		return fmt.Errorf("raising %s: %w", windows[target].Application, raiseError)
	}

	return nil
}
