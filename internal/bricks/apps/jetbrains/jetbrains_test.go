package jetbrains_test

import (
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/jetbrains"
)

// TestActions_AreAllPressableOnBothPlatforms: every JetBrains action has a
// friendly label and a shortcut on macOS and on Windows whose key a shortcut
// may name, and no name repeats (evaluation, app screens).
func TestActions_AreAllPressableOnBothPlatforms(t *testing.T) {
	t.Parallel()

	checkError := appkeys.Check(jetbrains.Actions())
	if checkError != nil {
		t.Fatal(checkError)
	}

	names := make([]string, 0, len(jetbrains.Actions()))
	for _, action := range jetbrains.Actions() {
		names = append(names, action.Name)
	}

	for _, wanted := range []string{"build", "run", "debug", "step_over", "comment_line", "terminal", "commit"} {
		if !slices.Contains(names, wanted) {
			t.Errorf("want the %s action", wanted)
		}
	}
}
