package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"unicode/utf8"

	"go.yaml.in/yaml/v4"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
)

// yamlErrorLine finds the line number in a YAML parser message, so a malformed
// file is located without the message's text, which can quote the file.
var yamlErrorLine = regexp.MustCompile(`line (\d+)`)

// Decoder reads and writes configuration files against one registry of brick
// kinds, one preset table and one deck grid. It holds no state between calls.
type Decoder struct {
	registry    *brick.Registry
	presetTable presets.Table
	grid        layout.Grid
	migrations  []migration
}

// NewDecoder returns a decoder for the given kinds, presets and grid, with this
// build's migrations.
func NewDecoder(registry *brick.Registry, presetTable presets.Table, grid layout.Grid) Decoder {
	return Decoder{registry: registry, presetTable: presetTable, grid: grid, migrations: productionMigrations()}
}

// Decode parses and validates a configuration file. Any problem rejects the
// whole file with a *RejectionError listing every problem found.
func (decoder Decoder) Decode(fileContent []byte) (Configuration, error) {
	var list problemList

	if len(fileContent) > MaximumFileBytes {
		list.addf(nil, "", ReasonTooLarge, "file is %d bytes; the limit is %d", len(fileContent), MaximumFileBytes)

		return Configuration{SchemaVersion: 0, Pages: nil}, &RejectionError{Problems: list.problems}
	}

	root, rootFound := parseRoot(fileContent, &list)
	if !rootFound {
		return Configuration{SchemaVersion: 0, Pages: nil}, &RejectionError{Problems: list.problems}
	}

	if !decoder.bringToCurrentVersion(root, &list) {
		return Configuration{SchemaVersion: 0, Pages: nil}, &RejectionError{Problems: list.problems}
	}

	configuration := decoder.decodeRoot(root, &list)
	if len(list.problems) > 0 {
		return Configuration{SchemaVersion: 0, Pages: nil}, &RejectionError{Problems: list.problems}
	}

	return configuration, nil
}

// parseRoot parses exactly one YAML document whose root is a mapping.
func parseRoot(fileContent []byte, list *problemList) (*yaml.Node, bool) {
	var document yaml.Node

	loadError := yaml.Load(fileContent, &document, yaml.WithUniqueKeys(true))
	if loadError != nil {
		list.problems = append(list.problems, Problem{
			Path:   "",
			Line:   lineOfYAMLError(loadError),
			Reason: ReasonMalformed,
			Detail: "not a single well formed YAML document with unique keys",
		})

		return nil, false
	}

	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		list.addf(nil, "", ReasonMalformed, "the file is empty")

		return nil, false
	}

	root := document.Content[0]

	if containsAlias(root) {
		list.addf(root, "", ReasonMalformed, "YAML anchors and aliases are not supported")

		return nil, false
	}

	if duplicateKey := firstDuplicateKey(root); duplicateKey != nil {
		list.addf(duplicateKey, "", ReasonMalformed, "field %q appears twice in one mapping", duplicateKey.Value)

		return nil, false
	}

	if root.Kind != yaml.MappingNode {
		list.addf(root, "", ReasonWrongType, "expected a mapping of names to values at the top level")

		return nil, false
	}

	return root, true
}

func lineOfYAMLError(loadError error) int {
	match := yamlErrorLine.FindStringSubmatch(loadError.Error())
	if match == nil {
		return 0
	}

	line, _ := strconv.Atoi(match[1])

	return line
}

