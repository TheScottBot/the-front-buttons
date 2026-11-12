package systemadapter

import (
	"errors"
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
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

	if conversionError := errors.Join(verbError, fileError); conversionError != nil {
		return fmt.Errorf("converting a shell request: %w", conversionError)
	}

	executeError := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
	if executeError != nil {
		return fmt.Errorf("ShellExecute: %w", executeError)
	}

	return nil
}
