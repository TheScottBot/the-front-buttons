// Package openapplication is the open_application brick: pressing its key
// launches an application, or raises it if it is already running.
package openapplication

import (
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/launching"
)

// KindName is the kind's name in configuration.
const KindName = "open_application"

// Parameters is the kind's one declaration of its parameters. The identifier has
// no format: on Windows it may be a path with spaces, and the platform adapter
// applies the launch target rule before anything runs.
type Parameters struct {
	brick.Appearance

	Application string `brick:"application,required" bounds:"1,512" suggest:"applications" description:"The application to launch: a bundle identifier or name on macOS; an executable path or application user model ID on Windows."`

	// UseAppIcon is the check box the author asked for. It stands apart so the
	// tag alignment check does not pad Application's tag out to the width of
	// this long description.
	UseAppIcon bool `brick:"use_app_icon" replaces:"icon,icon_file" description:"Draw the application's own icon instead of Icon or Icon file, and colour the key to match the icon's edge; a Background you set still wins. Not available on Windows yet, where the key shows the missing icon marker."`

	// AppScreen makes the key open the application's screen too: a page of
	// its own, outside the regular pages, with the application's icon top left
	// to return (evaluation, app screens).
	AppScreen bool `brick:"app_screen" description:"Also show the application's screen on the deck: a page of its own keys, with the application's icon top left to go back. Saving the key creates the screen if there is none."`
}

// ApplicationScreen offers the application's screen to the daemon when the
// key asks for it.
func (parameters Parameters) ApplicationScreen() (string, bool) {
	return parameters.Application, parameters.AppScreen
}

// ApplicationIcon offers the application's own icon to the daemon and the
// editor's preview when the key asks for it.
func (parameters Parameters) ApplicationIcon() (string, bool) {
	return parameters.Application, parameters.UseAppIcon
}

// Register adds the kind to a registry.
func Register(registry *brick.Registry) error {
	return brick.Register(registry, KindName, construct) //nolint:wrapcheck // Register already names the kind
}

func construct(parameters Parameters, ports brick.Ports) (any, error) {
	if ports.ApplicationLauncher == nil {
		return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, KindName)
	}

	launch := ports.ApplicationLauncher.LaunchApplication

	return launching.New(parameters.Application, launch, "launching the key's application"), nil
}
