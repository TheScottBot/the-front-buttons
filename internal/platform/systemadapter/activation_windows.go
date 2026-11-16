package systemadapter

import (
	"golang.org/x/sys/windows"
)

// The ApplicationActivator on Windows: switching.go decides which window to
// bring forward and when the application is in front; these are the calls.
//
// Windows lets a process put a window in front only in the cases
// SetForegroundWindow's documentation lists, and the daemon, a background
// process with no window of its own, meets none of them on its own: Windows
// then flashes the window's taskbar button instead. One listed case is that
// the calling process received the last input event, so the activator sends
// the Alt key down with SendInput, the call the key presser already makes,
// before asking, and lets it up afterwards. That was chosen over
// AttachThreadInput to the foreground window's thread, the other documented
// way, because attaching joins the daemon's input to whatever application is
// in front, and a hung application in front would then hang the daemon too.
// Letting Alt up only once the window is in front keeps both halves from
// reaching one window as a lone Alt press, which would open its menu.

// virtualKeyAlt is VK_MENU, the Alt key, from WinUser.h.
const virtualKeyAlt = 0x12

//nolint:gochecknoglobals // the lazily loaded system calls Go's Windows package lacks
var (
	procShowWindow          = windows.NewLazySystemDLL("user32.dll").NewProc("ShowWindow")
	procSetForegroundWindow = windows.NewLazySystemDLL("user32.dll").NewProc("SetForegroundWindow")
	procGetLastActivePopup  = windows.NewLazySystemDLL("user32.dll").NewProc("GetLastActivePopup")
)

// NewApplicationActivator returns this machine's ApplicationActivator. The
// bridge is ignored.
func NewApplicationActivator(_ Bridge) WindowsActivator { return NewWindowsActivator(systemDesktop()) }

// systemDesktop is Windows' own desktop.
func systemDesktop() WindowsDesktop {
	return WindowsDesktop{
		ProcessRule:       windowsProcessRule,
		Windows:           topLevelWindows,
		DescribeProcess:   describeProcess,
		ForegroundProcess: foregroundProcess,
		Restore:           restoreWindow,
		BringForward:      bringWindowForward,
	}
}

// foregroundProcess is the process of the window in front, or 0 if none is,
// as for a moment while windows change places.
func foregroundProcess() uint32 {
	front := windows.GetForegroundWindow()
	if front == 0 {
		return 0
	}

	var processID uint32

	_, _ = windows.GetWindowThreadProcessId(front, &processID)

	return processID
}

// restoreWindow restores a minimised window to its size and place before.
func restoreWindow(handle uintptr) {
	// ShowWindow's result says only whether the window was showing before.
	wasShowing, _, _ := procShowWindow.Call(handle, windows.SW_RESTORE)
	_ = wasShowing
}

// bringWindowForward puts a window in front, or the window it last had open
// over it, such as a dialog, as Alt and Tab does, with Alt held so Windows
// lets it (see above). It reports whether Windows agreed.
func bringWindowForward(handle uintptr) bool {
	target, _, _ := procGetLastActivePopup.Call(handle)
	if target == 0 {
		target = handle
	}

	_ = sendKeyboardInputs([]keyboardInput{
		{inputType: inputKeyboard, virtualKey: virtualKeyAlt, scanCode: 0, flags: 0, time: 0, extraInfo: 0},
	})

	defer func() {
		_ = sendKeyboardInputs([]keyboardInput{
			{inputType: inputKeyboard, virtualKey: virtualKeyAlt, scanCode: 0, flags: keyEventKeyUp, time: 0, extraInfo: 0},
		})
	}()

	brought, _, _ := procSetForegroundWindow.Call(target)

	return brought != 0
}
