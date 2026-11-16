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
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// switchPath is the program of the application the switching tests switch to.
const switchPath = `C:\Program Files\Thing\thing.exe`

// The processes of the switching tests: the application's two, and another
// application's.
const (
	switchProcess      uint32 = 20
	switchOtherProcess uint32 = 21
	switchSecond       uint32 = 22
)

var errRuleUnread = errors.New("fake Start menu entry unread")

// desktopSpy is a Windows desktop the activator's tests drive instead of the
// real one: it records what was restored and brought forward, and a window
// brought forward takes the foreground only when comesForward says so.
type desktopSpy struct {
	ruleError      error
	ruleReads      int
	windows        []systemadapter.WindowsWindow
	foreground     uint32
	comesForward   bool
	refuses        bool
	restored       []uintptr
	broughtForward []uintptr
}

func (spy *desktopSpy) desktop() systemadapter.WindowsDesktop {
	return systemadapter.WindowsDesktop{
		ProcessRule: func(context.Context, systemadapter.WindowsQuitTarget) (systemadapter.WindowsProcessRule, error) {
			spy.ruleReads++
			if spy.ruleError != nil {
				return systemadapter.WindowsProcessRule{}, spy.ruleError
			}

			return systemadapter.WindowsProcessRule{Path: switchPath, Directory: "", FileName: "", Package: ""}, nil
		},
		Windows: func() ([]systemadapter.WindowsWindow, error) { return spy.windows, nil },
		DescribeProcess: func(processID uint32) systemadapter.WindowsProcess {
			if processID == switchProcess || processID == switchSecond {
				return systemadapter.WindowsProcess{ImagePath: `c:\program files\thing\THING.exe`, Package: ""}
			}

			return systemadapter.WindowsProcess{ImagePath: `C:\Windows\notepad.exe`, Package: ""}
		},
		ForegroundProcess: func() uint32 { return spy.foreground },
		Restore:           func(handle uintptr) { spy.restored = append(spy.restored, handle) },
		BringForward: func(handle uintptr) bool {
			spy.broughtForward = append(spy.broughtForward, handle)

			if spy.refuses || !spy.comesForward {
				return !spy.refuses
			}

			for _, window := range spy.windows {
				if window.Handle == handle {
					spy.foreground = window.ProcessID
				}
			}

			return true
		},
	}
}

// newDesktopSpy is a desktop with windows, front to back, and a process in
// front.
func newDesktopSpy(windows []systemadapter.WindowsWindow, foreground uint32, comesForward bool) *desktopSpy {
	return &desktopSpy{
		ruleError: nil, ruleReads: 0, windows: windows, foreground: foreground,
		comesForward: comesForward, refuses: false, restored: nil, broughtForward: nil,
	}
}

// switchWindow is a top level window of the switching tests.
func switchWindow(handle uintptr, processID uint32, visible, owned, minimised bool) systemadapter.WindowsWindow {
	return systemadapter.WindowsWindow{
		Handle: handle, ProcessID: processID, Visible: visible, Owned: owned, Minimised: minimised,
	}
}

// TestWindowsActivator_BringsTheApplicationsFrontWindowForward: the window
// brought forward is the application's own visible top level window nearest
// the front, as Windows lists them front to back, restored first if it is
// minimised; hidden and owned windows, and other applications', are passed
// over.
func TestWindowsActivator_BringsTheApplicationsFrontWindowForward(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		windows      []systemadapter.WindowsWindow
		wantForward  uintptr
		wantRestored []uintptr
	}{
		"a window showing": {
			windows: []systemadapter.WindowsWindow{
				switchWindow(1, switchOtherProcess, true, false, false),
				switchWindow(2, switchProcess, false, false, false),
				switchWindow(3, switchProcess, true, true, false),
				switchWindow(4, switchSecond, true, false, false),
				switchWindow(5, switchProcess, true, false, false),
			},
			wantForward: 4, wantRestored: nil,
		},
		"a window minimised": {
			windows: []systemadapter.WindowsWindow{
				switchWindow(1, switchOtherProcess, true, false, false),
				switchWindow(6, switchProcess, true, false, true),
			},
			wantForward: 6, wantRestored: []uintptr{6},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spy := newDesktopSpy(testCase.windows, switchOtherProcess, true)

			activateError := systemadapter.NewWindowsActivator(spy.desktop()).ActivateApplication(t.Context(), switchPath)
			if activateError != nil {
				t.Fatalf("want the application brought forward, got %v", activateError)
			}

			if !slices.Equal(spy.broughtForward, []uintptr{testCase.wantForward}) {
				t.Errorf("want window %d brought forward, got %v", testCase.wantForward, spy.broughtForward)
			}

			if !slices.Equal(spy.restored, testCase.wantRestored) {
				t.Errorf("want %v restored, got %v", testCase.wantRestored, spy.restored)
			}
		})
	}
}

