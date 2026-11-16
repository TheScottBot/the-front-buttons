package editor_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// TestEditor_KeepsHandWrittenPlatformBridge: the editor has no form for the
// platform bridge, so a change it saves keeps the bridge as written, and a
// file without one is saved without one.
func TestEditor_KeepsHandWrittenPlatformBridge(t *testing.T) {
	t.Parallel()

	for _, want := range []config.PlatformBridge{config.PlatformBridgePurego, config.PlatformBridgeCgo, ""} {
		file := copyKeysConfiguration
		if want != "" {
			file = strings.Replace(file, "pages:\n", "platform_bridge: "+string(want)+"\npages:\n", 1)
		}

		harness := newHarness(t, nil)
		harness.replaceFile(t, file)

		renamePage(t, harness, "Arcade")

		if got := harness.onDisk(t); got.Pages[3].Name != "Arcade" || got.PlatformBridge != want {
			t.Errorf("want the rename saved and the bridge %q kept, got %q", want, got.PlatformBridge)
		}
	}
}

// TestEditor_UndoRefusesAfterAHandEditToThePlatformBridge: changing only the
// platform bridge by hand counts as a hand edit, so undo will not write over
// it.
func TestEditor_UndoRefusesAfterAHandEditToThePlatformBridge(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, copyKeysConfiguration)

	renamePage(t, harness, "Arcade")

	saved, readError := harness.store.OnDisk()
	if readError != nil {
		t.Fatal(readError)
	}

	saved.PlatformBridge = config.PlatformBridgePurego

	saveError := harness.store.Save(saved)
	if saveError != nil {
		t.Fatal(saveError)
	}

	response := harness.post(t, undoPath, url.Values{})
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want undo refused after the bridge changed by hand, got %d", response.Code)
	}

	requireUnchanged(t, harness, saved)
}
