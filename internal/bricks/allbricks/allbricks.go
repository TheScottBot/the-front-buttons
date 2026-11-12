// Package allbricks registers every brick kind. Adding a kind is one package
// and one line here (specification section 2); the daemon, the editor and the
// generated reference all take their registry from this one function.
package allbricks

import (
	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/deckcontrol"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openeditor"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openurl"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/pipelinestatus"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/presskey"
)

// Register adds every brick kind to a registry.
func Register(registry *brick.Registry) error {
	for _, register := range []func(registry *brick.Registry) error{
		openurl.Register,
		openapplication.Register,
		pipelinestatus.Register,
		openeditor.Register,
		presskey.Register,
		deckcontrol.Register,
	} {
		registerError := register(registry)
		if registerError != nil {
			return registerError
		}
	}

	return nil
}
