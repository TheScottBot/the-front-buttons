package editor_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// swapAction is the move form's third button: trading places with the key
// at the destination.
const swapAction = "swap"

// swapForm trades the key at a tile of Work for the one at a destination.
func swapForm(row string, column string, destination string) url.Values {
	return url.Values{
		pageField: {workPage}, rowField: {row}, columnField: {column}, toField: {destination}, actionField: {swapAction},
	}
}

// requireURLAt fails unless the key at a position opens the address given.
func requireURLAt(t *testing.T, harness *editorHarness, page int, position layout.Position, want string) {
	t.Helper()

	key, found := keyAtPosition(harness.onDisk(t).Pages[page], position)
	if !found || !strings.Contains(fmt.Sprintf("%+v", key.Parameters), want) {
		t.Errorf("want %s at page %d %v, got %+v", want, page, position, key)
	}
}

// TestEditor_SwapsTwoKeys: a key dropped on another trades places with it,
// on its page or another, each keeping everything but its tile.
func TestEditor_SwapsTwoKeys(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, movePath, swapForm("0", "1", "0.0.2"))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the swap saved, got %d: %s", response.Code, response.Body.String())
	}

	requireURLAt(t, harness, 0, layout.Position{Row: 0, Column: 1}, "https://example.test/paste")
	requireURLAt(t, harness, 0, layout.Position{Row: 0, Column: 2}, "https://example.test/cut")

	if keys := len(harness.onDisk(t).Pages[0].Keys); keys != 4 {
		t.Errorf("want four keys still on Work, got %d", keys)
	}
}

// TestEditor_RefusesASwapWithNoKeyOrOntoTheWrongPage: a swap needs a key at
// the destination, and refuses to put an application's own brick on a page
// that cannot hold it, changing nothing.
func TestEditor_RefusesASwapWithNoKeyOrOntoTheWrongPage(t *testing.T) {
	t.Parallel()

	for _, refused := range []struct {
		destination string
		want        string
	}{
		{"0.2.2", "there is no key there to swap with"},
		{golandScreen + ".1.1", "jetbrains_build belongs on its application&#39;s screen"},
		{"0.0.1", "that destination is where the key already is"},
	} {
		harness := newCopyKeysHarness(t)
		before := harness.onDisk(t)

		response := harness.post(t, movePath, swapForm("0", "1", refused.destination))
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), refused.want) {
			t.Errorf("%s: want refused with %q, got %d: %s", refused.destination, refused.want, response.Code,
				response.Body.String())
		}

		requireUnchanged(t, harness, before)
	}
}

// TestEditor_KeyPageOffersASwapWithEveryOtherKey: the keyboard's way to swap:
// a list of every other key on every page.
func TestEditor_KeyPageOffersASwapWithEveryOtherKey(t *testing.T) {
	t.Parallel()

	body := newCopyKeysHarness(t).get(t, "/key?page=0&row=0&column=1").Body.String()

	for _, want := range []string{
		`<label for="swap-with">Key to swap with</label>`,
		`<option value="0.0.2">Row 1, column 3: Paste</option>`,
		`<option value="1.1.1">Row 2, column 2: Build</option>`,
		`<button type="submit" name="action" value="swap">Swap keys</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("key page: want %s", want)
		}
	}

	if strings.Contains(body, `<option value="0.0.1">`) {
		t.Error("want the key itself left out of the list")
	}
}

// TestEditor_PasteReturnsToThePageItWentOnto: keys pasted onto the page open
// return there, not to the page they were copied from.
func TestEditor_PasteReturnsToThePageItWentOnto(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, copyKeysPath, url.Values{
		pageField: {workPage}, tileField: {cutTile}, toField: {playPage}, "showing": {playPage},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/page?page=3&saved=copied" {
		t.Fatalf("want a redirect to Play, got %d %s", response.Code, response.Header().Get("Location"))
	}
}

// TestEditor_ScriptSwapsAndPastesKeys: the script swaps a key dropped on
// another and copies, cuts and pastes keys with Command and C, X and V,
// through the same routes as the forms, and the hints say so.
func TestEditor_ScriptSwapsAndPastesKeys(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{
		`action: "swap"`, "swap-target", "front-buttons-keys", "Command-V", copyKeysPath, moveKeysPath,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	if !strings.Contains(harness.get(t, "/static/editor.css").Body.String(), ".grid td.swap-target::after {") {
		t.Error("stylesheet: want the swap marked on the key it would trade with")
	}

	if !strings.Contains(harness.get(t, firstPagePath).Body.String(), "Drop a key on another key to swap the two.") {
		t.Error("want the drag hint to say a key can be swapped")
	}
}
