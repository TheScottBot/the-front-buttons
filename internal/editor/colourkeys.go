package editor

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// Colouring ticked keys: select mode gives every ticked key the background
// chosen, or clears it, through the same save path as every other change, so
// keys meant to be seen together, such as all of Spotify's, can be coloured
// at once. A preset key keeps its colour among the preset's fields, which is
// what the file records, as well as in the parameters it expands to.

// The colour's fields and its message.
const (
	backgroundFieldName = "background"
	clearColourField    = "clear"
	colouredQuery       = "coloured"
	colouredMessage     = "Coloured. The deck shows the change within a second."
)

// errBadColour reports a colour a key's background cannot take.
var errBadColour = errors.New("that colour cannot be used")

// colourTickedKeys gives the ticked keys a background, or clears it.
func (pages *pageHandlers) colourTickedKeys(writer http.ResponseWriter, request *http.Request) {
	selection := selectionFrom(request)
	clearing := request.PostForm.Get(clearColourField) != ""
	colour := strings.ToUpper(strings.TrimSpace(request.PostForm.Get(backgroundFieldName)))

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if !pageExists(*configuration, selection.source) {
			return errCopyNoPage
		}

		if len(selection.tiles) == 0 {
			return errCopyNoKeys
		}

		page := &configuration.Pages[selection.source]

		keys, keysError := tickedKeys(*page, selection.tiles)
		if keysError != nil {
			return keysError
		}

		for _, key := range keys {
			coloured, colourError := pages.coloured(key, colour, clearing)
			if colourError != nil {
				return colourError
			}

			place(page, key.Position, coloured)
		}

		return nil
	})
	if saveError == nil {
		// The redirect's only variable parts are a formatted integer and a
		// constant, so it can only ever name one of the editor's own pages.
		address := fmt.Sprintf("/page?page=%d&saved=%s", selection.source, colouredQuery)
		http.Redirect(writer, request, address, http.StatusSeeOther)

		return
	}

	messages := saveProblems(saveError)
	if isCopyRefusal(saveError) || errors.Is(saveError, errBadColour) {
		messages = []string{errorPrefix + saveError.Error()}
	}

	configuration, _ := pages.configuration()
	pages.renderPageSelecting(writer, request, http.StatusUnprocessableEntity, configuration, selection, messages)
}

// coloured is a copy of a key with its background set, or cleared, in its
// parameters and, for a preset key, in the preset's fields.
func (pages *pageHandlers) coloured(key config.Key, colour string, clearing bool) (config.Key, error) {
	coloured := key.Clone()

	kind, known := pages.options.Registry.Kind(coloured.Kind)
	if !known {
		return coloured, fmt.Errorf("%w: the key at %s is of no known kind", errBadColour, tileWords(key.Position))
	}

	setError := setBackground(kind.Schema(), coloured.Parameters, colour, clearing)
	if setError == nil && coloured.Preset != nil {
		if row, found := pages.options.Presets.Row(coloured.Preset.Name); found {
			setError = setBackground(row.FieldsSchema(), coloured.Preset.Fields, colour, clearing)
		}
	}

	return coloured, setError
}

// setBackground sets or clears the background field of parameters made by a
// schema, if the schema has one.
func setBackground(schema brick.Schema, parameters any, colour string, clearing bool) error {
	field, found := schema.Field(backgroundFieldName)
	if !found {
		return nil
	}

	if clearing {
		return schema.Clear(parameters, field) //nolint:wrapcheck // the background is never required
	}

	assignError := schema.Assign(parameters, field, colour)
	if fieldError, isFieldError := errors.AsType[*brick.FieldError](assignError); isFieldError {
		return fmt.Errorf("%w: %s, written #RRGGBB", errBadColour, fieldError.Detail)
	}

	return assignError //nolint:wrapcheck // any other failure is the schema's own, shown as it is
}
