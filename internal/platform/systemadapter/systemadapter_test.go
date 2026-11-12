package systemadapter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

var errProcessFailed = errors.New("fake process exited with status 1")

// openVerb is ShellExecuteW's verb the Windows adapter is expected to use.
const openVerb = "open"

// commandSpy records commands instead of running them, so the macOS adapters'
// tests run nothing real on any machine.
type commandSpy struct {
	programs  []string
	arguments [][]string
	failNext  bool
}

func (spy *commandSpy) run(_ context.Context, program string, arguments ...string) error {
	if spy.failNext {
		spy.failNext = false

		return errProcessFailed
	}

	spy.programs = append(spy.programs, program)
	spy.arguments = append(spy.arguments, arguments)

	return nil
}

// shellSpy records ShellExecute requests instead of making them, so the Windows
// adapters' tests run on any machine and launch nothing.
type shellSpy struct {
	requests []systemadapter.ShellRequest
	failNext bool
}

func (spy *shellSpy) execute(request systemadapter.ShellRequest) error {
	if spy.failNext {
		spy.failNext = false

		return errProcessFailed
	}

	spy.requests = append(spy.requests, request)

	return nil
}

func TestMacOSURLOpener_PassesTheURLOpenerContract(t *testing.T) {
	t.Parallel()

	testsupport.RunURLOpenerContract(t, func(*testing.T) (platform.URLOpener, func()) {
		spy := &commandSpy{programs: nil, arguments: nil, failNext: false}

		return systemadapter.NewMacOSURLOpener(spy.run), func() { spy.failNext = true }
	})
}

func TestMacOSApplicationLauncher_PassesTheApplicationLauncherContract(t *testing.T) {
	t.Parallel()

	testsupport.RunApplicationLauncherContract(t, "com.apple.calculator",
		func(*testing.T) (platform.ApplicationLauncher, func()) {
			spy := &commandSpy{programs: nil, arguments: nil, failNext: false}

			return systemadapter.NewMacOSApplicationLauncher(spy.run), func() { spy.failNext = true }
		})
}

func TestWindowsURLOpener_PassesTheURLOpenerContract(t *testing.T) {
	t.Parallel()

	testsupport.RunURLOpenerContract(t, func(*testing.T) (platform.URLOpener, func()) {
		spy := &shellSpy{requests: nil, failNext: false}

		return systemadapter.NewWindowsURLOpener(spy.execute), func() { spy.failNext = true }
	})
}

func TestWindowsApplicationLauncher_PassesTheApplicationLauncherContract(t *testing.T) {
	t.Parallel()

	testsupport.RunApplicationLauncherContract(t, "MSTeams_8wekyb3d8bbwe!MSTeams",
		func(*testing.T) (platform.ApplicationLauncher, func()) {
			spy := &shellSpy{requests: nil, failNext: false}

			return systemadapter.NewWindowsApplicationLauncher(spy.execute), func() { spy.failNext = true }
		})
}

func TestURLOpenerTestDouble_PassesTheURLOpenerContract(t *testing.T) {
	t.Parallel()

	testsupport.RunURLOpenerContract(t, func(*testing.T) (platform.URLOpener, func()) {
		recorder := testsupport.NewLaunchRecorder()

		return testsupport.URLOpenerTestDouble{Recorder: recorder}, recorder.FailNext
	})
}

func TestApplicationLauncherTestDouble_PassesTheApplicationLauncherContract(t *testing.T) {
	t.Parallel()

	testsupport.RunApplicationLauncherContract(t, "com.apple.calculator",
		func(*testing.T) (platform.ApplicationLauncher, func()) {
			recorder := testsupport.NewLaunchRecorder()

			return testsupport.ApplicationLauncherTestDouble{Recorder: recorder}, recorder.FailNext
		})
}

