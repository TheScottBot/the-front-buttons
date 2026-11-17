package systemadapter

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// hiddenProgram is how the daemon starts a program on Windows, and the only
// way: the repository checks refuse any other in Windows code. The installed
// daemon is a windowed program with no console, so a console program it starts,
// such as powershell.exe, would get a console window of its own that flashes
// on screen for as long as it runs. CREATE_NO_WINDOW starts it with no console
// at all, and HideWindow asks the program to keep any window it opens hidden.
// Its output is still read through pipes, as before.
func hiddenProgram(startContext context.Context, program string, arguments ...string) *exec.Cmd {
	// Every caller passes a fixed program and command, with no part taken from
	// configuration; a value it needs goes in the environment.
	command := exec.CommandContext(startContext, program, arguments...) //nolint:gosec // see above

	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:                 true,
		CmdLine:                    "",
		CreationFlags:              windows.CREATE_NO_WINDOW,
		Token:                      0,
		ProcessAttributes:          nil,
		ThreadAttributes:           nil,
		NoInheritHandles:           false,
		AdditionalInheritedHandles: nil,
		ParentProcess:              0,
	}

	return command
}