// TestWindowsActivator_LeavesAnApplicationInFrontAlone: an application whose
// window is already in front, its dialog included, is neither restored nor
// brought forward again.
func TestWindowsActivator_LeavesAnApplicationInFrontAlone(t *testing.T) {
	t.Parallel()

	spy := newDesktopSpy([]systemadapter.WindowsWindow{switchWindow(1, switchProcess, true, false, true)},
		switchSecond, true)

	activateError := systemadapter.NewWindowsActivator(spy.desktop()).ActivateApplication(t.Context(), switchPath)
	if activateError != nil || len(spy.broughtForward) != 0 || len(spy.restored) != 0 {
		t.Fatalf("want nothing done, got %v forward, %v restored (%v)", spy.broughtForward, spy.restored, activateError)
	}
}

// TestWindowsActivator_ReportsAnApplicationWithNoWindowNotRunning: an
// application with no window on the taskbar is not running, as far as
// switching to it goes, and nothing is brought forward.
func TestWindowsActivator_ReportsAnApplicationWithNoWindowNotRunning(t *testing.T) {
	t.Parallel()

	spy := newDesktopSpy([]systemadapter.WindowsWindow{
		switchWindow(1, switchOtherProcess, true, false, false),
		switchWindow(2, switchProcess, false, false, false),
	}, switchOtherProcess, true)

	activateError := systemadapter.NewWindowsActivator(spy.desktop()).ActivateApplication(t.Context(), switchPath)
	if !errors.Is(activateError, platform.ErrApplicationNotRunning) || len(spy.broughtForward) != 0 {
		t.Fatalf("want not running and nothing brought forward, got %v (%v)", spy.broughtForward, activateError)
	}
}

// TestWindowsActivator_WaitsForTheApplicationToComeForward: a window Windows
// will not put in front is reported as not in front after the wait.
func TestWindowsActivator_WaitsForTheApplicationToComeForward(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		spy := newDesktopSpy([]systemadapter.WindowsWindow{switchWindow(1, switchProcess, true, false, false)},
			switchOtherProcess, false)
		started := time.Now()

		activateError := systemadapter.NewWindowsActivator(spy.desktop()).ActivateApplication(t.Context(), switchPath)
		if !errors.Is(activateError, platform.ErrApplicationNotInFront) {
			t.Fatalf("want not in front, got %v", activateError)
		}

		if waited := time.Since(started); waited < 400*time.Millisecond {
			t.Errorf("want a wait of about half a second, got %s", waited)
		}
	})
}

// TestWindowsActivator_ReportsARefusalAtOnce: a window Windows refuses to
// put in front is reported as not in front without waiting for it.
func TestWindowsActivator_ReportsARefusalAtOnce(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		spy := newDesktopSpy([]systemadapter.WindowsWindow{switchWindow(1, switchProcess, true, false, false)},
			switchOtherProcess, true)
		spy.refuses = true
		started := time.Now()

		activateError := systemadapter.NewWindowsActivator(spy.desktop()).ActivateApplication(t.Context(), switchPath)
		if !errors.Is(activateError, platform.ErrApplicationNotInFront) {
			t.Fatalf("want not in front, got %v", activateError)
		}

		if waited := time.Since(started); waited != 0 {
			t.Errorf("want no wait, got %s", waited)
		}
	})
}

