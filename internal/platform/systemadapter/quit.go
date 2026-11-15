package systemadapter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// QuitTarget is how the macOS quitter finds an application among those
// running: by bundle identifier, or else by name.
type QuitTarget struct {
	Name     string
	ByBundle bool
}

// MacOSQuitTarget reads an identifier as the launcher reads it: a bundle
// identifier by bundle, or else a name, Signal and Signal.app alike, as the
// open command takes it.
func MacOSQuitTarget(identifier string) QuitTarget {
	if isMacOSBundleIdentifier(identifier) {
		return QuitTarget{Name: identifier, ByBundle: true}
	}

	return QuitTarget{Name: strings.TrimSuffix(identifier, macOSApplicationSuffix), ByBundle: false}
}

// On Windows there is no one way to ask an application to quit, as macOS
// has. The quitter closes the application's own windows, as clicking each
// one's close button does, which quits most applications; one that keeps
// running in the notification area when its window closes, as Signal and
// Discord may, keeps running. It finds the application's processes by what
// its identifier names, as the launcher and the app picker write it.

// WindowsQuitKind is how the Windows quitter finds an application's
// processes.
type WindowsQuitKind int

// The ways an identifier names an application on Windows.
const (
	// WindowsQuitByPackage: a packaged application, by its app ID.
	WindowsQuitByPackage WindowsQuitKind = iota
	// WindowsQuitByPath: a desktop application, by its program's path.
	WindowsQuitByPath
	// WindowsQuitByAppID: any other Start menu entry, by its app ID, which
	// the Start menu's shortcut turns into a program.
	WindowsQuitByAppID
)

// WindowsQuitTarget is what an identifier names on Windows: how to find it,
// the app ID or path, and for a path under a known folder, such as
// {1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\notepad.exe, that folder's ID.
type WindowsQuitTarget struct {
	Kind        WindowsQuitKind
	Value       string
	KnownFolder string
}

// NewWindowsQuitTarget reads an identifier as the launcher reads it, with or
// without the Apps folder's prefix the app picker gives it.
func NewWindowsQuitTarget(identifier string) WindowsQuitTarget {
	within := strings.TrimPrefix(identifier, windowsAppsFolderPrefix)
	inFolder := within != identifier

	switch {
	case !strings.ContainsAny(within, `\/`) && strings.Contains(within, windowsAppIDSeparator):
		return WindowsQuitTarget{Kind: WindowsQuitByPackage, Value: within, KnownFolder: ""}
	case strings.HasPrefix(within, "{") && strings.Contains(within, `}\`):
		folder, rest, _ := strings.Cut(within, `\`)

		return WindowsQuitTarget{Kind: WindowsQuitByPath, Value: rest, KnownFolder: folder}
	case inFolder && !strings.ContainsAny(within, `\/`):
		return WindowsQuitTarget{Kind: WindowsQuitByAppID, Value: within, KnownFolder: ""}
	default:
		return WindowsQuitTarget{Kind: WindowsQuitByPath, Value: within, KnownFolder: ""}
	}
}

// squirrelUpdater is the program a Squirrel install's shortcut runs, with
// squirrelStartFlag naming the application's own program, which it starts
// from a versioned folder beside it.
const (
	squirrelUpdater   = "Update.exe"
	squirrelStartFlag = "--processStart"
)

// WindowsProcessRule picks out an application's processes: by its program's
// Path; by its program's FileName in Directory or beneath it; or by its
// Package's app ID.
type WindowsProcessRule struct {
	Path      string
	Directory string
	FileName  string
	Package   string
}

// WindowsProcess is what the quitter knows of a running process: its
// program's path and, for a packaged application, its app ID.
type WindowsProcess struct {
	ImagePath string
	Package   string
}

// Matches reports whether a process is one of the application's, ignoring
// case, as Windows does in paths and app IDs.
func (rule WindowsProcessRule) Matches(process WindowsProcess) bool {
	switch {
	case rule.Package != "":
		return strings.EqualFold(process.Package, rule.Package)
	case rule.Path != "":
		return strings.EqualFold(process.ImagePath, rule.Path)
	case rule.Directory != "":
		folder := strings.TrimSuffix(rule.Directory, `\`) + `\`
		_, fileName, _ := strings.CutLast(process.ImagePath, `\`)

		return len(process.ImagePath) > len(folder) && strings.EqualFold(process.ImagePath[:len(folder)], folder) &&
			strings.EqualFold(fileName, rule.FileName)
	default:
		return false
	}
}

// ParseWindowsAppLink reads a Start menu entry's shortcut, its target and its
// arguments a line each, as the program to look for: its target, or for a
// Squirrel install, the program Update.exe is told to start, in its folder
// or beneath it.
func ParseWindowsAppLink(output string) (WindowsProcessRule, error) {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	target := strings.TrimSpace(lines[0])
	arguments := ""

	if len(lines) > 1 {
		arguments = strings.TrimSpace(lines[1])
	}

	if target == "" {
		return WindowsProcessRule{}, fmt.Errorf("%w: no Start menu entry", platform.ErrApplicationNotFound)
	}

	directory, program, _ := strings.CutLast(target, `\`)
	if !strings.EqualFold(program, squirrelUpdater) {
		return WindowsProcessRule{Path: target, Directory: "", FileName: "", Package: ""}, nil
	}

	fields := strings.Fields(arguments)
	flagAt := slices.Index(fields, squirrelStartFlag)

	if flagAt < 0 || flagAt+1 >= len(fields) {
		return WindowsProcessRule{}, fmt.Errorf("%w: a Squirrel shortcut that starts nothing",
			platform.ErrApplicationNotFound)
	}

	program = strings.Trim(fields[flagAt+1], `"`)

	return WindowsProcessRule{Path: "", Directory: directory, FileName: program, Package: ""}, nil
}

// WindowsWindow is what the quitter knows of a top level window: its handle,
// its process, whether it is showing, and whether another window owns it, as
// a dialog is owned.
type WindowsWindow struct {
	Handle    uintptr
	ProcessID uint32
	Visible   bool
	Owned     bool
}

// WindowsWindowsToClose is the application's windows to close: its own
// visible top level windows, those on the taskbar, as a person would close
// them. Hidden and owned windows go with their application.
func WindowsWindowsToClose(windows []WindowsWindow, isApplications func(processID uint32) bool) []uintptr {
	var handles []uintptr

	for _, window := range windows {
		if window.Visible && !window.Owned && isApplications(window.ProcessID) {
			handles = append(handles, window.Handle)
		}
	}

	return handles
}