func TestMacOSURLOpener_RunsOpenWithTheURLAsOneArgument_AndNoShell(t *testing.T) {
	t.Parallel()

	spy := &commandSpy{programs: nil, arguments: nil, failNext: false}
	address := "https://teams.microsoft.com/l/chat/0/0?users=a@example.test,b@example.test&topicName=x y"

	openError := systemadapter.NewMacOSURLOpener(spy.run).OpenURL(t.Context(), address)
	if openError != nil {
		t.Fatalf("opening: %v", openError)
	}

	if spy.programs[0] != "/usr/bin/open" || !reflect.DeepEqual(spy.arguments[0], []string{"-u", address}) {
		t.Fatalf("want /usr/bin/open -u with the URL as one argument, got %s %q", spy.programs[0], spy.arguments[0])
	}
}

func TestMacOSApplicationLauncher_ChoosesBundleIdentifierOrName(t *testing.T) {
	t.Parallel()

	launches := map[string][]string{
		"com.microsoft.teams2": {"-b", "com.microsoft.teams2"},
		"Visual Studio Code":   {"-a", "Visual Studio Code"},
		"Calculator":           {"-a", "Calculator"},
		"Discord.app":          {"-a", "Discord.app"},
	}

	for identifier, wantArguments := range launches {
		spy := &commandSpy{programs: nil, arguments: nil, failNext: false}

		launchError := systemadapter.NewMacOSApplicationLauncher(spy.run).LaunchApplication(t.Context(), identifier)
		if launchError != nil || !reflect.DeepEqual(spy.arguments[0], wantArguments) {
			t.Errorf("%q: want %q, got %q (%v)", identifier, wantArguments, spy.arguments, launchError)
		}
	}
}

func TestWindowsURLOpener_AsksTheShellToOpenTheURL(t *testing.T) {
	t.Parallel()

	spy := &shellSpy{requests: nil, failNext: false}
	address := "https://discord.com/channels/1/2?x=1&y=2"

	openError := systemadapter.NewWindowsURLOpener(spy.execute).OpenURL(t.Context(), address)

	want := systemadapter.ShellRequest{Verb: openVerb, File: address}
	if openError != nil || len(spy.requests) != 1 || spy.requests[0] != want {
		t.Fatalf("want %+v, got %+v (%v)", want, spy.requests, openError)
	}
}

func TestWindowsApplicationLauncher_UsesAppsFolderForAppIDs_AndPathsDirectly(t *testing.T) {
	t.Parallel()

	launches := map[string]systemadapter.ShellRequest{
		"MSTeams_8wekyb3d8bbwe!MSTeams":    {Verb: openVerb, File: `shell:AppsFolder\MSTeams_8wekyb3d8bbwe!MSTeams`},
		`C:\Program Files\Thing\thing.exe`: {Verb: openVerb, File: `C:\Program Files\Thing\thing.exe`},
	}

	for identifier, want := range launches {
		spy := &shellSpy{requests: nil, failNext: false}

		launchError := systemadapter.NewWindowsApplicationLauncher(spy.execute).LaunchApplication(t.Context(), identifier)
		if launchError != nil || len(spy.requests) != 1 || spy.requests[0] != want {
			t.Errorf("%q: want %+v, got %+v (%v)", identifier, want, spy.requests, launchError)
		}
	}
}

func TestUserConfigLocation_PutsConfigAndStateInTheProductFolder(t *testing.T) {
	t.Parallel()

	location := systemadapter.UserConfigLocation{}

	configFile, configError := location.ConfigFile()
	stateDirectory, stateError := location.StateDirectory()

	if configError != nil || stateError != nil {
		t.Fatalf("locating: %v %v", configError, stateError)
	}

	if !filepath.IsAbs(configFile) || filepath.Base(configFile) != "config.yaml" ||
		filepath.Base(filepath.Dir(configFile)) != platform.ProductIdentifier {
		t.Fatalf("want an absolute .../%s/config.yaml, got %s", platform.ProductIdentifier, configFile)
	}

	if filepath.Dir(stateDirectory) != filepath.Dir(configFile) || filepath.Base(stateDirectory) != "state" {
		t.Fatalf("want the state directory beside the config file, got %s", stateDirectory)
	}
}

func TestShutdownSignals_IncludeInterruptAndTerminate(t *testing.T) {
	t.Parallel()

	signals := systemadapter.ShutdownSignals()

	if len(signals) != 2 || signals[0] != os.Interrupt || signals[1].String() != "terminated" {
		t.Fatalf("want interrupt and terminate, got %v", signals)
	}
}
