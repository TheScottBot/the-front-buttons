// Package presskey is the press_key brick: pressing its key sends one
// momentary key press, such as F13, to whatever has the keyboard focus, so a
// deck key can drive any application's shortcut (the author's request,
// evaluation, deck testing).
package presskey

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// KindName is the kind's name in configuration.
const KindName = "press_key"

// Parameters is the kind's one declaration of its parameters.
type Parameters struct {
	brick.Appearance

	Key string `brick:"key,required" format:"function_key" description:"The key to press: F13 to F24. macOS has only F13 to F20, and asks for Accessibility permission the first time."`
}

// Register adds the kind to a registry.
func Register(registry *brick.Registry) error {
	return brick.Register(registry, KindName, construct) //nolint:wrapcheck // Register already names the kind
}

func construct(parameters Parameters, ports brick.Ports) (any, error) {
	if ports.KeyPresser == nil {
		return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, KindName)
	}

	return keyBrick{key: parameters.Key, presser: ports.KeyPresser}, nil
}

// keyBrick presses one key.
type keyBrick struct {
	key     string
	presser platform.KeyPresser
}

// Press sends the key, and names any failure on the key by its category.
func (button keyBrick) Press(pressContext context.Context) error {
	pressError := button.presser.PressKey(pressContext, button.key)
	if pressError != nil {
		return fmt.Errorf("pressing %s: %w", button.key,
			brick.Categorised(platform.KeyFailureCategory(pressError), pressError))
	}

	return nil
}
