package editor_test

import (
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// The removal and undo routes, and the page a removal returns to.
const (
	removePath     = "/key/delete"
	undoPath       = "/undo"
	removedPage    = "/page?page=0&saved=removed"
	startingRow    = "0"
	startingColumn = "0"
)

func removeStartingKey(t *testing.T, harness *editorHarness) {
	t.Helper()

	recorder := harness.post(t, removePath, url.Values{
		pageField: {"0"}, rowField: {startingRow}, columnField: {startingColumn},
	})
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != removedPage {
		t.Fatalf("want a redirect to %s, got %d %s", removedPage, recorder.Code, recorder.Header().Get("Location"))
	}
}

// TestEditor_PageHoldsAHiddenBin: the author asked for a bin to drag a key to,
// appearing only while dragging (evaluation, deck testing). The page carries
// it hidden, drawn like a key with the trash icon and saying what it does.
func TestEditor_PageHoldsAHiddenBin(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	bin := regexp.MustCompile(`(?s)<div id="key-bin"[^>]*>.*?</div>`).FindString(body)
	if !strings.Contains(bin, " hidden") || !strings.Contains(bin, `src="/icon.png?name=trash"`) ||
		!strings.Contains(bin, "Drop here to remove") {
		t.Fatalf("want a hidden bin with the trash icon and its words, got %q", bin)
	}
}

// TestEditor_RemovingAKeyOffersUndo: a key binned by mistake can be put back,
// which is the guard against an accidental drop.
func TestEditor_RemovingAKeyOffersUndo(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	removeStartingKey(t, harness)

	body := harness.get(t, removedPage).Body.String()

	announcement := regexp.MustCompile(`(?s)<div class="announcement"[^>]*>.*?</div>`).FindString(body)
	if !strings.Contains(announcement, "Removed") || !strings.Contains(announcement, `action="/undo"`) ||
		!strings.Contains(announcement, `name="token" value="`+testToken+`"`) {
		t.Fatalf("want a removed message with an undo form carrying the token, got %q", announcement)
	}
}

func TestEditor_UndoPutsTheRemovedKeyBack(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	original := harness.onDisk(t).Pages[0].Keys[0]

	removeStartingKey(t, harness)

	if keys := harness.onDisk(t).Pages[0].Keys; len(keys) != 0 {
		t.Fatalf("want the key removed, got %d keys", len(keys))
	}

	recorder := harness.post(t, undoPath, url.Values{})
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("want a redirect after undoing, got %d: %s", recorder.Code, recorder.Body.String())
	}

	keys := harness.onDisk(t).Pages[0].Keys
	if len(keys) != 1 || !reflect.DeepEqual(keys[0], original) {
		t.Fatalf("want the key back exactly, got %+v from %+v", keys, original)
	}

	if again := harness.post(t, undoPath, url.Values{}); again.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(again.Body.String(), "nothing to undo") {
		t.Fatalf("want a second undo refused as nothing to undo, got %d", again.Code)
	}
}

// TestEditor_UndoStepsBackThroughEachChange: a key removed and its tile then
// filled is undone a change at a time: first the new key goes, then the
// removed one comes back.
func TestEditor_UndoStepsBackThroughEachChange(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	original := harness.onDisk(t).Pages[0].Keys[0]

	removeStartingKey(t, harness)
	harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {startingRow}, columnField: {startingColumn}, typeField: {openURLType},
		urlField: {"https://example.test/new"},
	})

	harness.post(t, undoPath, url.Values{})

	if keys := harness.onDisk(t).Pages[0].Keys; len(keys) != 0 {
		t.Fatalf("want the new key undone first, got %+v", keys)
	}

	harness.post(t, undoPath, url.Values{})

	keys := harness.onDisk(t).Pages[0].Keys
	if len(keys) != 1 || !reflect.DeepEqual(keys[0], original) {
		t.Fatalf("want the removed key back, got %+v", keys)
	}
}

func TestEditor_UndoNeedsTheToken(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	removeStartingKey(t, harness)

	recorder := harness.postRaw(t, undoPath, url.Values{}, func(request *http.Request) {
		request.Header.Set("Origin", testOrigin)
	})
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("want undo without the token refused, got %d", recorder.Code)
	}
}
