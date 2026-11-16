package config

import (
	"fmt"
	"reflect"
	"slices"
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
	// MaximumApplicationCharacters bounds an app screen's application, as an
	// application key's own field is bounded.
	MaximumApplicationCharacters = 512
	// MaximumWorldClocks bounds the world view's clocks: a row each, and the
	// XL has four rows.
	MaximumWorldClocks = 4
	// MaximumClockLabelCharacters bounds a world clock's label, which is drawn
	// on one key in the clock's face, four of its widest letters fitting.
	MaximumClockLabelCharacters = 4
	// MaximumZoneCharacters bounds a world clock's zone name; the longest in
	// the zone database is about thirty.
	MaximumZoneCharacters = 64
)

// The names used in the file. Each appears once here and is used by both the
// decoder and the encoder, so the two cannot disagree (datum 0.7).
const (
	schemaVersionName  = "schema_version"
	pagesName          = "pages"
	pageNameName       = "name"
	applicationName    = "application"
	keysName           = "keys"
	rowName            = "row"
	columnName         = "column"
	kindName           = "kind"
	parametersName     = "parameters"
	presetName         = "preset"
	fieldsName         = "fields"
	worldClocksName    = "world_clocks"
	clockLabelName     = "label"
	clockZoneName      = "zone"
	platformBridgeName = "platform_bridge"
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
	// ReasonWrongApplication is an application's own brick on a page that is
	// not that application's screen.
	ReasonWrongApplication Reason = "wrong_application"
	// ReasonUnknownZone is a world clock's zone the zone database does not
	// know.
	ReasonUnknownZone Reason = "unknown_zone"
	// ReasonDuplicateLabel is a world clock's label another clock has.
	ReasonDuplicateLabel Reason = "duplicate_label"
	// ReasonUnknownBridge is a platform bridge that is neither cgo nor purego.
	ReasonUnknownBridge Reason = "unknown_bridge"
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
	// WorldClocks are the page dashboard's world clocks, in order, or nil
	// when the file names none, meaning DefaultWorldClocks. The decoder never
	// fills the defaults in, so saving a file never writes them into it.
	WorldClocks []WorldClock
	// PlatformBridge is the bridge the daemon reaches the system through, or
	// empty when the file names none, meaning PlatformBridgeCgo. The decoder
	// never fills the default in, so saving a file never writes it into it.
	PlatformBridge PlatformBridge
	Pages          []Page
}

// WorldClock is one of the world view's clocks: the label on its first key,
// and its zone, a name in the zone database such as Europe/Berlin.
type WorldClock struct {
	Label string
	Zone  string
}

// DefaultWorldClocks are the world clocks of a file that names none: Thyme's
// own, UTC, London, Berlin and Sao Paulo.
func DefaultWorldClocks() []WorldClock {
	return []WorldClock{
		{Label: "UTC", Zone: "UTC"},
		{Label: "LDN", Zone: "Europe/London"},
		{Label: "BER", Zone: "Europe/Berlin"},
		{Label: "SAO", Zone: "America/Sao_Paulo"},
	}
}

// WorldClocksOrDefault is the world clocks the dashboard shows: the file's,
// or the defaults when it names none.
func (configuration Configuration) WorldClocksOrDefault() []WorldClock {
	if configuration.WorldClocks == nil {
		return DefaultWorldClocks()
	}

	return configuration.WorldClocks
}

// PlatformBridge names how the daemon reaches the system's own libraries:
// through cgo, or through purego with no C compiler in the build.
type PlatformBridge string

// The platform bridges a file may name.
const (
	PlatformBridgeCgo    PlatformBridge = "cgo"
	PlatformBridgePurego PlatformBridge = "purego"
)

// PlatformBridgeOrDefault is the bridge the daemon uses: the file's, or cgo
// when it names none.
func (configuration Configuration) PlatformBridgeOrDefault() PlatformBridge {
	if configuration.PlatformBridge == "" {
		return PlatformBridgeCgo
	}

	return configuration.PlatformBridge
}

// Page is one page of keys.
type Page struct {
	Name string
	// Application, when set, makes the page that application's screen: shown
	// in place of the regular pages when a key opens it, outside the regular
	// paging, with its top left key kept for the application's icon, which
	// returns (evaluation, app screens). Empty for a regular page.
	Application string
	Keys        []Key
}

// IsAppScreen reports whether the page is an application's screen.
func (page Page) IsAppScreen() bool { return page.Application != "" }

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

// Clone returns a deep copy of the configuration, including each key's
// parameters and preset fields, which are pointers to structs of plain values.
func (configuration Configuration) Clone() Configuration {
	pages := make([]Page, 0, len(configuration.Pages))

	for _, page := range configuration.Pages {
		keys := make([]Key, 0, len(page.Keys))

		for _, key := range page.Keys {
			keys = append(keys, key.Clone())
		}

		pages = append(pages, Page{Name: page.Name, Application: page.Application, Keys: keys})
	}

	return Configuration{
		SchemaVersion:  configuration.SchemaVersion,
		WorldClocks:    slices.Clone(configuration.WorldClocks),
		PlatformBridge: configuration.PlatformBridge,
		Pages:          pages,
	}
}

// Clone returns a deep copy of the key, so a copy made in the editor shares no
// parameters or preset fields with the key it was copied from.
func (key Key) Clone() Key {
	cloned := Key{Position: key.Position, Kind: key.Kind, Parameters: clonePointer(key.Parameters), Preset: nil}
	if key.Preset != nil {
		cloned.Preset = &PresetSource{Name: key.Preset.Name, Fields: clonePointer(key.Preset.Fields)}
	}

	return cloned
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
