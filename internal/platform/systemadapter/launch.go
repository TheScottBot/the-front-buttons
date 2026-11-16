package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The decisions each adapter makes live in this untagged file, so they are
// tested on every machine (datum Part 3: a host tool keeps its decisions in a
// pure module the tests can reach). The build tagged files only supply the real
// command runner and shell call.

// macOSOpenProgram is the system's open command (man open, macOS 27.2).
const macOSOpenProgram = "/usr/bin/open"

// The open command's flags (evaluation 6.1): -u opens a URL with whatever claims
// its scheme, -b names an application by bundle identifier, -a by name.
const (
	openURLFlag              = "-u"
	openBundleIdentifierFlag = "-b"
	openApplicationNameFlag  = "-a"
)

// macOSApplicationSuffix marks an application named by its bundle's file name,
// such as Discord.app, which is a name even though it holds a dot.
const macOSApplicationSuffix = ".app"

// bundleIdentifierPattern is the reverse DNS form of a bundle identifier, such as
// com.microsoft.teams2: two or more dot separated parts with no spaces.
var bundleIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$`)

// windowsOpenVerb is ShellExecuteW's verb for opening an item with its default
// handler.
const windowsOpenVerb = "open"

// windowsAppsFolderPrefix launches a packaged application by its application user
// model ID, the form Microsoft documents for publishing new Teams as a RemoteApp
// (shell:appsFolder\MSTeams_8wekyb3d8bbwe!MSTeams, evaluation 6.6).
const windowsAppsFolderPrefix = `shell:AppsFolder\`

// windowsAppIDSeparator separates a package family name from its application ID
// in an application user model ID; it never appears in a path.
const windowsAppIDSeparator = "!"

// CommandRunner runs a program with arguments, directly and with no shell, so no
// argument is ever parsed as shell syntax.
type CommandRunner func(runContext context.Context, program string, arguments ...string) error

// ShellRequest is one ShellExecuteW call.
type ShellRequest struct {
	Verb string
	File string
}

// ShellExecutor makes one ShellExecuteW call. It never goes through cmd.exe,
// which would read a URL's ampersands as command separators (evaluation 6.1).
type ShellExecutor func(request ShellRequest) error

// MacOSLauncher opens URLs and launches applications with /usr/bin/open.
type MacOSLauncher struct {
	run CommandRunner
}

// NewMacOSURLOpener returns the macOS URLOpener over a command runner.
func NewMacOSURLOpener(run CommandRunner) MacOSLauncher { return MacOSLauncher{run: run} }

// NewMacOSApplicationLauncher returns the macOS ApplicationLauncher over a command
// runner.
func NewMacOSApplicationLauncher(run CommandRunner) MacOSLauncher { return MacOSLauncher{run: run} }

// OpenURL opens a URL with the handler for its scheme.
func (launcher MacOSLauncher) OpenURL(openContext context.Context, address string) error {
	return launcher.open(openContext, address, openURLFlag)
}

// LaunchApplication launches or raises an application by bundle identifier or
// name.
func (launcher MacOSLauncher) LaunchApplication(launchContext context.Context, identifier string) error {
	flag := openApplicationNameFlag
	if isMacOSBundleIdentifier(identifier) {
		flag = openBundleIdentifierFlag
	}

	return launcher.open(launchContext, identifier, flag)
}

// isMacOSBundleIdentifier reports whether a macOS application identifier is a
// bundle identifier rather than a name. The launcher and the icon reader both
// decide by it, so a key's icon is always that of the application it opens.
func isMacOSBundleIdentifier(identifier string) bool {
	return bundleIdentifierPattern.MatchString(identifier) && !strings.HasSuffix(identifier, macOSApplicationSuffix)
}

func (launcher MacOSLauncher) open(runContext context.Context, target string, flag string) error {
	checkError := platform.CheckLaunchTarget(target)
	if checkError != nil {
		return fmt.Errorf("checking the launch target: %w", checkError)
	}

	runError := launcher.run(runContext, macOSOpenProgram, flag, target)
	if runError != nil {
		return fmt.Errorf("%w: open %s: %w", platform.ErrLaunchFailed, flag, runError)
	}

	return nil
}

// WindowsLauncher opens URLs and launches applications with ShellExecuteW,
// and for an application key, switches to the application first when there
// is an activator to switch with.
type WindowsLauncher struct {
	execute  ShellExecutor
	activate func(activateContext context.Context, identifier string) error
}

// NewWindowsURLOpener returns the Windows URLOpener over a shell executor.
func NewWindowsURLOpener(execute ShellExecutor) WindowsLauncher {
	return WindowsLauncher{execute: execute, activate: nil}
}

// NewWindowsApplicationLauncher returns the Windows ApplicationLauncher over a
// shell executor.
func NewWindowsApplicationLauncher(execute ShellExecutor) WindowsLauncher {
	return WindowsLauncher{execute: execute, activate: nil}
}

// NewWindowsApplicationSwitcher returns the Windows ApplicationLauncher that
// switches to a running application, as the open command does on macOS, and
// launches the application only when it is not running.
func NewWindowsApplicationSwitcher(activator platform.ApplicationActivator, execute ShellExecutor) WindowsLauncher {
	return WindowsLauncher{execute: execute, activate: activator.ActivateApplication}
}

// OpenURL opens a URL with its default handler.
func (launcher WindowsLauncher) OpenURL(_ context.Context, address string) error {
	return launcher.open(address)
}

// LaunchApplication switches to the application if it is running, and
// otherwise launches a packaged application by application user model ID
// through the Apps folder, or a desktop application by executable path. An
// application whose processes cannot be found, as one whose Start menu entry
// is not a shortcut, is launched, which for a single instance application
// brings back the one running; one found running that would not come forward
// is reported rather than started a second time, as is one whose press was
// abandoned.
func (launcher WindowsLauncher) LaunchApplication(launchContext context.Context, identifier string) error {
	checkError := platform.CheckLaunchTarget(identifier)
	if checkError != nil {
		return fmt.Errorf("checking the application identifier: %w", checkError)
	}

	if launcher.activate != nil {
		activateError := launcher.activate(launchContext, identifier)

		switch {
		case activateError == nil:
			return nil
		case errors.Is(activateError, platform.ErrApplicationNotInFront), launchContext.Err() != nil:
			return fmt.Errorf("%w: switching to the application: %w", platform.ErrLaunchFailed, activateError)
		}
	}

	target := identifier
	if strings.Contains(identifier, windowsAppIDSeparator) && !strings.ContainsAny(identifier, `\/`) {
		target = windowsAppsFolderPrefix + identifier
	}

	return launcher.open(target)
}

func (launcher WindowsLauncher) open(target string) error {
	checkError := platform.CheckLaunchTarget(target)
	if checkError != nil {
		return fmt.Errorf("checking the launch target: %w", checkError)
	}

	executeError := launcher.execute(ShellRequest{Verb: windowsOpenVerb, File: target})
	if executeError != nil {
		return fmt.Errorf("%w: ShellExecute: %w", platform.ErrLaunchFailed, executeError)
	}

	return nil
}
