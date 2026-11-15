package editor_test

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// withWorldClocks is the copy keys file with world clocks written in by hand.
func withWorldClocks(zone string) string {
	return strings.Replace(copyKeysConfiguration, "pages:\n",
		"world_clocks:\n  - label: BER\n    zone: "+zone+"\npages:\n", 1)
}

// TestEditor_KeepsHandWrittenWorldClocks: the editor has no form for the
// world clocks, so a change it saves keeps them as written, and a file
// without them is saved without them.
func TestEditor_KeepsHandWrittenWorldClocks(t *testing.T) {
	t.Parallel()

	for _, check := range []struct {
		file string
		want []config.WorldClock
	}{
		{withWorldClocks("Europe/Berlin"), []config.WorldClock{{Label: "BER", Zone: "Europe/Berlin"}}},
		{copyKeysConfiguration, nil},
	} {
		harness := newHarness(t, nil)
		harness.replaceFile(t, check.file)

		renamePage(t, harness, "Games")

		if got := harness.onDisk(t); got.Pages[3].Name != "Games" || !slices.Equal(got.WorldClocks, check.want) {
			t.Errorf("want the rename saved and world clocks %v kept, got %v", check.want, got.WorldClocks)
		}
	}
}

// TestEditor_UndoRefusesAfterAHandEditToTheWorldClocks: changing only the
// world clocks by hand counts as a hand edit, so undo will not write over it.
func TestEditor_UndoRefusesAfterAHandEditToTheWorldClocks(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, withWorldClocks("Europe/Berlin"))

	renamePage(t, harness, "Games")

	saved, readError := harness.store.OnDisk()
	if readError != nil {
		t.Fatal(readError)
	}

	saved.WorldClocks = []config.WorldClock{{Label: "BER", Zone: "Asia/Tokyo"}}

	saveError := harness.store.Save(saved)
	if saveError != nil {
		t.Fatal(saveError)
	}

	response := harness.post(t, undoPath, url.Values{})
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want undo refused after the world clocks changed by hand, got %d", response.Code)
	}

	requireUnchanged(t, harness, saved)
}
