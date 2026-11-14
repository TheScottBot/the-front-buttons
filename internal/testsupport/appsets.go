package testsupport

import (
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// AppSetExpectations are what an application's own keys must be: the set's
// scope, the name each identifier is matched to, identifiers it must not
// match, and how many kinds it registers, of which some must be among them.
type AppSetExpectations struct {
	SetID     string
	Names     map[string]string
	Unmatched []string
	KindCount int
	SomeKinds []string
}

// AssertAppSet checks a registry holding an application's set against what
// is expected of it, and that each of the set's groups names one of its kinds
// for each of its tiles.
func AssertAppSet(t *testing.T, registry *brick.Registry, want AppSetExpectations) {
	t.Helper()

	assertMatches(t, registry, want)
	assertKinds(t, registry, want)

	for _, set := range registry.AppSets() {
		if set.ID == want.SetID {
			for _, group := range set.Groups {
				assertGroupFilled(t, registry, want.SetID, group)
			}
		}
	}
}

// assertMatches checks which identifiers the set matches, and the names it
// gives them.
func assertMatches(t *testing.T, registry *brick.Registry, want AppSetExpectations) {
	t.Helper()

	for identifier, wantName := range want.Names {
		set, found := registry.AppSetFor(identifier)
		if !found || set.ID != want.SetID || set.NameFor(identifier) != wantName {
			t.Errorf("%s: want %s in the %s set, got %v %+v", identifier, wantName, want.SetID, found, set)
		}
	}

	for _, other := range want.Unmatched {
		if set, found := registry.AppSetFor(other); found && set.ID == want.SetID {
			t.Errorf("%s: want it left out of the %s set", other, want.SetID)
		}
	}
}

// assertKinds checks the set's kinds: how many, and some by name.
func assertKinds(t *testing.T, registry *brick.Registry, want AppSetExpectations) {
	t.Helper()

	kinds := registry.KindNamesIn(want.SetID)
	if len(kinds) != want.KindCount {
		t.Errorf("want %d kinds in %s, got %d", want.KindCount, want.SetID, len(kinds))
	}

	for _, kindName := range want.SomeKinds {
		if !slices.Contains(kinds, kindName) {
			t.Errorf("want %s among the %s kinds", kindName, want.SetID)
		}
	}
}

// assertGroupFilled checks a group names one of its set's kinds for each of
// its tiles.
func assertGroupFilled(t *testing.T, registry *brick.Registry, setID string, group brick.AppGroup) {
	t.Helper()

	if len(group.Kinds) != group.Width*group.Height {
		t.Errorf("%s: want %d kinds, got %d", group.Name, group.Width*group.Height, len(group.Kinds))
	}

	for _, kindName := range group.Kinds {
		if kind, found := registry.Kind(kindName); !found || kind.Scope() != setID {
			t.Errorf("%s: want %s among the %s kinds", group.Name, kindName, setID)
		}
	}
}
