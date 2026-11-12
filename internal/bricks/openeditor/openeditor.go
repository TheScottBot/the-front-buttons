// Package openeditor is the open_editor brick (TD9): pressing its key opens the
// local editor, so the deck can open its own configuration.
package openeditor

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/launching"
)

// KindName is the kind's name in configuration.
const KindName = "open_editor"

// Parameters is only the shared appearance; the editor's address is not
// configured, because it changes with each launch.
type Parameters struct {
	brick.Appearance
}

// Register adds the kind to a registry.
func Register(registry *brick.Registry) error {
	return brick.Register(registry, KindName, construct) //nolint:wrapcheck // Register already names the kind
}

func construct(_ Parameters, ports brick.Ports) (any, error) {
	if ports.URLOpener == nil || ports.EditorAddress == nil {
		return nil, fmt.Errorf("%w: %s needs a URL opener and the editor's address", brick.ErrPortMissing, KindName)
	}

	return editorBrick{address: ports.EditorAddress, opener: ports.URLOpener}, nil
}

// editorBrick opens the editor.
type editorBrick struct {
	address func() string
	opener  interface {
		OpenURL(openContext context.Context, address string) error
	}
}

// Press opens the editor at its current address.
func (button editorBrick) Press(pressContext context.Context) error {
	editorOpener := launching.New(button.address(), button.opener.OpenURL, "opening the editor")

	return editorOpener.Press(pressContext) //nolint:wrapcheck // launching names the action
}
