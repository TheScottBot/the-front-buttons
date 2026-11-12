package editor_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// TestEditor_GridMarksEveryTileForDragAndDrop: the author asked to drag a key
// to a new tile (evaluation, deck testing). Each tile says where it is and what
// it holds, only a key can be picked up, the page loads the drag script, and
// the drag hint stays hidden unless the script runs, so a page without it
// promises nothing.
func TestEditor_GridMarksEveryTileForDragAndDrop(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, "/page?page=0").Body.String()

	for _, want := range []string{
		`data-tile="0.0.0" data-state="key"`,
		`data-tile="0.1.1" data-state="empty"`,
		`data-tile="0.3.0" data-state="reserved"`,
		`data-tile="0.3.7" data-state="reserved"`,
		`<script src="/static/drag.js" defer></script>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s in the page", want)
		}
	}

	draggable := regexp.MustCompile(`<a [^>]*draggable="true"[^>]*>`).FindAllString(body, -1)
	if len(draggable) != 1 || !strings.Contains(draggable[0], "row=0&amp;column=0") {
		t.Errorf("want only the stored key draggable, got %q", draggable)
	}

	if !regexp.MustCompile(`<p class="drag-help"[^>]* hidden>`).MatchString(body) {
		t.Error("want the drag hint hidden until the script shows it")
	}
}

func TestEditor_ServesTheDragScript(t *testing.T) {
	t.Parallel()

	recorder := newHarness(t, nil).get(t, "/static/drag.js")

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("want the script served as JavaScript, got %d %s",
			recorder.Code, recorder.Header().Get("Content-Type"))
	}

	if !strings.Contains(recorder.Body.String(), "/key/move") {
		t.Fatal("want the script to post moves to /key/move")
	}
}
