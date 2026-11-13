package platform

import (
	"context"
	"errors"
	"image"
)

// ErrLaunchRefused reports a URL or application identifier refused before
// anything was run: empty, too long, starting with a hyphen (which a command line
// tool would read as an option), or holding a control character.
var ErrLaunchRefused = errors.New("launch target refused")

// ErrLaunchFailed reports that the operating system could not open a URL or
// launch an application. Its reason category is shown on the key.
var ErrLaunchFailed = errors.New("launch failed")

// ErrApplicationNotFound reports an application this system cannot find by the
// identifier given, so its icon cannot be read.
var ErrApplicationNotFound = errors.New("application not found")

// ErrIconUnavailable reports that this system's adapter cannot read
// applications' icons at all.
var ErrIconUnavailable = errors.New("application icons are not available on this system")

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

// Application is one installed application: the name a person knows it by and
// the identifier an ApplicationLauncher on this system takes for it.
type Application struct {
	Name       string
	Identifier string
}

// ApplicationCatalogue lists the applications installed for this user, for the
// editor's app picker. Each identifier is one this system's launcher accepts.
type ApplicationCatalogue interface {
	Applications(listContext context.Context) ([]Application, error)
}

// KeyPresser sends one momentary key press, down then up, to whatever has the
// keyboard focus. The key is a function_key value, F13 to F24.
type KeyPresser interface {
	PressKey(pressContext context.Context, key string) error
}

// ApplicationIcons reads an application's own icon, for a key that asks to show
// it. The identifier is read as the ApplicationLauncher on this system reads
// it, so the icon is that of the application a press opens.
type ApplicationIcons interface {
	Icon(iconContext context.Context, identifier string) (image.Image, error)
}

// ErrMediaUnavailable reports that this system's adapter cannot control a
// media player at all.
var ErrMediaUnavailable = errors.New("media control is not available on this system")

// ErrMediaNotAllowed reports that the operating system refused to let the
// daemon control the player; on macOS, the Automation permission for it.
var ErrMediaNotAllowed = errors.New("not allowed to control the media player")

// NowPlaying is what the player is doing: whether it is running and playing,
// and the current track's artwork, nil when there is none or it could not be
// read.
type NowPlaying struct {
	Running bool
	Playing bool
	Artwork image.Image
}

// MediaPlayer controls the author's music player, Spotify, for the 2x2 the
// author asked for (evaluation, deck testing): its album art, play or pause,
// and the previous and next tracks. Show brings the player to the front.
type MediaPlayer interface {
	NowPlaying(nowContext context.Context) (NowPlaying, error)
	PlayPause(pressContext context.Context) error
	Previous(pressContext context.Context) error
	Next(pressContext context.Context) error
	Show(pressContext context.Context) error
}

// MediaFailureCategory is the reason category a media key shows for a
// failure: permission, unavailable, or failed.
func MediaFailureCategory(failure error) string {
	switch {
	case errors.Is(failure, ErrMediaNotAllowed):
		return "permission"
	case errors.Is(failure, ErrMediaUnavailable):
		return "unavailable"
	default:
		return "failed"
	}
}

// The ways moving between windows fails, each shown on the key by its
// category.
var (
	// ErrWindowsNotPermitted reports that macOS has not given the daemon
	// Accessibility, which reading and raising other applications' windows
	// needs.
	ErrWindowsNotPermitted = errors.New("not permitted to move between windows")
	// ErrNoWindow reports that no window of the group is open.
	ErrNoWindow = errors.New("no window to move to")
	// ErrWindowsUnavailable reports that this system's adapter cannot move
	// between windows at all.
	ErrWindowsUnavailable = errors.New("moving between windows is not available on this system")
)

// WindowGroup names the applications whose windows a key moves through: every
// running application whose identifier starts with Prefix, but those named in
// Excluded.
type WindowGroup struct {
	Prefix   string
	Excluded []string
}

// WindowCycler brings forward the window a step from the one in front, among
// a group's windows in a stable order, wrapping round: the author's JetBrains
// next and previous window keys (evaluation, deck testing).
type WindowCycler interface {
	CycleWindows(cycleContext context.Context, group WindowGroup, step int) error
}

// WindowFailureCategory is the reason category a window key shows for a
// failure: permission, no_window, unavailable, or failed.
func WindowFailureCategory(failure error) string {
	switch {
	case errors.Is(failure, ErrWindowsNotPermitted):
		return "permission"
	case errors.Is(failure, ErrNoWindow):
		return "no_window"
	case errors.Is(failure, ErrWindowsUnavailable):
		return "unavailable"
	default:
		return "failed"
	}
}
