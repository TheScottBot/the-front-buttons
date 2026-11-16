package systemadapter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// On Windows an application key switches to the application when it is
// running, as the open command does on macOS, and launches it only when it is
// not: ShellExecuteW alone starts many applications a second time rather than
// bringing back the one running. The decisions live here, tested on every
// machine; activation_windows.go and quit_windows.go only make the calls.

// How long a Windows application has to come to the front after being asked,
// and how often it is checked meanwhile: the same as on macOS.
const (
	windowsFrontWait  = 500 * time.Millisecond
	windowsFrontCheck = 20 * time.Millisecond
)

// WindowsDesktop is the Windows calls the activator and the quitter make,
// held in one value so tests can drive them without a desktop.
type WindowsDesktop struct {
	// ProcessRule is how to pick out an application's processes.
	ProcessRule func(ruleContext context.Context, target WindowsQuitTarget) (WindowsProcessRule, error)
	// Windows lists every top level window, front to back as far as
	// Windows lists them that way.
	Windows func() ([]WindowsWindow, error)
	// DescribeProcess is what Windows says of a process.
	DescribeProcess func(processID uint32) WindowsProcess
	// ForegroundProcess is the process of the window in front, or 0 if none
	// is.
	ForegroundProcess func() uint32
	// Restore restores a minimised window.
	Restore func(handle uintptr)
	// BringForward asks Windows to put a window in front, and reports
	// whether Windows agreed; it may still take a moment to get there.
	BringForward func(handle uintptr) bool
}

// applicationWindows is the application's own visible top level windows,
// those on the taskbar, front to back, and a check of whether a process is
// one of the application's, which asks Windows of each process once.
func (desktop WindowsDesktop) applicationWindows(
	rule WindowsProcessRule,
) ([]WindowsWindow, func(processID uint32) bool, error) {
	found, listError := desktop.Windows()
	if listError != nil {
		return nil, nil, listError
	}

	known := map[uint32]bool{}
	isApplications := func(processID uint32) bool {
		matches, seen := known[processID]
		if !seen {
			matches = rule.Matches(desktop.DescribeProcess(processID))
			known[processID] = matches
		}

		return matches
	}

	return taskbarWindows(found, isApplications), isApplications, nil
}

// taskbarWindows is the application's own visible top level windows, those
// on the taskbar, in the order given. Hidden and owned windows go with their
// application.
func taskbarWindows(windows []WindowsWindow, isApplications func(processID uint32) bool) []WindowsWindow {
	var taskbar []WindowsWindow

	for _, window := range windows {
		if window.Visible && !window.Owned && isApplications(window.ProcessID) {
			taskbar = append(taskbar, window)
		}
	}

	return taskbar
}

// WindowsActivator brings applications to the front on Windows.
type WindowsActivator struct {
	desktop WindowsDesktop
	// rules is each identifier's process rule, read once: for a Start menu
	// entry reading it runs PowerShell, which takes the best part of a
	// second.
	rules *sync.Map
}

// NewWindowsActivator returns the Windows ApplicationActivator over a
// desktop.
func NewWindowsActivator(desktop WindowsDesktop) WindowsActivator {
	return WindowsActivator{desktop: desktop, rules: &sync.Map{}}
}

// ActivateApplication brings the application's front window forward,
// restoring it first if it is minimised, and waits until it is in front,
// unless Windows refuses outright. An
// application already in front is left alone; one with no window on the
// taskbar is not running, as far as switching to it goes.
func (activator WindowsActivator) ActivateApplication(activateContext context.Context, identifier string) error {
	rule, ruleError := activator.rule(activateContext, identifier)
	if ruleError != nil {
		return ruleError
	}

	windows, isApplications, listError := activator.desktop.applicationWindows(rule)
	if listError != nil {
		return listError
	}

	if len(windows) == 0 {
		return fmt.Errorf("%w: %s has no window", platform.ErrApplicationNotRunning, identifier)
	}

	inFront := func() FrontState {
		if isApplications(activator.desktop.ForegroundProcess()) {
			return FrontYes
		}

		return FrontNotYet
	}

	if inFront() == FrontYes {
		return nil
	}

	front := windows[0]
	if front.Minimised {
		activator.desktop.Restore(front.Handle)
	}

	if !activator.desktop.BringForward(front.Handle) {
		return fmt.Errorf("%w: Windows would not put %s in front", platform.ErrApplicationNotInFront, identifier)
	}

	return WaitUntilInFront(activateContext, identifier, inFront, windowsFrontWait, windowsFrontCheck)
}

// rule is the identifier's process rule, read once and remembered; a rule
// that could not be read is read again next time.
func (activator WindowsActivator) rule(ruleContext context.Context, identifier string) (WindowsProcessRule, error) {
	remembered, known := activator.rules.Load(identifier)
	if rule, isRule := remembered.(WindowsProcessRule); known && isRule {
		return rule, nil
	}

	rule, ruleError := activator.desktop.ProcessRule(ruleContext, NewWindowsQuitTarget(identifier))
	if ruleError != nil {
		return WindowsProcessRule{}, ruleError
	}

	activator.rules.Store(identifier, rule)

	return rule, nil
}
