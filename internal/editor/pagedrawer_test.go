package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestEditor_PageFormsSitInAPageDrawer: renaming, moving, copying and
// deleting the page, and adding one, sit together in the Page drawer, whose
// tab the script shows; without the script they stay on the page, open.
func TestEditor_PageFormsSitInAPageDrawer(t *testing.T) {
	t.Parallel()

	body := newCopyKeysHarness(t).get(t, "/page?page=0").Body.String()

	drawer := regexp.MustCompile(`(?s)<aside id="page-drawer".*?</aside>`).FindString(body)
	for _, want := range []string{
		`<button type="button" class="drawer-toggle page-drawer-toggle" aria-expanded="true" ` +
			`aria-controls="page-tools" hidden>Page</button>`,
		`<div id="page-tools" class="page-tools">`,
		`action="/page/rename"`, `action="/page/reorder"`, `action="/page/copy/new"`, `action="/page/delete"`,
		`action="/page/add"`,
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("page drawer: want %s", want)
		}
	}

	// With no pages, adding one is still on offer.
	empty := newHarness(t, nil)
	empty.replaceFile(t, "schema_version: 1\npages: []\n")

	if !regexp.MustCompile(`(?s)<aside id="page-drawer".*There are no pages yet.*action="/page/add"`).
		MatchString(empty.get(t, firstPagePath).Body.String()) {
		t.Error("want the drawer to offer a new page when there are none")
	}

	harness := newCopyKeysHarness(t)

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"page-drawer", "page-tools", "drawer-ready"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	if !strings.Contains(harness.get(t, "/static/editor.css").Body.String(), ".page-drawer.drawer-ready {") {
		t.Error("stylesheet: want the page drawer fixed to the right edge once the script has it")
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
