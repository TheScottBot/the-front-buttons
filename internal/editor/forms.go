package editor

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// Every form control for a brick field is named with this prefix, so a field
// can never collide with the form's own page, row, column, type or token.
const fieldPrefix = "field."

// The two halves of a type choice: a brick kind or a preset, written
// "kind:open_url" or "preset:discord_channel".
const (
	kindChoicePrefix   = "kind:"
	presetChoicePrefix = "preset:"
)

// textControl is the form control for text, and for durations written as text.
const textControl = "text"

// selectControl is the form control for a field chosen from a fixed list.
const selectControl = "select"

// errorPrefix starts every error a person reads, so an error is announced in
// words and never by colour alone (datum 0.16).
const errorPrefix = "Error: "

// The editor's own messages for a field it could not read.
const (
	requiredMessage = errorPrefix + "this field is required"
	integerMessage  = errorPrefix + "enter a whole number"
	durationMessage = errorPrefix + "enter a duration such as 90s or 5m"
)

// errUnknownType reports a type choice naming no kind or preset.
var errUnknownType = errors.New("unknown key type")

// fieldView is one form control, generated from a schema field.
type fieldView struct {
	ID            string
	Name          string
	Label         string
	Description   string
	DescriptionID string
	ErrorID       string
	Error         string
	Required      bool
	Secret        bool
	Control       string
	Value         string
	Checked       bool
	MinimumLength string
	MaximumLength string
	Minimum       string
	Maximum       string
	// Slider offers a slider beside a bounded number, kept in step with it
	// by the script; the number box is what is sent.
	Slider    bool
	Autofocus bool
	// Suggestions names the list a picker beside the field offers, and Choices
	// is that list, filled in when the page is shown.
	Suggestions brick.Suggestions
	Choices     []fieldChoice
	// Options is every value a field with a fixed set of values may take,
	// shown as a list rather than a text box (the author's request).
	Options []string
	// Replaces lists, space separated, the control IDs a check box sets aside
	// while it is ticked.
	Replaces string
	// A modifier is a check box named on each platform, shown as a chip in a
	// row with the others, the platform switch before the first: MacLabel and
	// WindowsLabel are its names, OnWindows whether this machine names it as
	// Windows does, and FirstModifier and LastModifier open and close the row.
	Modifier      bool
	MacLabel      string
	WindowsLabel  string
	OnWindows     bool
	FirstModifier bool
	LastModifier  bool
}

// The platforms the editor names modifier keys for.
const (
	ShortcutPlatformMac     = "mac"
	ShortcutPlatformWindows = "windows"
)

// nameModifiers names each modifier check box as this machine's platform does,
// and marks the first and last of the row they make.
func nameModifiers(views []fieldView, platform string) {
	first, last := -1, -1

	for index := range views {
		if !views[index].Modifier {
			continue
		}

		views[index].OnWindows = platform == ShortcutPlatformWindows
		views[index].Label = views[index].MacLabel

		if views[index].OnWindows {
			views[index].Label = views[index].WindowsLabel
		}

		if first < 0 {
			first = index
		}

		last = index
	}

	if first >= 0 {
		views[first].FirstModifier, views[last].LastModifier = true, true
	}
}

// fieldChoice is one value a picker offers: what the field takes, and the
// name a person knows it by.
type fieldChoice struct {
	Value string
	Label string
}

// DescribedBy lists the ids a control's aria-describedby names: its error
// first, so a screen reader reads why before what.
func (field fieldView) DescribedBy() string {
	if field.Error != "" {
		return field.ErrorID + " " + field.DescriptionID
	}

	return field.DescriptionID
}

// typeOption is one choice in the key type list.
type typeOption struct {
	Value    string
	Label    string
	Selected bool
}

// keyType resolves a type choice to the schema the form shows and, for a preset,
// the preset's name.
type keyType struct {
	choice     string
	schema     brick.Schema
	kindName   string
	presetName string
}

func (pages *pageHandlers) resolveType(choice string) (keyType, error) {
	if kindName, isKind := strings.CutPrefix(choice, kindChoicePrefix); isKind {
		kind, found := pages.options.Registry.Kind(kindName)
		if found {
			return keyType{choice: choice, schema: kind.Schema(), kindName: kindName, presetName: ""}, nil
		}
	}

	if presetName, isPreset := strings.CutPrefix(choice, presetChoicePrefix); isPreset {
		row, found := pages.options.Presets.Row(presetName)
		if found {
			return keyType{choice: choice, schema: row.FieldsSchema(), kindName: row.TargetKind(), presetName: presetName}, nil
		}
	}

	return keyType{choice: "", schema: brick.Schema{}, kindName: "", presetName: ""},
		fmt.Errorf("%w: %q", errUnknownType, choice)
}

