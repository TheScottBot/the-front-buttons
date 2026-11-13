package editor_test

import (
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The copy forms' routes, named once.
const (
	copyToNewPath   = "/page/copy/new"
	copyOntoPath    = "/page/copy/onto"
	copiedPageName  = "Work two"
	secondPageSaved = "/page?page=1&saved=key"
)

// twoPageConfiguration has Work with a key at row 1, column 1, and Play with
// a key at the tile given.
func twoPageConfiguration(playRow string, playColumn string) string {
	return startingConfiguration + `  - name: Play
    keys:
      - row: ` + playRow + `
        column: ` + playColumn + `
        kind: open_url
        parameters:
          url: https://example.test/play
          label: Play
`
}

func (harness *editorHarness) replaceFile(t *testing.T, content string) {
	t.Helper()

	writeError := os.WriteFile(harness.configPath, []byte(content), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}
}

// TestEditor_CopiesAPageToANewPage: the author asked to copy a page. A new
// page, at the end and named as given, gets a copy of every key in place.
func TestEditor_CopiesAPageToANewPage(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	response := harness.post(t, copyToNewPath, url.Values{pageField: {"0"}, nameField: {copiedPageName}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != secondPageSaved {
		t.Fatalf("want a redirect to the new page, got %d %s", response.Code, response.Header().Get("Location"))
	}

	saved := harness.onDisk(t)
	if len(saved.Pages) != 2 || saved.Pages[1].Name != copiedPageName {
		t.Fatalf("want the copy as a second page named %q, got %+v", copiedPageName, saved.Pages)
	}

	original, copied := saved.Pages[0].Keys, saved.Pages[1].Keys
	if len(copied) != len(original) || copied[0].Position != original[0].Position ||
		!reflect.DeepEqual(copied[0].Parameters, original[0].Parameters) {
		t.Fatalf("want every key copied in place, got %+v from %+v", copied, original)
	}

	refused := harness.post(t, copyToNewPath, url.Values{pageField: {"0"}, nameField: {"  "}})
	if refused.Code != http.StatusUnprocessableEntity || len(harness.onDisk(t).Pages) != 2 {
		t.Fatalf("want a new page without a name refused, got %d", refused.Code)
	}
}

// TestEditor_CopiesAPageOntoAnotherPage: copying onto an existing page adds
// every key beside the keys already there.
func TestEditor_CopiesAPageOntoAnotherPage(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, twoPageConfiguration("1", "1"))

	response := harness.post(t, copyOntoPath, url.Values{pageField: {"0"}, toField: {"1"}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != secondPageSaved {
		t.Fatalf("want a redirect to the page copied onto, got %d %s", response.Code, response.Header().Get("Location"))
	}

	play := harness.onDisk(t).Pages[1]
	for _, position := range []layout.Position{{Row: 0, Column: 0}, {Row: 1, Column: 1}} {
		if _, found := keyAtPosition(play, position); !found {
			t.Errorf("want a key at %+v on Play, got %+v", position, play.Keys)
		}
	}
}

// TestEditor_RefusesACopyThatWouldOverwriteAKey: as with a single key, nothing
// is ever silently overwritten. A copy that would land on a key copies
// nothing, and says which tiles clash.
func TestEditor_RefusesACopyThatWouldOverwriteAKey(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	clashing := twoPageConfiguration("0", "0")
	harness.replaceFile(t, clashing)

	response := harness.post(t, copyOntoPath, url.Values{pageField: {"0"}, toField: {"1"}})
	body := response.Body.String()

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 refusing the copy, got %d", response.Code)
	}

	for _, want := range []string{"Error: ", "row 1, column 1"} {
		if !strings.Contains(body, want) {
			t.Errorf("want %q in the refusal", want)
		}
	}

	onDisk, readError := os.ReadFile(harness.configPath)
	if readError != nil || string(onDisk) != clashing {
		t.Fatal("want the file untouched by a refused copy")
	}

	for _, target := range []string{"0", "7", "x"} {
		refused := harness.post(t, copyOntoPath, url.Values{pageField: {"0"}, toField: {target}})
		if refused.Code != http.StatusUnprocessableEntity {
			t.Errorf("copy onto %q: want refused, got %d", target, refused.Code)
		}
	}
}

// TestEditor_PageOffersBothCopies: the page holds a copy to a new page, and a
// copy onto any other page, never itself.
func TestEditor_PageOffersBothCopies(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, twoPageConfiguration("1", "1"))

	body := harness.get(t, "/page?page=0").Body.String()

	for _, want := range []string{
		`action="` + copyToNewPath + `"`,
		`action="` + copyOntoPath + `"`,
		`<option value="1">Page 2: Play</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s on the page", want)
		}
	}

	if strings.Contains(body, `<option value="0">`) {
		t.Error("want the page never offered as a copy of itself")
	}

	if strings.Contains(newHarness(t, nil).get(t, "/page?page=0").Body.String(), `action="`+copyOntoPath+`"`) {
		t.Error("want no copy onto another page when there is no other page")
	}
}
