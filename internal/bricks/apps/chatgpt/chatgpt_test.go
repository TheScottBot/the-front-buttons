package chatgpt_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/chatgpt"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// TestRegister_GivesChatGPTItsOwnKeys: ChatGPT's table is pressable on both
// platforms, and its set is matched by its identifier, and by nothing of
// Claude's, with a chatgpt_ kind for each action and its groups filled.
func TestRegister_GivesChatGPTItsOwnKeys(t *testing.T) {
	t.Parallel()

	checkError := appkeys.Check(chatgpt.Actions())
	if checkError != nil {
		t.Fatal(checkError)
	}

	registry := brick.NewRegistry()

	registerError := chatgpt.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	testsupport.AssertAppSet(t, registry, testsupport.AppSetExpectations{
		SetID:     chatgpt.SetID,
		Names:     map[string]string{"com.openai.codex": "ChatGPT", `C:\Program Files\ChatGPT\ChatGPT.exe`: "ChatGPT"},
		Unmatched: []string{"com.anthropic.claudefordesktop"},
		KindCount: len(chatgpt.Actions()),
		SomeKinds: []string{"chatgpt_next_chat", "chatgpt_previous_chat", "chatgpt_codex_mode", "chatgpt_terminal"},
	})
}

// TestActions_UseTheAppsOwnModeKeys: the modes are Control and a digit on
// macOS but Alt and a digit on Windows, as the app binds them, and moving
// between chats holds Ctrl and Alt on Windows where macOS holds Command and
// Option.
func TestActions_UseTheAppsOwnModeKeys(t *testing.T) {
	t.Parallel()

	codex, next := actionNamed(t, "codex_mode"), actionNamed(t, "next_chat")

	if !codex.Mac.Control || codex.Mac.Command || !codex.Windows.Alt || codex.Windows.Control || codex.Windows.Key != "3" {
		t.Errorf("want Codex as Control 3 and Alt 3, got %+v", codex)
	}

	if !next.Windows.Control || !next.Windows.Alt || next.Windows.Command || next.Windows.Key != "RightBracket" {
		t.Errorf("want the next chat as Ctrl Alt ] on Windows, got %+v", next.Windows)
	}
}

// actionNamed finds one action in ChatGPT's table.
func actionNamed(t *testing.T, name string) appkeys.Action {
	t.Helper()

	for _, action := range chatgpt.Actions() {
		if action.Name == name {
			return action
		}
	}

	t.Fatalf("want the %s action", name)

	return appkeys.Action{Name: "", Label: "", Mac: appkeys.Keys(""), Windows: appkeys.Keys("")}
}