// typeOptions lists every general kind and every preset from the registry and
// the preset table, so the editor keeps no list of its own (specification
// section 2). An application's own kinds are offered apart, on its app screen
// only (appTypeOptions).
func (pages *pageHandlers) typeOptions(selected string) []typeOption {
	kindNames := pages.options.Registry.KindNamesIn("")
	options := make([]typeOption, 0, len(kindNames)+len(pages.options.Presets.Names()))

	for _, kindName := range kindNames {
		value := kindChoicePrefix + kindName
		options = append(options, typeOption{Value: value, Label: "Brick: " + kindName, Selected: value == selected})
	}

	for _, presetName := range pages.options.Presets.Names() {
		row, _ := pages.options.Presets.Row(presetName)
		value := presetChoicePrefix + presetName
		label := "Preset: " + presetName + " (" + row.Action() + ")"
		options = append(options, typeOption{Value: value, Label: label, Selected: value == selected})
	}

	return options
}

// typeChoiceOf names the type of a configured key.
func typeChoiceOf(key config.Key) string {
	if key.Preset != nil {
		return presetChoicePrefix + key.Preset.Name
	}

	return kindChoicePrefix + key.Kind
}

// fieldViews builds a form control for every field of a schema, filled from the
// given parameters (nil for an empty form) and marked with any errors.
func fieldViews(schema brick.Schema, parameters any, fieldErrors map[string]string) []fieldView {
	views := make([]fieldView, 0, len(schema.Fields()))

	for _, field := range schema.Fields() {
		view := fieldView{
			ID:            "field-" + field.Name,
			Name:          fieldPrefix + field.Name,
			Label:         labelFor(field.Name),
			Description:   field.Description,
			DescriptionID: "field-" + field.Name + "-help",
			ErrorID:       "field-" + field.Name + "-error",
			Error:         fieldErrors[field.Name],
			Required:      field.Required,
			Secret:        field.Secret,
			Control:       controlFor(field.Type),
			Value:         "",
			Checked:       false,
			MinimumLength: "",
			MaximumLength: "",
			Minimum:       "",
			Maximum:       "",
			Slider:        false,
			Autofocus:     false,
			Suggestions:   field.Suggestions,
			Choices:       nil,
			Options:       nil,
			Replaces:      replacedIDs(field.Replaces),
			Modifier:      field.MacLabel != "",
			MacLabel:      field.MacLabel,
			WindowsLabel:  field.WindowsLabel,
			OnWindows:     false,
			FirstModifier: false,
			LastModifier:  false,
		}

		// A field whose format has a short, fixed set of values is chosen from
		// a list, so a typing slip cannot break the key; the schema still checks
		// whatever is sent.
		if options := brick.FormatChoices(field.Format); options != nil {
			view.Control, view.Options = selectControl, options
		}

		applyBounds(&view, field)

		if parameters != nil {
			fillValue(&view, field, schema.Read(parameters, field))
		} else {
			view.Value = field.Default
		}

		views = append(views, view)
	}

	return views
}

// labelFor turns a field name into words: api_base_url reads "Api base url".
func labelFor(fieldName string) string {
	words := strings.ReplaceAll(fieldName, "_", " ")

	return strings.ToUpper(words[:1]) + words[1:]
}

func controlFor(fieldType brick.FieldType) string {
	switch fieldType {
	case brick.FieldTypeBoolean:
		return "checkbox"
	case brick.FieldTypeInteger:
		return "number"
	case brick.FieldTypeText, brick.FieldTypeDuration:
		return textControl
	default:
		return textControl
	}
}

func applyBounds(view *fieldView, field brick.Field) {
	if field.Bounds == nil {
		return
	}

	switch field.Type { //nolint:exhaustive // only text and integers have bounds a browser can apply
	case brick.FieldTypeText:
		view.MinimumLength = strconv.FormatInt(field.Bounds.Minimum, 10)
		view.MaximumLength = strconv.FormatInt(field.Bounds.Maximum, 10)
	case brick.FieldTypeInteger:
		view.Minimum = strconv.FormatInt(field.Bounds.Minimum, 10)
		view.Maximum = strconv.FormatInt(field.Bounds.Maximum, 10)
		view.Slider = true
	}
}

