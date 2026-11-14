package claude_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/claude"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// TestRegister_GivesClaudeItsOwnKeys: Claude's table is pressable on both
// platforms, and its set is matched by its identifier, and by nothing of
// ChatGPT's, with a claude_ kind for each action and its group filled.
func TestRegister_GivesClaudeItsOwnKeys(t *testing.T) {
	t.Parallel()

	checkError := appkeys.Check(claude.Actions())
	if checkError != nil {
		t.Fatal(checkError)
	}

	registry := brick.NewRegistry()

	registerError := claude.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	testsupport.AssertAppSet(t, registry, testsupport.AppSetExpectations{
		SetID:     claude.SetID,
		Names:     map[string]string{"com.anthropic.claudefordesktop": "Claude"},
		Unmatched: []string{"com.openai.codex"},
		KindCount: len(claude.Actions()),
		SomeKinds: []string{"claude_next_session", "claude_previous_session", "claude_command_palette"},
	})
}