// TestWindowsActivator_ReadsEachApplicationsRuleOnce: how to find an
// application's processes, which for a Start menu entry means running
// PowerShell, is read once and remembered; a rule that could not be read is
// reported and read again next time.
func TestWindowsActivator_ReadsEachApplicationsRuleOnce(t *testing.T) {
	t.Parallel()

	spy := newDesktopSpy([]systemadapter.WindowsWindow{switchWindow(1, switchProcess, true, false, false)},
		switchProcess, true)
	spy.ruleError = errRuleUnread
	activator := systemadapter.NewWindowsActivator(spy.desktop())

	firstError := activator.ActivateApplication(t.Context(), switchPath)
	if !errors.Is(firstError, errRuleUnread) {
		t.Fatalf("want the unread rule reported, got %v", firstError)
	}

	spy.ruleError = nil

	for range 3 {
		activateError := activator.ActivateApplication(t.Context(), switchPath)
		if activateError != nil {
			t.Fatalf("want the application in front, got %v", activateError)
		}
	}

	if spy.ruleReads != 2 {
		t.Errorf("want the rule read twice, once failing, got %d reads", spy.ruleReads)
	}
}

// activatorStub answers every activation with one error.
type activatorStub struct {
	err       error
	activated []string
}

func (stub *activatorStub) ActivateApplication(_ context.Context, identifier string) error {
	stub.activated = append(stub.activated, identifier)

	return stub.err
}

// TestWindowsApplicationSwitcher_SwitchesToARunningApplication_AndLaunchesOtherwise:
// an application key switches to the application when it is running and
// launches it only when it is not, or when whether it is cannot be told; one
// running that would not come forward is reported, not started again, and
// neither is one whose press was abandoned.
func TestWindowsApplicationSwitcher_SwitchesToARunningApplication_AndLaunchesOtherwise(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	cases := map[string]struct {
		pressContext context.Context //nolint:containedctx // each case's press
		activation   error
		wantLaunch   bool
		wantErr      error
	}{
		"running":              {t.Context(), nil, false, nil},
		"not running":          {t.Context(), platform.ErrApplicationNotRunning, true, nil},
		"cannot tell":          {t.Context(), errRuleUnread, true, nil},
		"would not come front": {t.Context(), platform.ErrApplicationNotInFront, false, platform.ErrApplicationNotInFront},
		"abandoned":            {cancelled, context.Canceled, false, context.Canceled},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stub := &activatorStub{err: testCase.activation, activated: nil}
			spy := &shellSpy{requests: nil, failNext: false}

			launchError := systemadapter.NewWindowsApplicationSwitcher(stub, spy.execute).
				LaunchApplication(testCase.pressContext, switchPath)

			if !errors.Is(launchError, testCase.wantErr) || (testCase.wantErr == nil && launchError != nil) {
				t.Fatalf("want %v, got %v", testCase.wantErr, launchError)
			}

			if !slices.Equal(stub.activated, []string{switchPath}) {
				t.Errorf("want one switch to %s, got %v", switchPath, stub.activated)
			}

			if launched := len(spy.requests) == 1; launched != testCase.wantLaunch || len(spy.requests) > 1 {
				t.Errorf("want launched %t, got %v", testCase.wantLaunch, spy.requests)
			}
		})
	}
}

// TestWindowsApplicationSwitcher_RefusesABadTargetBeforeSwitching: an
// identifier the launch target rule refuses is neither switched to nor
// launched.
func TestWindowsApplicationSwitcher_RefusesABadTargetBeforeSwitching(t *testing.T) {
	t.Parallel()

	stub := &activatorStub{err: nil, activated: nil}
	spy := &shellSpy{requests: nil, failNext: false}

	launchError := systemadapter.NewWindowsApplicationSwitcher(stub, spy.execute).LaunchApplication(t.Context(), "")
	if launchError == nil || len(stub.activated) != 0 || len(spy.requests) != 0 {
		t.Fatalf("want refused untouched, got %v switched, %v launched (%v)", stub.activated, spy.requests, launchError)
	}
}

func TestWindowsApplicationSwitcher_PassesTheApplicationLauncherContract(t *testing.T) {
	t.Parallel()

	testsupport.RunApplicationLauncherContract(t, "MSTeams_8wekyb3d8bbwe!MSTeams",
		func(*testing.T) (platform.ApplicationLauncher, func()) {
			stub := &activatorStub{err: platform.ErrApplicationNotRunning, activated: nil}
			spy := &shellSpy{requests: nil, failNext: false}

			return systemadapter.NewWindowsApplicationSwitcher(stub, spy.execute), func() { spy.failNext = true }
		})
}
