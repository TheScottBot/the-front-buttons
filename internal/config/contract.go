package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The configuration contract's bounds, each taken from this table and never
// from the input (engineering datum 0.10). The contract is provisional until the
// author freezes it (datum 0.11); every change to a value here is recorded with
// its reason in the evaluation record.
const (
	// CurrentSchemaVersion is the schema version this build reads and writes.
	CurrentSchemaVersion = 1
	// MaximumFileBytes bounds the file before it is parsed. A deck of pages of
	// keys is a few kilobytes; a megabyte leaves room for hundreds of pages while
	// keeping a hostile or corrupted file from being parsed at all.
	MaximumFileBytes = 1 << 20
	// MaximumPages bounds the number of pages, so paging stays usable and a
	// file cannot ask for unbounded work.
	MaximumPages = 64
	// MaximumPageNameCharacters bounds a page name, which is shown on a
	// navigation key and so must fit on one.
	MaximumPageNameCharacters = 24
)

// The names used in the file. Each appears once here and is used by both the
// decoder and the encoder, so the two cannot disagree (datum 0.7).
const (
	schemaVersionName = "schema_version"
	pagesName         = "pages"
	pageNameName      = "name"
	keysName          = "keys"
	rowName           = "row"
	columnName        = "column"
	kindName          = "kind"
	parametersName    = "parameters"
	presetName        = "preset"
	fieldsName        = "fields"
)

// Reason categorises why a configuration was rejected, so rejections are counted
// and shown by category rather than by content (datum 0.15).
type Reason string

// The reasons a configuration is rejected. The field level reasons share their
// spelling with brick.RejectionReason.
const (
	ReasonTooLarge           Reason = "too_large"
	ReasonMalformed          Reason = "malformed"
	ReasonUnsupportedVersion Reason = "unsupported_version"
	ReasonUnknownField       Reason = "unknown_field"
	ReasonUnknownKind        Reason = "unknown_kind"
	ReasonUnknownPreset      Reason = "unknown_preset"
	ReasonKindOrPreset       Reason = "kind_or_preset"
	ReasonReservedPosition   Reason = "reserved_position"
	ReasonOutsideGrid        Reason = "outside_grid"
	ReasonDuplicatePosition  Reason = "duplicate_position"
	ReasonTooMany            Reason = "too_many"
	ReasonMissing            Reason = "missing"
	ReasonWrongType          Reason = "wrong_type"
	ReasonOutOfBounds        Reason = "out_of_bounds"
	ReasonBadFormat          Reason = "bad_format"
)

// Problem is one thing wrong with a configuration file: where, why, and a
// description that names fields but never echoes a value, since values can hold
// URLs with colleagues' addresses (specification section 7).
type Problem struct {
	Path   string
	Line   int
	Reason Reason
	Detail string
}

// RejectionError is a configuration file refused in full, with every problem found.
// A rejected file never replaces the running configuration.
type RejectionError struct {
	Problems []Problem
}

func (rejection *RejectionError) Error() string {
	descriptions := make([]string, 0, len(rejection.Problems))
	for _, problem := range rejection.Problems {
		descriptions = append(descriptions,
			fmt.Sprintf("%s (line %d): %s: %s", problem.Path, problem.Line, problem.Reason, problem.Detail))
	}

	return "configuration rejected: " + strings.Join(descriptions, "; ")
}

// Configuration is a decoded, validated configuration file.
type Configuration struct {
	SchemaVersion int
	Pages         []Page
}

// Page is one page of keys.
type Page struct {
	Name string
	Keys []Key
}

// Key is one configured key. Kind and Parameters are always the brick that runs;
// for a key written as a preset they are the expansion, and Preset keeps the
// preset's name and fields so the editor can show and re-edit what was written.
type Key struct {
	Position   layout.Position
	Kind       string
	Parameters any
	Preset     *PresetSource
}

// PresetSource is a preset as written in the file.
type PresetSource struct {
	Name   string
	Fields any
}

// clone returns a deep copy of the configuration, including each key's
// parameters and preset fields, which are pointers to structs of plain values.
func (configuration Configuration) clone() Configuration {
	pages := make([]Page, 0, len(configuration.Pages))

	for _, page := range configuration.Pages {
		keys := make([]Key, 0, len(page.Keys))

		for _, key := range page.Keys {
			clonedKey := Key{Position: key.Position, Kind: key.Kind, Parameters: clonePointer(key.Parameters), Preset: nil}
			if key.Preset != nil {
				clonedKey.Preset = &PresetSource{Name: key.Preset.Name, Fields: clonePointer(key.Preset.Fields)}
			}

			keys = append(keys, clonedKey)
		}

		pages = append(pages, Page{Name: page.Name, Keys: keys})
	}

	return Configuration{SchemaVersion: configuration.SchemaVersion, Pages: pages}
}

// clonePointer copies the struct a parameters pointer points to. Parameter
// structs hold only text, numbers, booleans and durations (the brick schema
// refuses anything else), so a shallow copy of the struct is a deep copy.
func clonePointer(pointer any) any {
	value := reflect.ValueOf(pointer)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return pointer
	}

	copied := reflect.New(value.Elem().Type())
	copied.Elem().Set(value.Elem())

	return copied.Interface()
}
