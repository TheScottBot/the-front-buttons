package editor_test

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestEditor_PageHoldsAHiddenBrickDrawer: the author asked for a drawer in
// the left gutter that opens on a click and offers bricks to drag onto an
// empty tile. It lists every kind and preset, as the key form does; dragging
// needs the script, so it stays hidden until the script shows it, and the
// grid's empty tiles stay the way to add a key from the keyboard.
func TestEditor_PageHoldsAHiddenBrickDrawer(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	body := harness.get(t, firstPagePath).Body.String()

	drawer := regexp.MustCompile(`(?s)<aside id="brick-drawer"[^>]*>.*?</aside>`).FindString(body)
	if drawer == "" {
		t.Fatal("want the brick drawer on the page")
	}

	for _, want := range []string{
		`<aside id="brick-drawer" class="brick-drawer" aria-label="Bricks" hidden>`,
		`<button type="button" class="drawer-toggle" aria-expanded="false" aria-controls="brick-list">`,
		`<div id="brick-list" class="drawer-panel" data-chars="`,
		`<li draggable="true" data-type="kind:open_url">`,
		`<li draggable="true" data-type="preset:discord_channel">`,
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("drawer: want %s", want)
		}
	}

	harness.replaceFile(t, "schema_version: 1\npages: []\n")

	if strings.Contains(harness.get(t, firstPagePath).Body.String(), "brick-drawer") {
		t.Error("want no drawer with no page to drop onto")
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"application/x-front-buttons-brick", "placing", "carrying", "brick-drawer"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	// While a brick is carried, a tile it cannot go on shows a cross.
	stylesheet := harness.get(t, "/static/editor.css").Body.String()
	if !strings.Contains(stylesheet, `.grid.placing td[data-state="key"]`) ||
		!strings.Contains(stylesheet, `.grid.placing td[data-state="reserved"]`) {
		t.Error("stylesheet: want a cross on keys and reserved tiles while a brick is carried")
	}

	// And the open panel lets the drag through to tiles it covers.
	if !regexp.MustCompile(`(?s)\.brick-drawer\.carrying \{[^}]*pointer-events: none`).MatchString(stylesheet) {
		t.Error("stylesheet: want the drawer to let a carried brick through to the tiles")
	}
}

// TestEditor_DrawerHasASearchBox: the drawer opens with a labelled search box
// that narrows its bricks as one types, and a line for when none match, hidden
// until then. Like the drawer, it works through the script.
func TestEditor_DrawerHasASearchBox(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	for _, want := range []string{
		`<label for="brick-search">Search bricks</label>`,
		`<input type="search" id="brick-search" class="drawer-search" autocomplete="off" spellcheck="false">`,
		`<p class="drawer-empty" role="status" hidden>No bricks match.</p>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("drawer: want %s", want)
		}
	}
}

// TestEditor_DrawerSaysWhatEachBrickAsksFor: a brick in the drawer opens to
// say what it is: a brick its kind and the fields it asks for, by their own
// descriptions, leaving out the appearance every key shares; a preset what it
// expands into.
func TestEditor_DrawerSaysWhatEachBrickAsksFor(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	openURL := regexp.MustCompile(`(?s)<li draggable="true" data-type="kind:open_url">.*?</li>`).FindString(body)
	for _, want := range []string{
		`<details class="brick-about"><summary>Brick: open_url</summary>`,
		`<p>The open_url brick. It asks for:</p>`,
		`<li>url, required: `,
	} {
		if !strings.Contains(openURL, want) {
			t.Errorf("open_url: want %s, got %s", want, openURL)
		}
	}

	if strings.Contains(openURL, "<li>label") {
		t.Errorf("open_url: want the shared appearance left out, got %s", openURL)
	}

	discord := regexp.MustCompile(`(?s)<li draggable="true" data-type="preset:discord_channel">.*?</details>`).
		FindString(body)
	if !strings.Contains(discord, `<p>A preset of the open_url brick, to open_server. It asks for:</p>`) {
		t.Errorf("discord_channel: want what it expands into, got %s", discord)
	}
}

// TestEditor_DrawerIsSizedForItsLongestName: the drawer carries the length of
// the longest name it can show, in any section, open or closed, so it is one
// width however its sections open; on an app screen its own bricks count too.
func TestEditor_DrawerIsSizedForItsLongestName(t *testing.T) {
	t.Parallel()

	for _, path := range []string{firstPagePath, "/page?page=" + golandScreenIndex} {
		body := newAppScreenHarness(t).get(t, path).Body.String()

		chars := regexp.MustCompile(`<div id="brick-list" class="drawer-panel" data-chars="(\d+)" hidden>`).
			FindStringSubmatch(body)
		if chars == nil {
			t.Fatalf("%s: want the drawer to carry its longest name's length", path)
		}

		longest := 0

		names := regexp.MustCompile(`<li draggable="true"[^>]*>(?:<details class="brick-about"><summary>)?([^<]+)<`)
		for _, name := range names.FindAllStringSubmatch(body, -1) {
			longest = max(longest, utf8.RuneCountInString(html.UnescapeString(name[1])))
		}

		if chars[1] != strconv.Itoa(longest) {
			t.Errorf("%s: want %d characters, the longest name, got %s", path, longest, chars[1])
		}
	}
}

// TestEditor_BOpensTheBrickDrawer: B, as / does for In use, opens the brick
// drawer with its search ready, and the shortcuts sheet names both.
func TestEditor_BOpensTheBrickDrawer(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	if !strings.Contains(harness.get(t, "/static/editor.js").Body.String(), `event.key.toLowerCase() !== "b"`) {
		t.Error("script: want B to open the brick drawer")
	}

	body := harness.get(t, firstPagePath).Body.String()
	for _, want := range []string{
		`<tr><th scope="row">Open Bricks</th><td>B</td></tr>`, `<tr><th scope="row">Open In use</th><td>/</td></tr>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("shortcuts: want %s", want)
		}
	}
}
