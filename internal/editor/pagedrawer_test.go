package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestEditor_PageFormsSitInPageMaintenance: renaming, moving, copying and
// deleting the page, and adding one, sit together in a Page maintenance
// drawer under the grid, a plain disclosure that opens in place without the
// script, closed while there is a page and open while there are none.
func TestEditor_PageFormsSitInPageMaintenance(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	body := harness.get(t, "/page?page=0").Body.String()

	start := strings.Index(body, `<details class="maintenance" id="page-maintenance">`)
	if start < 0 {
		t.Fatal("want a page maintenance drawer, closed")
	}

	drawer := body[start:strings.Index(body, `<p class="shortcuts-help"`)]
	for _, want := range []string{
		`<summary>Page maintenance</summary>`,
		`<div class="maintenance-body">`,
		`action="/page/rename"`, `action="/page/reorder"`, `action="/page/copy/new"`, `action="/page/delete"`,
		`action="/page/add"`,
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("page maintenance: want %s", want)
		}
	}

	if strings.Index(body, `<table class="grid">`) > strings.Index(body, `id="page-maintenance"`) {
		t.Error("want page maintenance under the grid")
	}

	empty := newHarness(t, nil)
	empty.replaceFile(t, "schema_version: 1\npages: []\n")

	if !regexp.MustCompile(`(?s)<details class="maintenance" id="page-maintenance" open>.*There are no pages yet.*` +
		`action="/page/add"`).MatchString(empty.get(t, firstPagePath).Body.String()) {
		t.Error("want page maintenance open, offering a new page, when there are none")
	}

	stylesheet := harness.get(t, "/static/editor.css").Body.String()
	for _, want := range []string{
		".maintenance[open] > summary::before {",
		".maintenance-body {\n  padding: 0.25rem 1rem 1rem;\n  background: var(--group);",
	} {
		if !strings.Contains(stylesheet, want) {
			t.Errorf("stylesheet: want %s", want)
		}
	}
}

// TestEditor_ShortcutsSheetOpensOnAQuestionMark: the pages screen carries a
// sheet of every shortcut, opened with ? or the pointer's button, which the
// script shows.
func TestEditor_ShortcutsSheetOpensOnAQuestionMark(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	body := harness.get(t, firstPagePath).Body.String()

	for _, want := range []string{
		`<p class="shortcuts-help" hidden>Press ? for every shortcut, or ` +
			`<button type="button" class="shortcuts-open">show them</button>.</p>`,
		`<dialog id="shortcuts-dialog" aria-labelledby="shortcuts-title">`,
		`<caption>Command is Ctrl on Windows.</caption>`,
		`<tr><th scope="row">Undo or redo</th><td>Command-Z, Shift-Command-Z</td></tr>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page: want %s", want)
		}
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"shortcuts-dialog", `"?"`, "showModal"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}
}
