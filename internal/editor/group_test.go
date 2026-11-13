package editor_test

import (
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The group form's route and the field names of its parts, named once.
const (
	groupPath      = "/group"
	characterField = "character.field.character_id"
	campaignField  = "roll20.field.campaign_id"
	serverField    = "discord.field.server_id"
	channelField   = "discord.field.channel_id"
	orderField     = "order"
	sizeField      = "size"
	threeHigh      = "1x3"
)

// groupForm is a three key group dropped at row 2, column 3 of the first page,
// in the order given.
func groupForm(order ...string) url.Values {
	return url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"2"}, sizeField: {threeHigh},
		orderField:     order,
		characterField: {"143332240"}, "character.field.label": {"Yeku"},
		campaignField: {"19929142"},
		serverField:   {"111"}, channelField: {"222"},
	}
}

// TestEditor_GroupFormAsksForEveryPartInOrder: the author asked for a 1x2
// and a 1x3 brick that place a D&D Beyond sheet and a Roll20 campaign, and a
// Discord channel for the 1x3, down a column, each set up in one form whose
// parts can be put in any order.
func TestEditor_GroupFormAsksForEveryPartInOrder(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	two := harness.get(t, "/group?page=0&row=1&column=2&size=1x2").Body.String()
	three := harness.get(t, "/group?page=0&row=1&column=2&size=1x3").Body.String()

	for _, want := range []string{
		`<div id="key-editor">`,
		`<form method="post" action="/group"`,
		`name="` + characterField + `"`,
		`name="` + campaignField + `"`,
		`<input type="hidden" name="order" value="character">`,
		`class="move-up"`,
	} {
		if !strings.Contains(three, want) {
			t.Errorf("1x3: want %s", want)
		}
	}

	for _, want := range []string{`name="` + serverField + `"`, `name="` + channelField + `"`} {
		if !strings.Contains(three, want) {
			t.Errorf("1x3: want %s", want)
		}

		if strings.Contains(two, want) {
			t.Errorf("1x2: want no Discord part, found %s", want)
		}
	}

	character := strings.Index(three, `value="character"`)
	roll20 := strings.Index(three, `value="roll20"`)
	discord := strings.Index(three, `value="discord"`)

	if character >= roll20 || roll20 >= discord {
		t.Error("want the parts first in the order sheet, Roll20, Discord")
	}

	if got := harness.get(t, "/group?page=0&row=1&column=2&size=3x3"); got.Code != http.StatusBadRequest {
		t.Errorf("want an unknown size refused, got %d", got.Code)
	}
}

// TestEditor_GroupPlacesItsKeysDownTheColumnInTheChosenOrder: saved, each
// part becomes a key of its own, a preset key the editor can re-edit, top to
// bottom in the order the form gave.
func TestEditor_GroupPlacesItsKeysDownTheColumnInTheChosenOrder(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	response := harness.post(t, groupPath, groupForm("roll20", "discord", "character"))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != firstPageSaved {
		t.Fatalf("want the group saved, got %d: %s", response.Code, response.Body.String())
	}

	page := harness.onDisk(t).Pages[0]

	for row, wantPreset := range []string{"roll20_campaign", "discord_channel", "dndbeyond_character"} {
		key, found := keyAtPosition(page, layout.Position{Row: row + 1, Column: 2})
		if !found || key.Preset == nil || key.Preset.Name != wantPreset {
			t.Errorf("row %d: want a %s key, got %+v", row+2, wantPreset, key)
		}
	}
}

