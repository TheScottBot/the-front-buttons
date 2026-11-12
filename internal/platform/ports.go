package platform

import (
	"context"
	"errors"
)

// ErrLaunchRefused reports a URL or application identifier refused before
// anything was run: empty, too long, starting with a hyphen (which a command line
// tool would read as an option), or holding a control character.
var ErrLaunchRefused = errors.New("launch target refused")

// ErrLaunchFailed reports that the operating system could not open a URL or
// launch an application. Its reason category is shown on the key.
var ErrLaunchFailed = errors.New("launch failed")

// URLOpener opens a URL or deep link with the system's handler for it.
type URLOpener interface {
	OpenURL(openContext context.Context, address string) error
}

// ApplicationLauncher launches an application, or raises it if it is running.
// The identifier is interpreted by the operating system's adapter: a bundle
// identifier or name on macOS, an executable path or application user model ID on
// Windows (specification section 2).
type ApplicationLauncher interface {
	LaunchApplication(launchContext context.Context, identifier string) error
}

// ConfigLocation gives the per machine configuration file and state directory
// (TD1).
type ConfigLocation interface {
	ConfigFile() (string, error)
	StateDirectory() (string, error)
}
