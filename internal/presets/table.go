package presets

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// ErrInvalidPreset reports a preset row that is malformed or cannot expand into
// its target kind. It is a programming error found when the table is built,
// before any configuration is read.
var ErrInvalidPreset = errors.New("invalid preset")

// presetNamePattern is the spelling a preset name must use, the same as a brick
// kind's, so the two read alike in configuration.
var presetNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// templateFunctions are the only functions a preset template may call. Each
// escapes for one place in a URL, so a friendly field can never change a URL's
// structure, only fill its slot.
func templateFunctions() template.FuncMap {
	return template.FuncMap{
		"path_segment":  url.PathEscape,
		"query":         url.QueryEscape,
		"uri_component": uriComponent,
	}
}

// uriComponent escapes a value as JavaScript's encodeURIComponent does, the
// encoding Microsoft's Teams deep link examples show (a colon as %3A, an at sign
// as %40, a space as %20). url.PathEscape leaves those characters as they are.
func uriComponent(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

// Row is one preset: friendly fields, declared with the same tag mechanism as a
// brick's parameters, and templates that expand them into a target kind's
// parameters. A row is data, so adding a preset adds no code path.
type Row struct {
	name         string
	targetKind   string
	fieldsSchema brick.Schema
	templates    map[string]*template.Template
}

// Define declares a preset whose friendly fields are the struct F. Each template
// is keyed by the target field it fills and refers to F's fields by name, as in
// {{path_segment .server_id}}.
func Define[F any](presetName string, targetKind string, templates map[string]string) (Row, error) {
	if !presetNamePattern.MatchString(presetName) {
		return Row{}, fmt.Errorf("%w: name %q must be lower snake case", ErrInvalidPreset, presetName)
	}

	fieldsSchema, schemaError := brick.DeriveSchema[F]()
	if schemaError != nil {
		return Row{}, fmt.Errorf("%w: %s: %w", ErrInvalidPreset, presetName, schemaError)
	}

	parsedTemplates := make(map[string]*template.Template, len(templates))

	for targetField, templateText := range templates {
		parsedTemplate, parseError := template.New(targetField).
			Funcs(templateFunctions()).
			Option("missingkey=error").
			Parse(templateText)
		if parseError != nil {
			return Row{}, fmt.Errorf("%w: %s: template for %s: %w", ErrInvalidPreset, presetName, targetField, parseError)
		}

		parsedTemplates[targetField] = parsedTemplate
	}

	return Row{name: presetName, targetKind: targetKind, fieldsSchema: fieldsSchema, templates: parsedTemplates}, nil
}

// Name is the preset's name as written in configuration.
func (row Row) Name() string { return row.name }

// TargetKind is the brick kind the preset expands into.
func (row Row) TargetKind() string { return row.targetKind }

// FieldsSchema is the schema of the preset's friendly fields.
func (row Row) FieldsSchema() brick.Schema { return row.fieldsSchema }

// Table holds every preset, each checked against the registry when the table is
// built so a preset that cannot expand is found at start up, not on a key press.
type Table struct {
	registry *brick.Registry
	rows     map[string]Row
}

// NewTable builds a table of the given rows over the registry's kinds.
func NewTable(registry *brick.Registry, rows ...Row) (Table, error) {
	table := Table{registry: registry, rows: make(map[string]Row, len(rows))}

	for _, row := range rows {
		if _, duplicate := table.rows[row.name]; duplicate {
			return Table{}, fmt.Errorf("%w: duplicate preset %q", ErrInvalidPreset, row.name)
		}

		checkError := checkRowAgainstTarget(registry, row)
		if checkError != nil {
			return Table{}, checkError
		}

		table.rows[row.name] = row
	}

	return table, nil
}

func checkRowAgainstTarget(registry *brick.Registry, row Row) error {
	refuse := func(format string, arguments ...any) error {
		return fmt.Errorf("%w: %s: %s", ErrInvalidPreset, row.name, fmt.Sprintf(format, arguments...))
	}

	targetKind, targetFound := registry.Kind(row.targetKind)
	if !targetFound {
		return refuse("target kind %q is not registered", row.targetKind)
	}

	appearanceNames := appearanceFieldNames()

	for targetFieldName := range row.templates {
		targetField, fieldFound := targetKind.Schema().Field(targetFieldName)

		switch {
		case !fieldFound:
			return refuse("template fills %q, which %s does not have", targetFieldName, row.targetKind)
		case appearanceNames[targetFieldName]:
			return refuse("template fills appearance field %q, which passes through unchanged", targetFieldName)
		case targetField.Type != brick.FieldTypeText:
			return refuse("template fills %q, a %s field; templates produce text", targetFieldName, targetField.Type)
		}
	}

	for _, targetField := range targetKind.Schema().Fields() {
		if _, templated := row.templates[targetField.Name]; targetField.Required && !templated {
			return refuse("required target field %q has no template", targetField.Name)
		}
	}

	return nil
}

// appearanceFieldNames is the set of shared appearance fields, which pass from a
// preset's friendly fields to its target unchanged.
func appearanceFieldNames() map[string]bool {
	names := make(map[string]bool)
	for _, fieldName := range brick.AppearanceFieldNames() {
		names[fieldName] = true
	}

	return names
}

// Row returns the preset with the given name.
func (table Table) Row(presetName string) (Row, bool) {
	row, found := table.rows[presetName]

	return row, found
}

// Names returns every preset name in sorted order.
func (table Table) Names() []string {
	names := make([]string, 0, len(table.rows))
	for name := range table.rows {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

// Expand turns a preset's friendly fields, created by the row's FieldsSchema,
// into its target kind's parameters. Every rendered value is validated by the
// target's schema, so a preset cannot produce parameters the kind would refuse
// if they were written by hand.
func (table Table) Expand(row Row, fields any) (any, error) {
	targetKind, targetFound := table.registry.Kind(row.targetKind)
	if !targetFound {
		return nil, fmt.Errorf("%w: %s: target kind %q is not registered", ErrInvalidPreset, row.name, row.targetKind)
	}

	templateValues := make(map[string]any)
	for _, field := range row.fieldsSchema.Fields() {
		templateValues[field.Name] = row.fieldsSchema.Read(fields, field)
	}

	targetSchema := targetKind.Schema()
	parameters := targetSchema.NewParameters()

	for targetFieldName, fieldTemplate := range row.templates {
		var rendered strings.Builder

		executeError := fieldTemplate.Execute(&rendered, templateValues)
		if executeError != nil {
			return nil, fmt.Errorf("%w: %s: rendering %s: %w", ErrInvalidPreset, row.name, targetFieldName, executeError)
		}

		targetField, _ := targetSchema.Field(targetFieldName)

		assignError := targetSchema.Assign(parameters, targetField, rendered.String())
		if assignError != nil {
			return nil, fmt.Errorf("preset %s: %w", row.name, assignError)
		}
	}

	passError := passAppearanceThrough(row.fieldsSchema, fields, targetSchema, parameters)
	if passError != nil {
		return nil, fmt.Errorf("preset %s: %w", row.name, passError)
	}

	return parameters, nil
}

// passAppearanceThrough copies each appearance value that was set. An unset
// value is the zero value, which is what absence means, so it is not copied.
func passAppearanceThrough(fieldsSchema brick.Schema, fields any, targetSchema brick.Schema, parameters any) error {
	for appearanceName := range appearanceFieldNames() {
		sourceField, sourceFound := fieldsSchema.Field(appearanceName)
		targetField, targetFound := targetSchema.Field(appearanceName)

		if !sourceFound || !targetFound {
			continue
		}

		value := fieldsSchema.Read(fields, sourceField)
		if value == "" {
			continue
		}

		assignError := targetSchema.Assign(parameters, targetField, value)
		if assignError != nil {
			return fmt.Errorf("passing %s through: %w", appearanceName, assignError)
		}
	}

	return nil
}
