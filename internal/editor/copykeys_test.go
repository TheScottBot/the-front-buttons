package editor_test

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The copy of ticked keys: its route and fields, and the pages of its
// configuration by their place in the file.
const (
	copyKeysPath    = "/keys/copy"
	tileField       = "tile"
	workPage        = "0"
	golandScreen    = "1"
	pycharmScreen   = "2"
	playPage        = "3"
	firstPageCopied = "/page?page=0&saved=copied"
	cutTile         = "0.1"
	musicTile       = "1.1"
)

// copyKeysConfiguration has Work, with keys at row 1 columns 1 to 3 and at
// row 2 column 2; GoLand's screen, with its build key at row 2 column 2;
// PyCharm's screen, empty; and Play, empty.
const copyKeysConfiguration = `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: open_url
        parameters:
          url: https://example.test/corner
          label: Corner
      - row: 0
        column: 1
        kind: open_url
        parameters:
          url: https://example.test/cut
          label: Cut
      - row: 0
        column: 2
        kind: open_url
        parameters:
          url: https://example.test/paste
          label: Paste
      - row: 1
        column: 1
        kind: open_url
        parameters:
          url: https://example.test/music
          label: Music
  - name: GoLand
    application: com.jetbrains.goland
    keys:
      - row: 1
        column: 1
        kind: jetbrains_build
        parameters:
          label: Build
  - name: PyCharm
    application: com.jetbrains.pycharm
    keys: []
  - name: Play
    keys: []
`

func newCopyKeysHarness(t *testing.T) *editorHarness {
	t.Helper()

	harness := newHarness(t, nil)
	harness.replaceFile(t, copyKeysConfiguration)

	return harness
}

// copyKeys posts a copy of the ticked tiles of a page onto the pages given.
func copyKeys(
	t *testing.T, harness *editorHarness, page string, tiles []string, targets []string,
) (int, string) {
	t.Helper()

	response := harness.post(t, copyKeysPath, url.Values{pageField: {page}, tileField: tiles, toField: targets})

	return response.Code, response.Body.String()
}

// requireUnchanged fails unless the file on disk is as it was before.
func requireUnchanged(t *testing.T, harness *editorHarness, before config.Configuration) {
	t.Helper()

	if after := harness.onDisk(t); !reflect.DeepEqual(after, before) {
		t.Errorf("want nothing copied anywhere, got %+v", after.Pages)
	}
}

// TestEditor_CopiesTickedKeysOntoSeveralPagesInPlace: the ticked keys of a
// page are copied onto every page ticked, app screens among them, each on the
// tile it came from.
func TestEditor_CopiesTickedKeysOntoSeveralPagesInPlace(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, copyKeysPath, url.Values{
		pageField: {workPage}, tileField: {cutTile, "0.2"}, toField: {pycharmScreen, playPage},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != firstPageCopied {
		t.Fatalf("want a redirect to the page copied from, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}

	saved := harness.onDisk(t)
	originals := map[layout.Position]config.Key{}

	for _, key := range saved.Pages[0].Keys {
		originals[key.Position] = key
	}

	for _, target := range []int{2, 3} {
		copied := saved.Pages[target].Keys
		if len(copied) != 2 {
			t.Fatalf("want the two ticked keys on %s, got %+v", saved.Pages[target].Name, copied)
		}

		for _, key := range copied {
			original := originals[key.Position]
			if key.Position.Row != 0 || !reflect.DeepEqual(key.Parameters, original.Parameters) {
				t.Errorf("want %+v copied in place, got %+v", original, key)
			}
		}
	}

	if len(saved.Pages[0].Keys) != 4 {
		t.Errorf("want the page copied from left as it was, got %+v", saved.Pages[0].Keys)
	}

	notice := harness.get(t, firstPageCopied).Body.String()
	requireBody(t, notice, "Copied. The deck shows the change within a second.")
}

// TestEditor_RefusesACopyThatClashes_NamingEveryTile: if any ticked key would
// land on a key already there, on any page ticked, nothing is copied anywhere
// and every clash is named with its page.
func TestEditor_RefusesACopyThatClashes_NamingEveryTile(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	code, body := copyKeys(t, harness, workPage, []string{cutTile, musicTile}, []string{playPage, golandScreen})
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("want the copy refused, got %d", code)
	}

	requireBody(t, body, "Error: nothing was copied", "the GoLand screen, row 2, column 2, already has a key")
	refuseBody(t, body, "Play, row")
	requireUnchanged(t, harness, before)

	// What was ticked stays ticked, to change and try again.
	requireBody(t, body, `value="0.1" form="copy-keys" checked>`, `value="1.1" form="copy-keys" checked>`,
		`name="to" value="3" checked>`, `name="to" value="1" checked>`)
	refuseBody(t, body, `name="to" value="2" checked>`)
}

// TestEditor_RefusesACopyOntoAnAppScreensIconTile: a key from a regular
// page's top left cannot be copied onto an app screen, where that tile is kept
// for the application's icon; nothing is copied, even onto pages it fits.
func TestEditor_RefusesACopyOntoAnAppScreensIconTile(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	code, body := copyKeys(t, harness, workPage, []string{"0.0", cutTile}, []string{playPage, pycharmScreen})
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("want the copy refused, got %d", code)
	}

	requireBody(t, body, "the PyCharm screen, row 1, column 1, is kept for the application&#39;s icon")
	requireUnchanged(t, harness, before)
}

// TestEditor_CopiesAnAppsOwnKeysOntoItsOwnScreensOnly: GoLand's build key
// copies onto PyCharm's screen, which shares its bricks, and is refused, by
// name, on a regular page.
func TestEditor_CopiesAnAppsOwnKeysOntoItsOwnScreensOnly(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	code, body := copyKeys(t, harness, golandScreen, []string{musicTile}, []string{pycharmScreen, playPage})
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("want the copy onto Play refused, got %d", code)
	}

	requireBody(t, body,
		"Play, row 2, column 2, cannot hold jetbrains_build, which belongs on its application&#39;s screen")
	requireUnchanged(t, harness, before)

	if code, body = copyKeys(t, harness, golandScreen, []string{musicTile}, []string{pycharmScreen}); code !=
		http.StatusSeeOther {
		t.Fatalf("want the copy onto PyCharm made, got %d: %s", code, body)
	}

	if copied := harness.onDisk(t).Pages[2].Keys; len(copied) != 1 || copied[0].Kind != "jetbrains_build" {
		t.Errorf("want GoLand's build key on PyCharm's screen, got %+v", copied)
	}
}