// TestEditor_GroupRefusesTilesItCannotHave: every tile the group needs must
// be on the deck, free and not kept for paging, or nothing is placed; a part
// missing its ID, or an order that is not the group's parts, is refused too.
func TestEditor_GroupRefusesTilesItCannotHave(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	before, _ := os.ReadFile(harness.configPath)

	refusals := map[string]url.Values{
		"a key in the way": withValues(groupForm("character", "roll20", "discord"), rowField, "0", columnField, "0"),
		"off the bottom":   withValues(groupForm("character", "roll20", "discord"), rowField, "2"),
		"onto paging":      withValues(groupForm("character", "roll20", "discord"), rowField, "1", columnField, "7"),
		"a missing ID":     withValues(groupForm("character", "roll20", "discord"), campaignField, ""),
		"a part twice":     groupForm("character", "character", "discord"),
		"a part left out":  groupForm("character", "roll20"),
		"a part not in it": groupForm("character", "roll20", "teams"),
	}

	for name, form := range refusals {
		response := harness.post(t, groupPath, form)
		if response.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: want 422, got %d", name, response.Code)

			continue
		}

		body := response.Body.String()
		if !strings.Contains(body, "Error: ") || !strings.Contains(body, `id="key-editor"`) {
			t.Errorf("%s: want the form again with why", name)
		}
	}

	after, _ := os.ReadFile(harness.configPath)
	if string(after) != string(before) {
		t.Error("want the file untouched by every refused group")
	}

	blocked := harness.post(t, groupPath,
		withValues(groupForm("character", "roll20", "discord"), rowField, "0", columnField, "0"))
	if !strings.Contains(blocked.Body.String(), "would cover a key at row 1, column 1") {
		t.Error("want the tile in the way named")
	}

	missing := harness.post(t, groupPath, withValues(groupForm("character", "roll20", "discord"), campaignField, ""))
	if !strings.Contains(missing.Body.String(), `value="143332240"`) {
		t.Error("want a refused form to keep what was typed")
	}
}

// TestEditor_DrawerHoldsADrawerForEachSize: the bricks drawer holds drawers
// of its own: 1x1 with every brick and preset, and 1x2 and 1x3 with the
// groups, each marked with how many tiles it needs.
func TestEditor_DrawerHoldsADrawerForEachSize(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	body := harness.get(t, "/page?page=0").Body.String()

	for _, want := range []string{
		`<summary>1x1</summary>`,
		`<summary>1x2</summary>`,
		`<summary>1x3</summary>`,
		`<summary>2x2</summary>`,
		`<li draggable="true" data-group="1x2" data-width="1" data-height="2">`,
		`<li draggable="true" data-group="1x3" data-width="1" data-height="3">`,
		`<li draggable="true" data-group="2x2" data-width="2" data-height="2">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s in the drawer", want)
		}
	}

	if !regexp.MustCompile(`(?s)<summary>1x1</summary>.*data-type="kind:open_url".*<summary>1x2</summary>`).
		MatchString(body) {
		t.Error("want the single bricks in the 1x1 drawer")
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"stack-target", "/group?"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}
}

// withValues returns a copy of a form with some fields set, in name, value
// pairs.
func withValues(form url.Values, pairs ...string) url.Values {
	changed := url.Values{}
	for name, values := range form {
		changed[name] = append([]string(nil), values...)
	}

	for index := 0; index+1 < len(pairs); index += 2 {
		changed.Set(pairs[index], pairs[index+1])
	}

	return changed
}

// spotifySquare is a 2x2 dropped at row 2, column 2 of the first page.
func spotifySquare(row string, column string, order ...string) url.Values {
	return url.Values{
		pageField: {"0"}, rowField: {row}, columnField: {column}, sizeField: {"2x2"}, orderField: order,
	}
}

// TestEditor_SpotifySquareFillsTwoByTwoInReadingOrder: the author's first
// 2x2 is Spotify: the album, play or pause, previous track and next track,
// left to right then top to bottom, in whatever order the form gives.
func TestEditor_SpotifySquareFillsTwoByTwoInReadingOrder(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	form := harness.get(t, "/group?page=0&row=1&column=1&size=2x2").Body.String()
	for _, part := range []string{"album", "play", "previous", "next"} {
		if !strings.Contains(form, `<input type="hidden" name="order" value="`+part+`">`) {
			t.Errorf("want the %s part in the form", part)
		}
	}

	response := harness.post(t, groupPath, spotifySquare("1", "1", "album", "play", "previous", "next"))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the square saved, got %d: %s", response.Code, response.Body.String())
	}

	page := harness.onDisk(t).Pages[0]

	for position, wantKind := range map[layout.Position]string{
		{Row: 1, Column: 1}: "spotify_album",
		{Row: 1, Column: 2}: "spotify_play_pause",
		{Row: 2, Column: 1}: "spotify_previous",
		{Row: 2, Column: 2}: "spotify_next",
	} {
		if key, found := keyAtPosition(page, position); !found || key.Kind != wantKind {
			t.Errorf("%+v: want %s, got %+v", position, wantKind, key)
		}
	}

	for name, refused := range map[string]url.Values{
		"off the right edge": spotifySquare("1", "7", "album", "play", "previous", "next"),
		"onto paging":        spotifySquare("2", "0", "album", "play", "previous", "next"),
	} {
		if got := harness.post(t, groupPath, refused); got.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: want refused, got %d", name, got.Code)
		}
	}
}
