package systemadapter //nolint:testpackage // hiddenProgram is unexported

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// noConsoleVariable makes the test binary run the body of
// TestHiddenProgram_StartsWithNoConsole as the child it starts.
const noConsoleVariable = "THE_FRONT_BUTTONS_HIDDEN_PROGRAM_CHILD"

// childHasConsoleExit is the child's exit code when it finds a console window.
const childHasConsoleExit = 3

func TestHiddenProgram_AsksForNoWindow(t *testing.T) {
	t.Parallel()

	command := hiddenProgram(t.Context(), windowsPowerShellProgram, "-NoProfile")

	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow ||
		command.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("want HideWindow and CREATE_NO_WINDOW, got %+v", command.SysProcAttr)
	}
}

// TestHiddenProgram_StartsWithNoConsole starts this test binary through
// hiddenProgram and has it look for a console window of its own: a program
// started with one would have flashed it on screen.
func TestHiddenProgram_StartsWithNoConsole(t *testing.T) {
	t.Parallel()

	if os.Getenv(noConsoleVariable) == "1" {
		consoleWindow, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
		if consoleWindow != 0 {
			os.Exit(childHasConsoleExit)
		}

		return
	}

	child := hiddenProgram(t.Context(), os.Args[0], "-test.run=^TestHiddenProgram_StartsWithNoConsole$", "-test.count=1")

	child.Env = append(os.Environ(), noConsoleVariable+"=1")

	output, runError := child.CombinedOutput()
	if runError != nil {
		t.Fatalf("want the child to find no console window, got %v\n%s", runError, output)
	}
}

// TestHiddenProgram_StillHandsBackOutput wants a hidden program's output read
// as before, since both PowerShell calls parse what it writes.
func TestHiddenProgram_StillHandsBackOutput(t *testing.T) {
	t.Parallel()

	output, runError := hiddenProgram(t.Context(), windowsPowerShellProgram,
		"-NoProfile", "-NonInteractive", "-Command", "Write-Output hidden").Output()
	if runError != nil {
		t.Fatal(runError)
	}

	if strings.TrimSpace(string(output)) != "hidden" {
		t.Errorf("want the program's output, got %q", output)
	}
}
