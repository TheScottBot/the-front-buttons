package vscode_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/vscode"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// TestRegister_GivesVSCodeItsOwnKeys: VS Code's table is pressable on both
// platforms, and its set is matched by its identifiers, Insiders named apart,
// with a vscode_ kind for each action and its groups filled.
func TestRegister_GivesVSCodeItsOwnKeys(t *testing.T) {
	t.Parallel()

	checkError := appkeys.Check(vscode.Actions())
	if checkError != nil {
		t.Fatal(checkError)
	}

	registry := brick.NewRegistry()

	registerError := vscode.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	testsupport.AssertAppSet(t, registry, testsupport.AppSetExpectations{
		SetID: vscode.SetID,
		Names: map[string]string{
			"com.microsoft.VSCode": "VS Code", `C:\Program Files\Microsoft VS Code\Code.exe`: "VS Code",
			"com.microsoft.VSCodeInsiders": "VS Code Insiders",
		},
		Unmatched: []string{"com.jetbrains.goland"},
		KindCount: len(vscode.Actions()),
		SomeKinds: []string{"vscode_chat", "vscode_inline_chat", "vscode_next_suggestion"},
	})
}

// TestActions_OpenTheChatDifferentlyOnEachPlatform: the chat view is Control
// Command I on macOS and Ctrl Alt I on Windows, never the Windows key.
func TestActions_OpenTheChatDifferentlyOnEachPlatform(t *testing.T) {
	t.Parallel()

	for _, action := range vscode.Actions() {
		if action.Windows.Command {
			t.Errorf("%s: want no Windows key held on Windows", action.Name)
		}

		if action.Name == "chat" && (!action.Mac.Command || !action.Mac.Control || !action.Windows.Alt) {
			t.Errorf("want the chat as Control Command I and Ctrl Alt I, got %+v", action)
		}
	}
}
