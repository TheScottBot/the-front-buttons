package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The ApplicationQuitter on Windows closes an application's own windows, as
// clicking each one's close button does (quit.go says why that is as far as
// Windows goes, and decides which windows). Not yet run on a Windows machine:
// the author is testing on macOS only for now.

// The PowerShell command that reads a Start menu entry's shortcut: its target
// and its arguments, a line each. The entry's app ID comes in an environment
// variable, never in the command, so nothing a configuration holds is ever
// read as PowerShell.
const (
	windowsAppIDVariable  = "THE_FRONT_BUTTONS_APP_ID"
	windowsAppLinkCommand = "$id = $env:" + windowsAppIDVariable + "; " +
		"$item = (New-Object -ComObject Shell.Application).NameSpace('shell:AppsFolder').Items() | " +
		"Where-Object { $_.Path -eq $id } | Select-Object -First 1; " +
		"if ($item) { $item.ExtendedProperty('System.Link.TargetParsingPath'); " +
		"$item.ExtendedProperty('System.Link.Arguments') }"
)

// The Win32 values the quitter uses, from WinUser.h and AppModel.h: the
// message a window's close button sends, GetWindow's request for a window's
// owner, and room for the longest program path and application user model
// ID.
const (
	windowMessageClose           = 0x0010
	getWindowOwner               = 4
	longestPath                  = 32768
	longestApplicationUserModeID = 130
)

//nolint:gochecknoglobals // the lazily loaded system calls Go's Windows package lacks
var (
	procPostMessage               = windows.NewLazySystemDLL("user32.dll").NewProc("PostMessageW")
	procGetWindow                 = windows.NewLazySystemDLL("user32.dll").NewProc("GetWindow")
	procIsIconic                  = windows.NewLazySystemDLL("user32.dll").NewProc("IsIconic")
	procGetApplicationUserModelID = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetApplicationUserModelId")
)

// errWindowNotClosed reports a window Windows would not post the close
// message to.
var errWindowNotClosed = errors.New("window not asked to close")

// ApplicationQuitter is the ApplicationQuitter on Windows.
type ApplicationQuitter struct{}

// NewApplicationQuitter returns this machine's ApplicationQuitter. The bridge
// is ignored.
func NewApplicationQuitter(_ Bridge) ApplicationQuitter { return ApplicationQuitter{} }

// QuitApplication closes the application's own windows. An application not
// running has none, and is left alone.
func (ApplicationQuitter) QuitApplication(quitContext context.Context, identifier string) error {
	rule, ruleError := windowsProcessRule(quitContext, NewWindowsQuitTarget(identifier))
	if ruleError != nil {
		return ruleError
	}

	found, _, listError := systemDesktop().applicationWindows(rule)
	if listError != nil {
		return listError
	}

	for _, window := range found {
		posted, _, postError := procPostMessage.Call(window.Handle, windowMessageClose, 0, 0)
		if posted == 0 {
			return fmt.Errorf("%w: %w", errWindowNotClosed, postError)
		}
	}

	return nil
}

