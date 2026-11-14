package gitkraken_test

import (
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/gitkraken"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// TestActions_CoverGitKrakensTabsAndTerminal: GitKraken's table is pressable
// on both platforms and has its tabs, its terminal and its repository actions.
func TestActions_CoverGitKrakensTabsAndTerminal(t *testing.T) {
	t.Parallel()

	checkError := appkeys.Check(gitkraken.Actions())
	if checkError != nil {
		t.Fatal(checkError)
	}

	names := make([]string, 0, len(gitkraken.Actions()))
	for _, action := range gitkraken.Actions() {
		names = append(names, action.Name)
	}

	for _, wanted := range []string{
		"next_tab", "previous_tab", "new_terminal_tab", "next_terminal_tab", "open_in_terminal",
		"fetch_all", "create_branch", "stage_all", "commit", "command_palette",
	} {
		if !slices.Contains(names, wanted) {
			t.Errorf("want the %s action", wanted)
		}
	}
}

// TestActions_HoldCtrlOnWindows: GitKraken's Windows shortcuts hold Ctrl
// where macOS holds Command, and never the Windows key.
func TestActions_HoldCtrlOnWindows(t *testing.T) {
	t.Parallel()

	for _, action := range gitkraken.Actions() {
		if action.Windows.Command {
			t.Errorf("%s: want no Windows key held on Windows", action.Name)
		}

		if action.Name == "fetch_all" && (!action.Windows.Control || !action.Mac.Command || action.Windows.Key != "L") {
			t.Errorf("want fetch as Command L and Ctrl L, got %+v", action)
		}
	}
}

// TestRegister_GivesGitKrakenItsOwnKeys: GitKraken has its own set, matched
// by its identifier on both platforms and by nothing of JetBrains', with a
// gitkraken_ kind for each action and its groups filled.
func TestRegister_GivesGitKrakenItsOwnKeys(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := gitkraken.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	testsupport.AssertAppSet(t, registry, testsupport.AppSetExpectations{
		SetID: gitkraken.SetID,
		Names: map[string]string{
			"com.axosoft.gitkraken": "GitKraken", `C:\Users\me\AppData\Local\gitkraken\app-12.6.2\gitkraken.exe`: "GitKraken",
		},
		Unmatched: []string{"com.jetbrains.goland"},
		KindCount: len(gitkraken.Actions()),
		SomeKinds: []string{"gitkraken_next_tab"},
	})
}
