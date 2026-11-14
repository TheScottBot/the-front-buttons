package editor

import (
	"fmt"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// What the drawer says of a brick when it is opened, from what the code
// holds and nothing more: a kind's own description where it has one, such as
// an app's key and the shortcut it presses; otherwise the kind's name and the
// fields it asks for, by their own descriptions. The appearance every key
// shares is left out.

// appearanceFields are the fields every key has, which say nothing of what a
// brick does.
//
//nolint:gochecknoglobals // a fixed list
var appearanceFields = []string{"label", "icon", "background", "icon_file"}

// aboutKind says what a kind is, and the fields it asks for.
func (pages *pageHandlers) aboutKind(kind brick.Kind) (string, []string) {
	if description := pages.options.Registry.Description(kind.Name()); description != "" {
		return description, nil
	}

	return fmt.Sprintf("The %s brick.", kind.Name()) + asksIntro(kind.Schema()), fieldAsks(kind.Schema())
}

// asksIntro introduces the fields a schema asks for, if it asks for any.
func asksIntro(schema brick.Schema) string {
	if len(fieldAsks(schema)) == 0 {
		return " It asks for nothing more than a key's appearance."
	}

	return " It asks for:"
}

// fieldAsks are a schema's fields, beyond the appearance, each by its name,
// whether it is required, and its own description.
func fieldAsks(schema brick.Schema) []string {
	var asks []string

	for _, field := range schema.Fields() {
		if slices.Contains(appearanceFields, field.Name) {
			continue
		}

		required := ""
		if field.Required {
			required = ", required"
		}

		asks = append(asks, fmt.Sprintf("%s%s: %s", field.Name, required, field.Description))
	}

	return asks
}