// bringToCurrentVersion checks schema_version and runs any migrations needed to
// reach CurrentSchemaVersion. The version is required: a silent default would
// read an old file as a new one (datum 0.11).
func (decoder Decoder) bringToCurrentVersion(root *yaml.Node, list *problemList) bool {
	versionNode := mappingValue(root, schemaVersionName)
	if versionNode == nil {
		list.addf(root, schemaVersionName, ReasonMissing, "missing required field %q", schemaVersionName)

		return false
	}

	fileVersion, isInteger := integerOf(versionNode)
	if !isInteger {
		list.addf(versionNode, schemaVersionName, ReasonWrongType, "expected a whole number written in decimal")

		return false
	}

	if fileVersion > CurrentSchemaVersion {
		list.addf(versionNode, schemaVersionName, ReasonUnsupportedVersion,
			"version %d is newer than this build, which reads version %d", fileVersion, CurrentSchemaVersion)

		return false
	}

	for version := fileVersion; version < CurrentSchemaVersion; version++ {
		step, stepFound := decoder.migrationFrom(version)
		if !stepFound {
			list.addf(versionNode, schemaVersionName, ReasonUnsupportedVersion,
				"version %d cannot be migrated to version %d", fileVersion, CurrentSchemaVersion)

			return false
		}

		migrateError := step.migrate(root)
		if migrateError != nil {
			list.addf(versionNode, schemaVersionName, ReasonMalformed,
				"migrating from version %d failed: %v", version, migrateError)

			return false
		}
	}

	versionNode.Value = strconv.Itoa(CurrentSchemaVersion)

	return true
}

// mappingValue returns the value for a name in a mapping, or nil.
func mappingValue(mapping *yaml.Node, entryName string) *yaml.Node {
	for entryIndex := 0; entryIndex+1 < len(mapping.Content); entryIndex += mappingEntryWidth {
		if mapping.Content[entryIndex].Value == entryName {
			return mapping.Content[entryIndex+1]
		}
	}

	return nil
}

func (decoder Decoder) decodeRoot(root *yaml.Node, list *problemList) Configuration {
	configuration := Configuration{SchemaVersion: CurrentSchemaVersion, Pages: nil}

	entries, isMapping := list.mappingEntries(root, "", schemaVersionName, pagesName)
	if !isMapping {
		return configuration
	}

	pagesNode, pagesPresent := entries[pagesName]
	if !pagesPresent {
		list.addf(root, pagesName, ReasonMissing, "missing required field %q", pagesName)

		return configuration
	}

	if pagesNode.Kind != yaml.SequenceNode {
		list.addf(pagesNode, pagesName, ReasonWrongType, "expected a list of pages")

		return configuration
	}

	if len(pagesNode.Content) > MaximumPages {
		list.addf(pagesNode, pagesName, ReasonTooMany, "%d pages; the limit is %d", len(pagesNode.Content), MaximumPages)

		return configuration
	}

	for pageIndex, pageNode := range pagesNode.Content {
		pagePath := fmt.Sprintf("%s[%d]", pagesName, pageIndex)
		configuration.Pages = append(configuration.Pages, decoder.decodePage(pageNode, pagePath, list))
	}

	return configuration
}

func (decoder Decoder) decodePage(pageNode *yaml.Node, pagePath string, list *problemList) Page {
	page := Page{Name: "", Keys: nil}

	entries, isMapping := list.mappingEntries(pageNode, pagePath, pageNameName, keysName)
	if !isMapping {
		return page
	}

	page.Name = decodePageName(entries, pageNode, pagePath, list)

	keysNode, keysPresent := entries[keysName]
	if !keysPresent {
		list.addf(pageNode, pagePath, ReasonMissing, "missing required field %q", keysName)

		return page
	}

	keysPath := pagePath + "." + keysName

	if keysNode.Kind != yaml.SequenceNode {
		list.addf(keysNode, keysPath, ReasonWrongType, "expected a list of keys")

		return page
	}

	gridCapacity := decoder.grid.Rows() * decoder.grid.Columns()
	if len(keysNode.Content) > gridCapacity {
		list.addf(keysNode, keysPath, ReasonTooMany,
			"%d keys; the deck has %d positions", len(keysNode.Content), gridCapacity)

		return page
	}

	occupied := make(map[layout.Position]bool, len(keysNode.Content))

	for keyIndex, keyNode := range keysNode.Content {
		keyPath := fmt.Sprintf("%s[%d]", keysPath, keyIndex)

		key, keyDecoded := decoder.decodeKey(keyNode, keyPath, list)
		if !keyDecoded {
			continue
		}

		if occupied[key.Position] {
			list.addf(keyNode, keyPath, ReasonDuplicatePosition,
				"row %d column %d already has a key on this page", key.Position.Row, key.Position.Column)

			continue
		}

		occupied[key.Position] = true
		page.Keys = append(page.Keys, key)
	}

	return page
}

