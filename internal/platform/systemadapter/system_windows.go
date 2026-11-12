package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The Start menu listing, as PowerShell gives it: every entry's name and AppID,
// written as JSON. Windows PowerShell 5.1 ships with every Windows 10 and 11.
const (
	windowsPowerShellProgram = "powershell.exe"
	windowsStartAppsCommand  = "Get-StartApps | Select-Object Name, AppID | ConvertTo-Json -Compress"
)

// NewURLOpener returns this machine's URLOpener: ShellExecuteW.
func NewURLOpener() WindowsLauncher { return NewWindowsURLOpener(shellExecute) }

// NewApplicationLauncher returns this machine's ApplicationLauncher:
// ShellExecuteW.
func NewApplicationLauncher() WindowsLauncher { return NewWindowsApplicationLauncher(shellExecute) }

// shellExecute makes one ShellExecuteW call on a thread with COM initialised, as
// Microsoft advises before every call (evaluation 6.1). COM state belongs to a
// thread, so the goroutine is pinned to its thread for the call.
func shellExecute(request ShellRequest) error {
	runtime.LockOSThread()

	defer runtime.UnlockOSThread()

	comError := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE)

	// S_FALSE means COM was already initialised on this thread, which still
	// needs a matching uninitialise; any other failure means it is initialised
	// in another mode, which ShellExecuteW tolerates, and must not be undone.
	if comError == nil || errors.Is(comError, windows.Errno(windows.S_FALSE)) {
		defer windows.CoUninitialize()
	}

	verb, verbError := windows.UTF16PtrFromString(request.Verb)
	file, fileError := windows.UTF16PtrFromString(request.File)

	conversionError := errors.Join(verbError, fileError)
	if conversionError != nil {
		return fmt.Errorf("converting a shell request: %w", conversionError)
	}

	executeError := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
	if executeError != nil {
		return fmt.Errorf("ShellExecute: %w", executeError)
	}

	return nil
}

// NewApplicationCatalogue lists the Start menu's applications with
// Get-StartApps, each as a shell:AppsFolder target the launcher opens. Not yet
// run on a Windows machine: the author is testing on macOS only for now.
func NewApplicationCatalogue() ApplicationLister {
	return ApplicationLister{list: func(listContext context.Context) ([]platform.Application, error) {
		// A fixed program and a fixed command, with no part taken from
		// configuration.
		output, runError := exec.CommandContext(listContext, windowsPowerShellProgram,
			"-NoProfile", "-NonInteractive", "-Command", windowsStartAppsCommand).Output()
		if runError != nil {
			return nil, fmt.Errorf("listing Start menu applications: %w", runError)
		}

		applications, parseError := ParseStartApps(output)
		if parseError != nil {
			return nil, parseError
		}

		return nonEmpty(applications)
	}}
}
