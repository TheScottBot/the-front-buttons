package brick

import (
	"fmt"
	"slices"
	"strings"
)

// AppSet is the bricks of one or more applications that share them: their
// scope, the set's name, the applications it is for, and the groups the
// editor's drawer offers. Its kinds are registered with RegisterScoped under
// its ID, and the editor offers them only on those applications' screens
// (evaluation, app screens).
type AppSet struct {
	// ID is the scope its kinds carry, lower snake case, such as jetbrains.
	ID string
	// Name is how the set as a whole is named, such as JetBrains IDEs.
	Name string
	// Applications are the applications it is for, each with its own name.
	Applications []AppIdentity
	// Groups are the keys the drawer offers placed together.
	Groups []AppGroup
}

// AppIdentity is one application an app set is for: its name, such as
// PyCharm, and the identifiers it is known by, as a key's application field
// holds them: bundle identifiers on macOS, and executable file names on
// Windows, matched against a path's last part.
type AppIdentity struct {
	Name        string
	Identifiers []string
}

// AppGroup is one group of an app set's keys placed at once, filled left to
// right then top to bottom.
type AppGroup struct {
	Name   string
	Label  string
	Width  int
	Height int
	Kinds  []string
}

// Matches reports whether the set is for an application identifier, ignoring
// case; a Windows path matches by its file name.
func (set AppSet) Matches(identifier string) bool {
	_, found := set.application(identifier)

	return found
}

// NameFor names the application an identifier is for, such as PyCharm, or
// the set itself for an identifier it is not for.
func (set AppSet) NameFor(identifier string) string {
	application, found := set.application(identifier)
	if !found {
		return set.Name
	}

	return application.Name
}

// application finds the application an identifier is for.
func (set AppSet) application(identifier string) (AppIdentity, bool) {
	if identifier == "" {
		return AppIdentity{Name: "", Identifiers: nil}, false
	}

	lastPart := identifier[strings.LastIndexAny(identifier, `/\`)+1:]
	index := slices.IndexFunc(set.Applications, func(application AppIdentity) bool {
		return slices.ContainsFunc(application.Identifiers, func(known string) bool {
			return strings.EqualFold(known, identifier) || strings.EqualFold(known, lastPart)
		})
	})

	if index < 0 {
		return AppIdentity{Name: "", Identifiers: nil}, false
	}

	return set.Applications[index], true
}

// RegisterAppSet adds an application's set, refusing a malformed or repeated
// ID.
func (registry *Registry) RegisterAppSet(set AppSet) error {
	switch {
	case !lowerSnakeCaseName.MatchString(set.ID):
		return fmt.Errorf("%w: app set %q must be lower snake case", ErrInvalidKind, set.ID)
	case slices.ContainsFunc(registry.appSets, func(known AppSet) bool { return known.ID == set.ID }):
		return fmt.Errorf("%w: app set %q is already registered", ErrInvalidKind, set.ID)
	}

	registry.appSets = append(registry.appSets, set)

	return nil
}

// AppSetFor returns the set for an application identifier, if there is one.
func (registry *Registry) AppSetFor(identifier string) (AppSet, bool) {
	index := slices.IndexFunc(registry.appSets, func(set AppSet) bool { return set.Matches(identifier) })
	if index < 0 {
		return AppSet{ID: "", Name: "", Applications: nil, Groups: nil}, false
	}

	return registry.appSets[index], true
}

// AppSetName returns the name of the set with a scope, if there is one.
func (registry *Registry) AppSetName(scope string) (string, bool) {
	index := slices.IndexFunc(registry.appSets, func(set AppSet) bool { return set.ID == scope })
	if index < 0 {
		return "", false
	}

	return registry.appSets[index].Name, true
}

// AppSets returns every registered set.
func (registry *Registry) AppSets() []AppSet {
	return slices.Clone(registry.appSets)
}
