package jetbrains_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/jetbrains"
)

// golandName is how the set names GoLand.
const golandName = "GoLand"

// TestRegister_GivesEveryJetBrainsIDETheActions: every IDE on the IntelliJ
// platform shares the JetBrains actions, each matched by its identifiers on
// both platforms, early access builds among them, and named for itself; other
// JetBrains applications are left out, and the drawer's groups are filled.
func TestRegister_GivesEveryJetBrainsIDETheActions(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := jetbrains.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	for identifier, wantName := range map[string]string{
		"com.jetbrains.goland":                      golandName,
		"com.jetbrains.goland-EAP":                  golandName,
		`C:\JetBrains\GoLand\bin\goland64.exe`:      golandName,
		"com.jetbrains.intellij":                    "IntelliJ IDEA",
		"com.jetbrains.intellij.ce":                 "IntelliJ IDEA",
		"com.jetbrains.pycharm":                     "PyCharm",
		"com.jetbrains.rider":                       "Rider",
		`C:\JetBrains\Rider\bin\rider.exe`:          "Rider",
		"com.jetbrains.webstorm":                    "WebStorm",
		"com.jetbrains.CLion-EAP":                   "CLion",
		"com.jetbrains.rustrover":                   "RustRover",
		"com.google.android.studio":                 "Android Studio",
		`C:\Program Files\Android\bin\studio64.exe`: "Android Studio",
	} {
		set, found := registry.AppSetFor(identifier)
		if !found || set.ID != jetbrains.SetID || set.NameFor(identifier) != wantName {
			t.Errorf("%s: want %s in the JetBrains set, got %v %q", identifier, wantName, found, set.NameFor(identifier))
		}
	}

	for _, other := range []string{"com.jetbrains.toolbox", "com.jetbrains.gateway", "com.example.idea"} {
		if _, found := registry.AppSetFor(other); found {
			t.Errorf("%s: want it left out", other)
		}
	}

	if kinds := registry.KindNamesIn(jetbrains.SetID); len(kinds) != len(jetbrains.Actions()) {
		t.Errorf("want one kind per action, got %d", len(kinds))
	}

	set, _ := registry.AppSetFor("com.jetbrains.pycharm")
	for _, group := range set.Groups {
		assertGroupFilled(t, registry, group)
	}
}

// assertGroupFilled checks a group has a JetBrains kind for each of its keys.
func assertGroupFilled(t *testing.T, registry *brick.Registry, group brick.AppGroup) {
	t.Helper()

	if len(group.Kinds) != group.Width*group.Height {
		t.Errorf("%s: want %d kinds for a %dx%d group, got %d",
			group.Name, group.Width*group.Height, group.Width, group.Height, len(group.Kinds))
	}

	for _, kindName := range group.Kinds {
		if kind, found := registry.Kind(kindName); !found || kind.Scope() != jetbrains.SetID {
			t.Errorf("%s: want %s among the JetBrains kinds", group.Name, kindName)
		}
	}
}