func decodePageName(entries map[string]*yaml.Node, pageNode *yaml.Node, pagePath string, list *problemList) string {
	nameNode, namePresent := entries[pageNameName]
	if !namePresent {
		list.addf(pageNode, pagePath, ReasonMissing, "missing required field %q", pageNameName)

		return ""
	}

	pageName, isText := textOf(nameNode)
	if !isText {
		list.addf(nameNode, pagePath+"."+pageNameName, ReasonWrongType, "expected text")

		return ""
	}

	if nameLength := utf8.RuneCountInString(pageName); nameLength < 1 || nameLength > MaximumPageNameCharacters {
		list.addf(nameNode, pagePath+"."+pageNameName, ReasonOutOfBounds,
			"must be between 1 and %d characters", MaximumPageNameCharacters)

		return ""
	}

	return pageName
}

func (decoder Decoder) decodeKey(keyNode *yaml.Node, keyPath string, list *problemList) (Key, bool) {
	key := Key{Position: layout.Position{Row: 0, Column: 0}, Kind: "", Parameters: nil, Preset: nil}

	entries, isMapping := list.mappingEntries(keyNode, keyPath,
		rowName, columnName, kindName, parametersName, presetName, fieldsName)
	if !isMapping {
		return key, false
	}

	position, positionValid := decoder.decodePosition(entries, keyNode, keyPath, list)

	_, namesKind := entries[kindName]
	_, namesPreset := entries[presetName]

	var brickDecoded bool

	switch {
	case namesKind == namesPreset:
		list.addf(keyNode, keyPath, ReasonKindOrPreset, "a key names exactly one of %q or %q", kindName, presetName)
	case namesKind:
		brickDecoded = decoder.decodeKindKey(entries, keyNode, keyPath, list, &key)
	default:
		brickDecoded = decoder.decodePresetKey(entries, keyNode, keyPath, list, &key)
	}

	key.Position = position

	return key, positionValid && brickDecoded
}

func (decoder Decoder) decodePosition(
	entries map[string]*yaml.Node,
	keyNode *yaml.Node,
	keyPath string,
	list *problemList,
) (layout.Position, bool) {
	row, rowValid := list.requireInteger(entries, keyNode, keyPath, rowName)
	column, columnValid := list.requireInteger(entries, keyNode, keyPath, columnName)

	if !rowValid || !columnValid {
		return layout.Position{Row: 0, Column: 0}, false
	}

	position := layout.Position{Row: int(row), Column: int(column)}

	switch {
	case !decoder.grid.Contains(position):
		list.addf(keyNode, keyPath, ReasonOutsideGrid, "row %d column %d is outside the %d by %d deck",
			row, column, decoder.grid.Rows(), decoder.grid.Columns())

		return position, false
	case decoder.grid.IsReserved(position):
		list.addf(keyNode, keyPath, ReasonReservedPosition,
			"row %d column %d is reserved on every page for navigation", row, column)

		return position, false
	default:
		return position, true
	}
}

func (decoder Decoder) decodeKindKey(
	entries map[string]*yaml.Node,
	keyNode *yaml.Node,
	keyPath string,
	list *problemList,
	key *Key,
) bool {
	if fieldsNode, hasFields := entries[fieldsName]; hasFields {
		list.addf(fieldsNode, keyPath, ReasonKindOrPreset, "%q belongs to a preset key, not a kind key", fieldsName)

		return false
	}

	kindNode := entries[kindName]

	kindValue, isText := textOf(kindNode)
	if !isText {
		list.addf(kindNode, keyPath+"."+kindName, ReasonWrongType, "expected text")

		return false
	}

	kind, kindFound := decoder.registry.Kind(kindValue)
	if !kindFound {
		list.addf(kindNode, keyPath+"."+kindName, ReasonUnknownKind, "unknown brick kind %q", kindValue)

		return false
	}

	parameters, parametersValid := decodeFields(kind.Schema(), entries[parametersName], keyNode,
		keyPath+"."+parametersName, list)

	key.Kind = kindValue
	key.Parameters = parameters

	return parametersValid
}

