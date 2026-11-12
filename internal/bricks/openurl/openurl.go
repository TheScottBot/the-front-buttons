// Package openurl is the open_url brick: pressing its key opens a URL with the
// system's handler (specification section 2). The Discord, Teams and D&D Beyond
// presets expand into it.
package openurl

import (
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/launching"
)

// KindName is the kind's name in configuration.
const KindName = "open_url"

// Parameters is the kind's one declaration of its parameters.
type Parameters struct {
	brick.Appearance

	URL string `brick:"url,required" bounds:"1,2048" format:"link_url" description:"The address the key opens: https, or a discord:// link, which opens the Discord app."`
}

// Register adds the kind to a registry.
func Register(registry *brick.Registry) error {
	return brick.Register(registry, KindName, construct) //nolint:wrapcheck // Register already names the kind
}

func construct(parameters Parameters, ports brick.Ports) (any, error) {
	if ports.URLOpener == nil {
		return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, KindName)
	}

	return launching.New(parameters.URL, ports.URLOpener.OpenURL, "opening the key's URL"), nil
}
