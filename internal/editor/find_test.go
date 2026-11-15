package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// findResults is the list of a page's find results.
func findResults(t *testing.T, harness *editorHarness, path string) string {
	t.Helper()

	return regexp.MustCompile(`(?s)<div id="find-results"[^>]*>.*?</div>`).FindString(harness.get(t, path).Body.String())
}

// TestEditor_FindsKeysOnEveryPage: words typed into Find a key list every key,
// on any page or app screen, whose label, kind or page holds them all, case
// aside, each a link to its page with the key marked.
func TestEditor_FindsKeysOnEveryPage(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	results := findResults(t, harness, "/page?page=3&find=PASTE")
	for _, want := range []string{
		`<a href="/page?page=0&amp;found=0.2&amp;find=PASTE">Page 1, Work, row 1, column 3: Paste (open_url)</a>`,
	} {
		if !strings.Contains(results, want) {
			t.Errorf("results: want %s, got %s", want, results)
		}
	}

	// Every word must match: the kind, then a page's name.
	build := findResults(t, harness, "/page?page=0&find=goland+build")
	if !strings.Contains(build, `<a href="/page?page=1&amp;found=1.1&amp;find=goland&#43;build">`+
		`The GoLand screen, row 2, column 2: Build (jetbrains_build)</a>`) || strings.Contains(build, "Cut") {
		t.Errorf("want only GoLand's build key, got %s", build)
	}

	if none := findResults(t, harness, "/page?page=0&find=nothing+here"); !strings.Contains(none, "No keys match.") {
		t.Errorf("want no matches said, got %s", none)
	}
}

// TestEditor_MarksTheKeyFound: a page opened from a result marks the key
// found, and keeps the search open with its results.
func TestEditor_MarksTheKeyFound(t *testing.T) {
	t.Parallel()

	body := newCopyKeysHarness(t).get(t, "/page?page=0&found=0.2&find=paste").Body.String()

	for _, want := range []string{
		`<td data-tile="0.0.2" data-state="key" class="found">`,
		`<details class="maintenance find-drawer" id="find-drawer" open>`,
		`<input type="search" id="find-box" name="find" value="paste" autocomplete="off" spellcheck="false">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page: want %s", want)
		}
	}

	if strings.Count(body, `class="found"`) != 1 {
		t.Error("want only the key found marked")
	}
}

// TestEditor_FindDrawerWorksWithoutTheScript: Find a key is a plain search
// form, closed with no search, sent with its button; the script hides the
// button, searches as one types, and opens it on /.
func TestEditor_FindDrawerWorksWithoutTheScript(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	body := harness.get(t, "/page?page=0").Body.String()

	for _, want := range []string{
		`<details class="maintenance find-drawer" id="find-drawer">`,
		`<summary>Find a key</summary>`,
		`<form method="get" action="/page" class="find-form" role="search">`,
		`<label for="find-box">Words to find on any page</label>`,
		`<button type="submit" class="find-submit">Find</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page: want %s", want)
		}
	}

	if strings.Index(body, `id="find-drawer"`) > strings.Index(body, `id="select-drawer"`) {
		t.Error("want Find a key first of the drawers")
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"find-box", "find-results", `event.key !== "/"`} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	if !strings.Contains(harness.get(t, "/static/editor.css").Body.String(), ".grid td.found {") {
		t.Error("stylesheet: want the key found marked")
	}
}

// TestEditor_NamesAnUnlabelledKeyFoundByItsKindOnce: a key with no label of
// its own is named in a result by its kind, once.
func TestEditor_NamesAnUnlabelledKeyFoundByItsKindOnce(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: page_marker
`)

	results := findResults(t, harness, "/page?page=0&find=marker")
	if !strings.Contains(results, `>Page 1, Work, row 1, column 1: page_marker</a>`) {
		t.Errorf("want the marker named once, got %s", results)
	}
}