func fillValue(view *fieldView, field brick.Field, value any) {
	switch typedValue := value.(type) {
	case string:
		view.Value = typedValue
	case int64:
		if typedValue != 0 || field.Required {
			view.Value = strconv.FormatInt(typedValue, 10)
		}
	case bool:
		view.Checked = typedValue
	case time.Duration:
		if typedValue != 0 {
			view.Value = typedValue.String()
		}
	}
}

// parseFields reads a submitted form through a schema, as the configuration
// decoder reads a file: each value converted by the field's type, then checked
// by the schema. It returns the parameters and a message for each field in
// error.
func parseFields(schema brick.Schema, form url.Values) (any, map[string]string) {
	parameters := schema.NewParameters()
	fieldErrors := map[string]string{}

	for _, field := range schema.Fields() {
		raw := strings.TrimSpace(form.Get(fieldPrefix + field.Name))

		value, message, present := convertField(field, raw)

		switch {
		case message != "":
			fieldErrors[field.Name] = message
		case !present && field.Required:
			fieldErrors[field.Name] = requiredMessage
		case present:
			assignError := schema.Assign(parameters, field, value)
			if fieldError, isFieldError := errors.AsType[*brick.FieldError](assignError); isFieldError {
				fieldErrors[field.Name] = errorPrefix + fieldError.Detail
			}
		}
	}

	return parameters, fieldErrors
}

// convertField returns a field's value, a message if it could not be read, and
// whether a value was given at all. An unticked checkbox is a present false.
func convertField(field brick.Field, raw string) (any, string, bool) {
	if field.Type == brick.FieldTypeBoolean {
		return raw != "", "", true
	}

	if raw == "" {
		return nil, "", false
	}

	switch field.Type { //nolint:exhaustive // booleans are handled above
	case brick.FieldTypeInteger:
		integerValue, parseError := strconv.ParseInt(raw, 10, 64)
		if parseError != nil {
			return nil, integerMessage, true
		}

		return integerValue, "", true
	case brick.FieldTypeDuration:
		duration, parseError := time.ParseDuration(raw)
		if parseError != nil {
			return nil, durationMessage, true
		}

		return duration, "", true
	default:
		return raw, "", true
	}
}

// buildKey turns parsed fields into a configured key, expanding a preset as the
// decoder would.
func (pages *pageHandlers) buildKey(resolved keyType, position layout.Position, fields any) (config.Key, error) {
	if resolved.presetName == "" {
		return config.Key{Position: position, Kind: resolved.kindName, Parameters: fields, Preset: nil}, nil
	}

	row, _ := pages.options.Presets.Row(resolved.presetName)

	parameters, expandError := pages.options.Presets.Expand(row, fields)
	if expandError != nil {
		return config.Key{Position: position, Kind: "", Parameters: nil, Preset: nil},
			fmt.Errorf("the preset's fields do not make a valid key: %w", expandError)
	}

	return config.Key{
		Position:   position,
		Kind:       resolved.kindName,
		Parameters: parameters,
		Preset:     &config.PresetSource{Name: resolved.presetName, Fields: fields},
	}, nil
}

// problemsByField sorts a rejection's problems into those about one field,
// keyed by the field's name, and the rest.
func problemsByField(rejection *config.RejectionError) (map[string]string, []string) {
	fieldErrors := map[string]string{}

	var general []string

	for _, problem := range rejection.Problems {
		fieldName := ""

		for _, section := range []string{".parameters.", ".fields."} {
			if _, after, found := strings.Cut(problem.Path, section); found {
				fieldName = after
			}
		}

		message := errorPrefix + problem.Detail
		if fieldName != "" {
			fieldErrors[fieldName] = message
		} else {
			general = append(general, message)
		}
	}

	return fieldErrors, general
}

// focusFirstError marks the first field in error to take focus when the page
// opens, so a refused save lands a keyboard or screen reader user on the
// problem, which then reads its label and its error. It reports whether any
// field is in error.
func focusFirstError(fields []fieldView) bool {
	for index := range fields {
		if fields[index].Error != "" {
			fields[index].Autofocus = true

			return true
		}
	}

	return false
}

// replacedIDs turns the fields a check box stands in for into their controls'
// IDs, as fieldViews names them.
func replacedIDs(fieldNames []string) string {
	ids := make([]string, 0, len(fieldNames))
	for _, fieldName := range fieldNames {
		ids = append(ids, "field-"+fieldName)
	}

	return strings.Join(ids, " ")
}