// windowsProcessRule is how to pick out the application's processes: by its
// package's app ID, by its program's path, with a known folder's path put in
// front, or by the program its Start menu shortcut runs.
func windowsProcessRule(ruleContext context.Context, target WindowsQuitTarget) (WindowsProcessRule, error) {
	switch target.Kind {
	case WindowsQuitByPackage:
		return WindowsProcessRule{Path: "", Directory: "", FileName: "", Package: target.Value}, nil
	case WindowsQuitByAppID:
		command := hiddenProgram(ruleContext, windowsPowerShellProgram,
			"-NoProfile", "-NonInteractive", "-Command", windowsAppLinkCommand)

		command.Env = append(os.Environ(), windowsAppIDVariable+"="+target.Value)

		output, runError := command.Output()
		if runError != nil {
			return WindowsProcessRule{}, fmt.Errorf("reading the Start menu entry: %w", runError)
		}

		return ParseWindowsAppLink(string(output))
	case WindowsQuitByPath:
	}

	path := target.Value

	if target.KnownFolder != "" {
		folder, folderError := knownFolderPath(target.KnownFolder)
		if folderError != nil {
			return WindowsProcessRule{}, folderError
		}

		path = folder + `\` + path
	}

	return WindowsProcessRule{Path: path, Directory: "", FileName: "", Package: ""}, nil
}

// knownFolderPath is the path of the known folder whose ID is given, such as
// {1AC14E77-02E7-4E5D-B744-2EB1AE5198B7} for System32.
func knownFolderPath(folderID string) (string, error) {
	parsed, guidError := windows.GUIDFromString(folderID)
	if guidError != nil {
		return "", fmt.Errorf("reading a known folder's ID: %w", guidError)
	}

	folder, folderError := windows.KnownFolderPath((*windows.KNOWNFOLDERID)(&parsed), 0)
	if folderError != nil {
		return "", fmt.Errorf("finding a known folder: %w", folderError)
	}

	return folder, nil
}

// The windows EnumWindows lists, collected by one callback made once, as
// Windows allows a process only so many; the mutex keeps one listing at a
// time.
//
//nolint:gochecknoglobals // the one callback EnumWindows calls, and what it collects
var (
	windowListMutex    sync.Mutex
	windowList         []WindowsWindow
	windowListCallback = sync.OnceValue(func() uintptr { return windows.NewCallback(collectWindow) })
)

// collectWindow notes one top level window and asks for the next.
func collectWindow(handle windows.HWND, _ uintptr) uintptr {
	var processID uint32

	_, _ = windows.GetWindowThreadProcessId(handle, &processID)
	owner, _, _ := procGetWindow.Call(uintptr(handle), getWindowOwner)
	minimised, _, _ := procIsIconic.Call(uintptr(handle))

	windowList = append(windowList, WindowsWindow{
		Handle: uintptr(handle), ProcessID: processID, Visible: windows.IsWindowVisible(handle), Owned: owner != 0,
		Minimised: minimised != 0,
	})

	return 1
}

// topLevelWindows lists every top level window in the order EnumWindows
// gives them, which is front to back in practice, though Microsoft does not
// promise it.
func topLevelWindows() ([]WindowsWindow, error) {
	windowListMutex.Lock()
	defer windowListMutex.Unlock()

	windowList = nil

	enumerateError := windows.EnumWindows(windowListCallback(), nil)
	if enumerateError != nil {
		return nil, fmt.Errorf("listing windows: %w", enumerateError)
	}

	return windowList, nil
}

// describeProcess is a process's program path and, for a packaged
// application, its app ID; either is empty where Windows will not say.
func describeProcess(processID uint32) WindowsProcess {
	described := WindowsProcess{ImagePath: "", Package: ""}

	handle, openError := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, processID)
	if openError != nil {
		return described
	}

	defer func() { _ = windows.CloseHandle(handle) }()

	path := make([]uint16, longestPath)
	pathLength := uint32(longestPath)

	if windows.QueryFullProcessImageName(handle, 0, &path[0], &pathLength) == nil {
		described.ImagePath = windows.UTF16ToString(path[:pathLength])
	}

	appID := make([]uint16, longestApplicationUserModeID)
	appIDLength := uint32(longestApplicationUserModeID)

	// GetApplicationUserModelId returns ERROR_SUCCESS, nothing, for a
	// packaged process, and an error for any other. It takes pointers to the
	// length and the buffer; unsafe is the only way to pass them.
	status, _, _ := procGetApplicationUserModelID.Call(uintptr(handle),
		uintptr(unsafe.Pointer(&appIDLength)), uintptr(unsafe.Pointer(&appID[0]))) //nolint:gosec // see above
	if status == 0 {
		described.Package = windows.UTF16ToString(appID)
	}

	return described
}
