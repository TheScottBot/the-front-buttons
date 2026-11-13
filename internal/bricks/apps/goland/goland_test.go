package goland_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/goland"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/jetbrains"
)

// TestRegister_GivesGoLandItsOwnBricks: GoLand is the first application with
// its own bricks, the JetBrains actions under its scope, matched to GoLand's
// identifiers, with groups for the drawer (evaluation, app screens).
func TestRegister_GivesGoLandItsOwnBricks(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := goland.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	for _, identifier := range []string{"com.jetbrains.goland", `C:\JetBrains\GoLand\bin\goland64.exe`} {
		set, found := registry.AppSetFor(identifier)
		if !found || set.ID != goland.SetID || set.Name != "GoLand" {
			t.Errorf("%s: want GoLand's set, got %+v", identifier, set)
		}
	}

	if _, found := registry.AppSetFor("com.jetbrains.pycharm"); found {
		t.Error("want PyCharm left out of GoLand's set")
	}

	kinds := registry.KindNamesIn(goland.SetID)
	if len(kinds) != len(jetbrains.Actions()) {
		t.Errorf("want one kind per action, got %d", len(kinds))
	}

	set, _ := registry.AppSetFor("com.jetbrains.goland")
	for _, group := range set.Groups {
		assertGroupFilled(t, registry, group)
	}
}

// assertGroupFilled checks a group has a GoLand kind for each of its keys.
func assertGroupFilled(t *testing.T, registry *brick.Registry, group brick.AppGroup) {
	t.Helper()

	if len(group.Kinds) != group.Width*group.Height {
		t.Errorf("%s: want %d kinds for a %dx%d group, got %d",
			group.Name, group.Width*group.Height, group.Width, group.Height, len(group.Kinds))
	}

	for _, kindName := range group.Kinds {
		if kind, found := registry.Kind(kindName); !found || kind.Scope() != goland.SetID {
			t.Errorf("%s: want %s among GoLand's kinds", group.Name, kindName)
		}
	}
}
