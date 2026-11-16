package systemadapter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
		"MSTeams_8wekyb3d8bbwe!MSTeams": {Verb: openVerb, File: `shell:AppsFolder\MSTeams_8wekyb3d8bbwe!MSTeams`},
		thingPath:                       {Verb: openVerb, File: thingPath},
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

// signalName is an application named by name rather than bundle identifier.
const signalName = "Signal"

// TestMacOSQuitTarget_ReadsTheIdentifierAsTheLauncherDoes: the quitter finds
// a bundle identifier by bundle, and a name, with or without .app, by name,
// so a key quits the application it opens.
func TestMacOSQuitTarget_ReadsTheIdentifierAsTheLauncherDoes(t *testing.T) {
	t.Parallel()

	for identifier, want := range map[string]systemadapter.QuitTarget{
		signalBundle:        {Name: signalBundle, ByBundle: true},
		signalName:          {Name: signalName, ByBundle: false},
		signalName + ".app": {Name: signalName, ByBundle: false},
	} {
		if got := systemadapter.MacOSQuitTarget(identifier); got != want {
			t.Errorf("%q: want %+v, got %+v", identifier, want, got)
		}
	}
}

// The Windows applications the quitter tests name.
const (
	thingPath        = `C:\Program Files\Thing\thing.exe`
	signalBundle     = "org.whispersystems.signal-desktop"
	teamsAppID       = "MSTeams_8wekyb3d8bbwe!MSTeams"
	discordDirectory = `C:\Users\Wash\AppData\Local\Discord`
	discordExe       = "Discord.exe"
)