// TestEditor_RefusesACopyWithNothingTicked: a copy needs a key ticked, a page
// ticked to copy onto, keys still where they were ticked, and no page copied
// onto itself.
func TestEditor_RefusesACopyWithNothingTicked(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	cases := []struct {
		name    string
		tiles   []string
		targets []string
		want    string
	}{
		{"no key", nil, []string{playPage}, "no keys were ticked"},
		{"no page", []string{cutTile}, nil, "no page was ticked to copy onto"},
		{"empty tile", []string{"2.2"}, []string{playPage}, "there is no key at row 3, column 3 to copy"},
		{"itself", []string{cutTile}, []string{workPage}, "a page cannot be copied onto itself"},
		{"no such page", []string{cutTile}, []string{"9"}, "that page no longer exists"},
	}

	for _, refused := range cases {
		code, body := copyKeys(t, harness, workPage, refused.tiles, refused.targets)
		if code != http.StatusUnprocessableEntity {
			t.Errorf("%s: want the copy refused, got %d", refused.name, code)
		}

		requireBody(t, body, refused.want)
	}

	requireUnchanged(t, harness, before)
}

// TestEditor_SelectModeTicksKeysAndPages: the grid has a link into select
// mode, where each key is a check box rather than a link to its form, beside
// a check box for every other page and app screen to copy onto.
func TestEditor_SelectModeTicksKeysAndPages(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	normal := harness.get(t, "/page?page=0").Body.String()
	requireBody(t, normal, `<a href="/page?page=0&amp;select=1">Select keys to copy</a>`)
	refuseBody(t, normal, `name="tile"`)

	selecting := harness.get(t, "/page?page=0&select=1").Body.String()
	requireBody(t, selecting,
		`<input type="checkbox" id="tile-0-1" name="tile" value="0.1" form="copy-keys">`,
		`<label for="tile-0-1" aria-label="Row 1, column 2: Cut (open_url)">Cut`,
		`<form method="post" action="/keys/copy" id="copy-keys"`,
		`<input type="checkbox" id="copy-to-1" name="to" value="1"> <label for="copy-to-1">GoLand screen</label>`,
		`<label for="copy-to-3">Page 2: Play</label>`,
		`<a href="/page?page=0" class="select-done">Done selecting</a>`,
		// A fixed click area over the key, which moves when ticked.
		`<label class="select-hit" for="tile-0-1" aria-hidden="true"></label>`,
	)
	refuseBody(t, selecting, `href="/key?page=0&amp;row=0&amp;column=1"`, `value="2.2" form="copy-keys"`,
		`id="copy-to-0"`)
}

// TestEditor_SelectModeTicksTheKeysItsAddressNames: select mode opened with
// tiles in its address, as Shift and a click on a key opens it, starts with
// those keys ticked; a tile with no key to tick is left alone.
func TestEditor_SelectModeTicksTheKeysItsAddressNames(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	body := harness.get(t, "/page?page=0&select=1&tile=0.1&tile=1.1&tile=2.2&tile=bad").Body.String()
	requireBody(t, body, `value="0.1" form="copy-keys" checked>`, `value="1.1" form="copy-keys" checked>`)
	refuseBody(t, body, `value="0.2" form="copy-keys" checked>`, `value="2.2" form="copy-keys"`)
}

// TestEditor_ShiftClickHintsWaitForTheScript: the hints for Shift and a click,
// and for Escape, stay hidden until the script, which handles them, shows
// them.
func TestEditor_ShiftClickHintsWaitForTheScript(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	requireBody(t, harness.get(t, "/page?page=0").Body.String(),
		`<p class="select-help" hidden>Or hold Shift and click a key to start selecting with it.</p>`)

	// Escape leaves select mode by the same link as Done selecting.
	requireBody(t, harness.get(t, "/page?page=0&select=1").Body.String(),
		`<a href="/page?page=0" class="select-done">Done selecting</a>`,
		`<span class="select-help" hidden>, or press Escape</span>`)
}
