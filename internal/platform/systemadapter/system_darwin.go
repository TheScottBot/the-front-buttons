package systemadapter

import (
	"context"
	"fmt"
	"os/exec"
)

// NewURLOpener returns this machine's URLOpener: /usr/bin/open.
func NewURLOpener() MacOSLauncher { return NewMacOSURLOpener(runCommand) }

// NewApplicationLauncher returns this machine's ApplicationLauncher:
// /usr/bin/open.
func NewApplicationLauncher() MacOSLauncher { return NewMacOSApplicationLauncher(runCommand) }

// runCommand runs a program directly, never through a shell. Its output is
// discarded unread, because open's messages can echo the URL, which logs must
// never hold (specification section 7).
func runCommand(runContext context.Context, program string, arguments ...string) error {
	// The program is the fixed open command from launch.go, and the arguments
	// go straight to it with no shell, so no argument is parsed as syntax.
	runError := exec.CommandContext(runContext, program, arguments...).Run() //nolint:gosec // see above
	if runError != nil {
		return fmt.Errorf("running %s: %w", program, runError)
	}

	return nil
}