// TestWindowsQuitTarget_ReadsTheIdentifierAsTheLauncherDoes: on Windows the
// quitter reads an identifier as the launcher and the app picker write it: a
// packaged application by its app ID, bare or in the Apps folder; a desktop
// application by its path, under a known folder or not; and any other Apps
// folder entry by its app ID, to be looked up in the Start menu.
func TestWindowsQuitTarget_ReadsTheIdentifierAsTheLauncherDoes(t *testing.T) {
	t.Parallel()

	for identifier, want := range map[string]systemadapter.WindowsQuitTarget{
		teamsAppID: {Kind: systemadapter.WindowsQuitByPackage, Value: teamsAppID, KnownFolder: ""},
		`shell:AppsFolder\` + teamsAppID: {
			Kind: systemadapter.WindowsQuitByPackage, Value: teamsAppID, KnownFolder: "",
		},
		thingPath: {Kind: systemadapter.WindowsQuitByPath, Value: thingPath, KnownFolder: ""},
		`shell:AppsFolder\{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\notepad.exe`: {
			Kind: systemadapter.WindowsQuitByPath, Value: "notepad.exe",
			KnownFolder: "{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}",
		},
		`shell:AppsFolder\` + signalBundle: {Kind: systemadapter.WindowsQuitByAppID, Value: signalBundle, KnownFolder: ""},
	} {
		if got := systemadapter.NewWindowsQuitTarget(identifier); got != want {
			t.Errorf("%q: want %+v, got %+v", identifier, want, got)
		}
	}
}

// TestParseWindowsAppLink_FindsTheProgramAStartMenuEntryRuns: a Start menu
// entry's shortcut, its target and arguments a line each, names the program
// to look for: its target, or for a Squirrel install, whose shortcut runs
// Update.exe --processStart, the program it starts, in Update.exe's folder or
// beneath it. An entry not found is said so.
func TestParseWindowsAppLink_FindsTheProgramAStartMenuEntryRuns(t *testing.T) {
	t.Parallel()

	signal := `C:\Users\Wash\AppData\Local\Programs\signal-desktop\Signal.exe`
	squirrel := systemadapter.WindowsProcessRule{Path: "", Directory: discordDirectory, FileName: discordExe, Package: ""}

	for output, want := range map[string]systemadapter.WindowsProcessRule{
		signal + "\r\n\r\n": {Path: signal, Directory: "", FileName: "", Package: ""},
		discordDirectory + `\Update.exe` + "\r\n--processStart Discord.exe\r\n": squirrel,
		discordDirectory + `\update.exe` + "\r\n" +
			`--processStart "Discord.exe" --process-start-args "--start-minimized"`: squirrel,
	} {
		got, parseError := systemadapter.ParseWindowsAppLink(output)
		if parseError != nil || got != want {
			t.Errorf("%q: want %+v, got %+v (%v)", output, want, got, parseError)
		}
	}

	for _, output := range []string{"", "\r\n", discordDirectory + `\Update.exe` + "\r\n--squirrel-firstrun\r\n"} {
		_, parseError := systemadapter.ParseWindowsAppLink(output)
		if !errors.Is(parseError, platform.ErrApplicationNotFound) {
			t.Errorf("%q: want not found, got %v", output, parseError)
		}
	}
}

// discordProcess is a process whose program is at a path in or beside
// Discord's folder.
func discordProcess(after string) systemadapter.WindowsProcess {
	return systemadapter.WindowsProcess{ImagePath: discordDirectory + after, Package: ""}
}

// TestWindowsProcessRule_MatchesTheApplicationsProcessesOnly: a rule matches
// a process by its program's path, by its program's name in a folder or
// beneath it, or by its package's app ID, ignoring case as Windows does, and
// nothing else.
func TestWindowsProcessRule_MatchesTheApplicationsProcessesOnly(t *testing.T) {
	t.Parallel()

	byPath := systemadapter.WindowsProcessRule{Path: `C:\Apps\Thing.exe`, Directory: "", FileName: "", Package: ""}
	byFolder := systemadapter.WindowsProcessRule{Path: "", Directory: discordDirectory, FileName: discordExe, Package: ""}
	byPackage := systemadapter.WindowsProcessRule{Path: "", Directory: "", FileName: "", Package: teamsAppID}

	for _, check := range []struct {
		rule    systemadapter.WindowsProcessRule
		process systemadapter.WindowsProcess
		want    bool
	}{
		{byPath, systemadapter.WindowsProcess{ImagePath: `c:\apps\thing.EXE`, Package: ""}, true},
		{byPath, systemadapter.WindowsProcess{ImagePath: `C:\Apps\Other.exe`, Package: ""}, false},
		{byFolder, discordProcess(`\app-1.0.9\Discord.exe`), true},
		{byFolder, discordProcess(`\Update.exe`), false},
		{byFolder, systemadapter.WindowsProcess{ImagePath: `C:\Other\Discord.exe`, Package: ""}, false},
		{byFolder, discordProcess(`Canary\Discord.exe`), false},
		{byPackage, systemadapter.WindowsProcess{ImagePath: `C:\WindowsApps\ms-teams.exe`, Package: teamsAppID}, true},
		{byPackage, systemadapter.WindowsProcess{ImagePath: `C:\WindowsApps\ms-teams.exe`, Package: ""}, false},
	} {
		if got := check.rule.Matches(check.process); got != check.want {
			t.Errorf("%+v with %+v: want %v, got %v", check.rule, check.process, check.want, got)
		}
	}
}

// TestWindowsWindowsToClose_ClosesTheApplicationsOwnWindows: the windows
// closed are the application's own visible top level windows, those on the
// taskbar, as clicking each one's close button would; its hidden and owned
// windows, and every other process's, are left alone.
func TestWindowsWindowsToClose_ClosesTheApplicationsOwnWindows(t *testing.T) {
	t.Parallel()

	windows := []systemadapter.WindowsWindow{
		{Handle: 1, ProcessID: 10, Visible: true, Owned: false, Minimised: false},
		{Handle: 2, ProcessID: 10, Visible: false, Owned: false, Minimised: false},
		{Handle: 3, ProcessID: 10, Visible: true, Owned: true, Minimised: false},
		{Handle: 4, ProcessID: 11, Visible: true, Owned: false, Minimised: false},
		{Handle: 5, ProcessID: 12, Visible: true, Owned: false, Minimised: false},
	}

	got := systemadapter.WindowsWindowsToClose(windows, func(processID uint32) bool { return processID != 11 })
	if !slices.Equal(got, []uintptr{1, 5}) {
		t.Errorf("want windows 1 and 5 closed, got %v", got)
	}
}
