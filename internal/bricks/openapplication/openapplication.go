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

	Application string `brick:"application,required" bounds:"1,512" description:"The application to launch: a bundle identifier or name on macOS; an executable path or application user model ID on Windows."`
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