func (decoder Decoder) decodePresetKey(
	entries map[string]*yaml.Node,
	keyNode *yaml.Node,
	keyPath string,
	list *problemList,
	key *Key,
) bool {
	if parametersNode, hasParameters := entries[parametersName]; hasParameters {
		list.addf(parametersNode, keyPath, ReasonKindOrPreset,
			"%q belongs to a kind key; a preset key has %q", parametersName, fieldsName)

		return false
	}

	presetNode := entries[presetName]

	presetValue, isText := textOf(presetNode)
	if !isText {
		list.addf(presetNode, keyPath+"."+presetName, ReasonWrongType, "expected text")

		return false
	}

	row, rowFound := decoder.presetTable.Row(presetValue)
	if !rowFound {
		list.addf(presetNode, keyPath+"."+presetName, ReasonUnknownPreset, "unknown preset %q", presetValue)

		return false
	}

	fieldsPath := keyPath + "." + fieldsName

	fields, fieldsValid := decodeFields(row.FieldsSchema(), entries[fieldsName], keyNode, fieldsPath, list)
	if !fieldsValid {
		return false
	}

	parameters, expandError := decoder.presetTable.Expand(row, fields)
	if expandError != nil {
		reason := ReasonBadFormat
		if fieldError, isFieldError := errors.AsType[*brick.FieldError](expandError); isFieldError {
			reason = Reason(fieldError.Reason)
		}

		list.addf(presetNode, fieldsPath, reason, "the preset's fields do not expand into a valid %s key", row.TargetKind())

		return false
	}

	key.Kind = row.TargetKind()
	key.Parameters = parameters
	key.Preset = &PresetSource{Name: presetValue, Fields: fields}

	return true
}

// decodeFields reads a mapping of brick fields through the schema: each value's
// YAML tag must match the field's type, then the schema checks bounds and
// format. An absent mapping is an empty one, so a kind whose fields are all
// optional needs none.
func decodeFields(
	schema brick.Schema,
	fieldsNode *yaml.Node,
	parentNode *yaml.Node,
	fieldsPath string,
	list *problemList,
) (any, bool) {
	parameters := schema.NewParameters()
	problemsBefore := len(list.problems)

	entries := map[string]*yaml.Node{}

	if fieldsNode != nil {
		fieldNames := make([]string, 0, len(schema.Fields()))
		for _, field := range schema.Fields() {
			fieldNames = append(fieldNames, field.Name)
		}

		decodedEntries, isMapping := list.mappingEntries(fieldsNode, fieldsPath, fieldNames...)
		if !isMapping {
			return parameters, false
		}

		entries = decodedEntries
	}

	for _, field := range schema.Fields() {
		valueNode, present := entries[field.Name]
		if !present {
			if field.Required {
				list.addf(parentNode, fieldsPath, ReasonMissing, "missing required field %q", field.Name)
			}

			continue
		}

		value, expected := fieldValueOf(field, valueNode)
		if expected != "" {
			list.addf(valueNode, fieldsPath+"."+field.Name, ReasonWrongType, "%s", expected)

			continue
		}

		assignError := schema.Assign(parameters, field, value)
		if fieldError, isFieldError := errors.AsType[*brick.FieldError](assignError); isFieldError {
			list.addf(valueNode, fieldsPath+"."+field.Name, Reason(fieldError.Reason), "%s", fieldError.Detail)
		} else if assignError != nil {
			list.addf(valueNode, fieldsPath+"."+field.Name, ReasonWrongType, "%v", assignError)
		}
	}

	return parameters, len(list.problems) == problemsBefore
}
