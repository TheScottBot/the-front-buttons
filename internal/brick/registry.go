package brick

import (
	"fmt"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// Ports carries the port interfaces a brick constructor may use. Constructors
// receive ports, never adapters, so a brick can be tested against test doubles
// (specification section 2). Fields are added as the phases that own each port
// arrive: URL opening and application launching in TE7, credentials and the
// pipeline provider in TE8. A kind whose port is missing refuses to construct.
type Ports struct {
	URLOpener           platform.URLOpener
	ApplicationLauncher platform.ApplicationLauncher
	PipelineProvider    pipeline.Provider
	// EditorAddress returns the local editor's address, known only once the
	// editor's port is open.
	EditorAddress func() string
	// KeyPresser sends a key press, for press_key.
	KeyPresser platform.KeyPresser
	// MediaPlayer controls the music player, for the Spotify keys.
	MediaPlayer platform.MediaPlayer
	// WindowCycler moves between windows, for the JetBrains window keys.
	WindowCycler platform.WindowCycler
}

// Kind is one registered brick kind: its name, its derived schema and its
// constructor.
type Kind struct {
	name      string
	schema    Schema
	construct func(parameters any, ports Ports) (any, error)
}

// Name is the kind's name as written in configuration.
func (kind Kind) Name() string { return kind.name }

// Schema is the kind's derived parameter schema.
func (kind Kind) Schema() Schema { return kind.schema }

// Construct builds a brick from parameters created by the kind's schema.
func (kind Kind) Construct(parameters any, ports Ports) (any, error) {
	return kind.construct(parameters, ports)
}

// Registry holds every brick kind. It is a value the composition root builds,
// not a package global, so tests build their own registries of fake kinds.
type Registry struct {
	kinds map[string]Kind
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{kinds: make(map[string]Kind)}
}

// Register adds a kind whose parameters are the struct P. It derives the schema
// from P and refuses a malformed or duplicate kind name, or a P that breaks a tag
// rule. Adding a kind is one package and one call to Register.
func Register[P any](
	registry *Registry,
	kindName string,
	construct func(parameters P, ports Ports) (any, error),
) error {
	if !lowerSnakeCaseName.MatchString(kindName) {
		return fmt.Errorf("%w: kind name %q must be lower snake case", ErrInvalidKind, kindName)
	}

	if _, alreadyRegistered := registry.kinds[kindName]; alreadyRegistered {
		return fmt.Errorf("%w: kind %q is already registered", ErrInvalidKind, kindName)
	}

	schema, schemaError := DeriveSchema[P]()
	if schemaError != nil {
		return fmt.Errorf("registering %s: %w", kindName, schemaError)
	}

	registry.kinds[kindName] = Kind{
		name:   kindName,
		schema: schema,
		construct: func(parameters any, ports Ports) (any, error) {
			typedParameters, isTyped := parameters.(*P)
			if !isTyped {
				return nil, fmt.Errorf("%w: kind %q received %T", ErrParametersMismatch, kindName, parameters)
			}

			return construct(*typedParameters, ports)
		},
	}

	return nil
}

// Kind returns the kind with the given name.
func (registry *Registry) Kind(kindName string) (Kind, bool) {
	kind, found := registry.kinds[kindName]

	return kind, found
}

// KindNames returns every registered kind name in sorted order, so the editor's
// palette and the generated reference list kinds the same way every time.
func (registry *Registry) KindNames() []string {
	kindNames := make([]string, 0, len(registry.kinds))
	for kindName := range registry.kinds {
		kindNames = append(kindNames, kindName)
	}

	slices.Sort(kindNames)

	return kindNames
}
