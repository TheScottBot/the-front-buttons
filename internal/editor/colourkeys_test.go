package editor_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The colouring route and its fields.
const (
	colourKeysPath   = "/keys/colour"
	backgroundField  = "background"
	clearColourField = "clear"
	red              = "#FF0000"
)

// backgroundAt is the background of the key at a position on a page.
func backgroundAt(t *testing.T, page config.Page, position layout.Position) string {
	t.Helper()

	key, found := keyAtPosition(page, position)
	if !found {
		t.Fatalf("want a key at %v", position)
	}

	appearance, _ := brick.AppearanceOf(key.Parameters)

	return appearance.Background
}

// TestEditor_ColoursTheTickedKeys: select mode gives every ticked key the
// background chosen, leaving the rest alone, and clears it again.
func TestEditor_ColoursTheTickedKeys(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	cut, paste, music := layout.Position{Row: 0, Column: 1}, layout.Position{Row: 0, Column: 2},
		layout.Position{Row: 1, Column: 1}

	response := harness.post(t, colourKeysPath, url.Values{
		pageField: {workPage}, tileField: {cutTile, pasteTile}, backgroundField: {red},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/page?page=0&saved=coloured" {
		t.Fatalf("want a redirect to the page, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}

	work := harness.onDisk(t).Pages[0]
	if backgroundAt(t, work, cut) != red || backgroundAt(t, work, paste) != red || backgroundAt(t, work, music) != "" {
		t.Fatalf("want only the ticked keys red, got %+v", work.Keys)
	}

	harness.post(t, colourKeysPath, url.Values{
		pageField: {workPage}, tileField: {cutTile}, backgroundField: {red}, clearColourField: {"1"},
	})

	work = harness.onDisk(t).Pages[0]
	if backgroundAt(t, work, cut) != "" || backgroundAt(t, work, paste) != red {
		t.Fatalf("want the cleared key back to no colour, the other kept, got %+v", work.Keys)
	}

	if !strings.Contains(harness.get(t, "/page?page=0&saved=coloured").Body.String(), "Coloured.") {
		t.Error("want the page to say the keys were coloured")
	}
}

// TestEditor_ColoursAPresetKeyThroughItsFields: a preset key keeps its colour
// among the preset's fields, which is what the file records, as well as in
// the parameters it expands to.
func TestEditor_ColoursAPresetKeyThroughItsFields(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        preset: discord_channel
        fields:
          server_id: "123456789"
          channel_id: "987654321"
          label: Party
`)

	response := harness.post(t, colourKeysPath, url.Values{pageField: {"0"}, tileField: {"0.0"}, backgroundField: {red}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the colour saved, got %d: %s", response.Code, response.Body.String())
	}

	key := harness.onDisk(t).Pages[0].Keys[0]
	fields, _ := brick.AppearanceOf(key.Preset.Fields)

	if backgroundAt(t, harness.onDisk(t).Pages[0], layout.Position{Row: 0, Column: 0}) != red || fields.Background != red {
		t.Fatalf("want the preset key red in its fields and parameters, got %+v", key)
	}
}

// TestEditor_RefusesABadColourOrNothingTicked: a colour must be written
// #RRGGBB, and something must be ticked; either refusal changes nothing.
func TestEditor_RefusesABadColourOrNothingTicked(t *testing.T) {
	t.Parallel()

	for _, refused := range []struct {
		tiles  []string
		colour string
		want   string
	}{
		{[]string{cutTile}, "red", "must be colour"},
		{nil, red, "no keys were ticked"},
	} {
		harness := newCopyKeysHarness(t)
		before := harness.onDisk(t)

		response := harness.post(t, colourKeysPath, url.Values{
			pageField: {workPage}, tileField: refused.tiles, backgroundField: {refused.colour},
		})
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), refused.want) {
			t.Errorf("%+v: want refused with %q, got %d", refused, refused.want, response.Code)
		}

		requireUnchanged(t, harness, before)
	}
}

// TestEditor_SelectModeOffersAColour: select mode's form has a colour to
// give the ticked keys and a way to clear it.
func TestEditor_SelectModeOffersAColour(t *testing.T) {
	t.Parallel()

	body := newCopyKeysHarness(t).get(t, "/page?page=0&select=1").Body.String()

	for _, want := range []string{
		`<label for="select-colour">Background colour</label>`,
		`<input type="color" id="select-colour" name="background" value="#000000">`,
		`<button type="submit" formaction="/keys/colour" formnovalidate>Colour the ticked keys</button>`,
		`<button type="submit" formaction="/keys/colour" formnovalidate name="clear" value="1">Clear their colour</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("select mode: want %s", want)
		}
	}
}
