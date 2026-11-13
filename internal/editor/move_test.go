package editor_test

import (
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/bricks/openurl"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The move form's field names and the route it posts to, named once.
const (
	movePath        = "/key/move"
	toField         = "to"
	actionField     = "action"
	moveAction      = "move"
	copyAction      = "copy"
	firstPageSaved  = "/page?page=0&saved=key"
	discordPreset   = "discord_channel"
	refusedMoveText = "want 422 refusing the move"
	offGridText     = "not on the deck"
	labelField      = "field.label"
	serverIDField   = "field.server_id"
	channelIDField  = "field.channel_id"
)

// moveForm is a move or copy of the starting key at page 0, row 0, column 0.
func moveForm(action string, toPage string, toRow string, toColumn string) url.Values {
	return url.Values{
		pageField: {"0"}, rowField: {"0"}, columnField: {"0"},
		toField: {toPage + "." + toRow + "." + toColumn}, actionField: {action},
	}
}

func keyAtPosition(page config.Page, position layout.Position) (config.Key, bool) {
	for _, key := range page.Keys {
		if key.Position == position {
			return key, true
		}
	}

	return config.Key{Position: position, Kind: "", Parameters: nil, Preset: nil}, false
}

// requireSameKeyElsewhere checks a key kept its kind, parameters and preset
// source exactly, with only its position changed.
func requireSameKeyElsewhere(t *testing.T, original config.Key, moved config.Key, position layout.Position) {
	t.Helper()

	if moved.Position != position {
		t.Fatalf("want the key at %+v, got %+v", position, moved.Position)
	}

	if moved.Kind != original.Kind || !reflect.DeepEqual(moved.Parameters, original.Parameters) ||
		!reflect.DeepEqual(moved.Preset, original.Preset) {
		t.Fatalf("want the key carried over unchanged, got %+v from %+v", moved, original)
	}
}

func TestEditor_KeyPageOffersMoveAndCopy_OnlyForAnExistingKey(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	body := harness.get(t, "/key?page=0&row=0&column=0").Body.String()

	for _, want := range []string{
		`action="/key/move"`,
		`name="action" value="move"`,
		`name="action" value="copy"`,
		`name="to"`,
		`<optgroup label="Page 1: Work">`,
		`<option value="0.0.1">Row 1, column 2</option>`,
		`<option value="0.3.6">Row 4, column 7</option>`,
		"Move or copy this key",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s in the key page", want)
		}
	}

	moveForm := regexp.MustCompile(`(?s)<form[^>]*action="/key/move".*?</form>`).FindString(body)
	if !strings.Contains(moveForm, `name="token" value="`+testToken+`"`) {
		t.Error("want the move form to carry the token")
	}

	if strings.Contains(harness.get(t, "/key?page=0&row=1&column=1").Body.String(), `action="/key/move"`) {
		t.Error("want no move form where there is no key")
	}
}

func TestEditor_MovesAKeyToAFreeTile(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	original, _ := keyAtPosition(harness.onDisk(t).Pages[0], layout.Position{Row: 0, Column: 0})

	recorder := harness.post(t, movePath, moveForm(moveAction, "0", "2", "5"))
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != firstPageSaved {
		t.Fatalf("want a redirect to the page, got %d %q: %s",
			recorder.Code, recorder.Header().Get("Location"), recorder.Body.String())
	}

	page := harness.onDisk(t).Pages[0]
	if _, stillThere := keyAtPosition(page, layout.Position{Row: 0, Column: 0}); stillThere {
		t.Fatal("want the key gone from its old tile")
	}

	moved, found := keyAtPosition(page, layout.Position{Row: 2, Column: 5})
	if !found || len(page.Keys) != 1 {
		t.Fatalf("want exactly the one key, at row 2 column 5, got %+v", page.Keys)
	}

	requireSameKeyElsewhere(t, original, moved, layout.Position{Row: 2, Column: 5})
}

func TestEditor_CopiesAKey_AndTheCopiesAreIndependent(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	original, _ := keyAtPosition(harness.onDisk(t).Pages[0], layout.Position{Row: 0, Column: 0})

	recorder := harness.post(t, movePath, moveForm(copyAction, "0", "1", "3"))
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != firstPageSaved {
		t.Fatalf("want a redirect to the page, got %d: %s", recorder.Code, recorder.Body.String())
	}

	page := harness.onDisk(t).Pages[0]
	_, sourceFound := keyAtPosition(page, layout.Position{Row: 0, Column: 0})
	copied, copyFound := keyAtPosition(page, layout.Position{Row: 1, Column: 3})

	if !sourceFound || !copyFound || len(page.Keys) != 2 {
		t.Fatalf("want both keys, got %+v", page.Keys)
	}

	requireSameKeyElsewhere(t, original, copied, layout.Position{Row: 1, Column: 3})

	// Editing the copy must leave the original as it was.
	saveRecorder := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"3"}, typeField: {openURLType},
		urlField: {"https://example.test/changed"}, labelField: {"Changed"},
	})
	if saveRecorder.Code != http.StatusSeeOther {
		t.Fatalf("want the copy saved, got %d", saveRecorder.Code)
	}

	source, _ := keyAtPosition(harness.onDisk(t).Pages[0], layout.Position{Row: 0, Column: 0})

	parameters, isOpenURL := source.Parameters.(*openurl.Parameters)
	if !isOpenURL || parameters.Label != "Docs" || parameters.URL != "https://example.test/docs" {
		t.Fatalf("want the original unchanged, got %+v", source.Parameters)
	}
}

func TestEditor_MovesAPresetKeyAcrossPages_KeepingItsPreset(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, step := range []struct {
		path string
		form url.Values
	}{
		{"/page/add", url.Values{nameField: {"Spare"}}},
		{"/key", url.Values{
			pageField: {"0"}, rowField: {"0"}, columnField: {"4"}, typeField: {"preset:" + discordPreset},
			serverIDField: {"123"}, channelIDField: {"456"}, labelField: {"Party"},
		}},
	} {
		if recorder := harness.post(t, step.path, step.form); recorder.Code != http.StatusSeeOther {
			t.Fatalf("%s: want a redirect, got %d: %s", step.path, recorder.Code, recorder.Body.String())
		}
	}

	original, _ := keyAtPosition(harness.onDisk(t).Pages[0], layout.Position{Row: 0, Column: 4})
	if original.Preset == nil || original.Preset.Name != discordPreset {
		t.Fatalf("want a preset key to move, got %+v", original)
	}

	form := moveForm(moveAction, "1", "2", "2")
	form.Set(columnField, "4")

	recorder := harness.post(t, movePath, form)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/page?page=1&saved=key" {
		t.Fatalf("want a redirect to the destination page, got %d %q", recorder.Code, recorder.Header().Get("Location"))
	}

	configuration := harness.onDisk(t)
	if _, stillThere := keyAtPosition(configuration.Pages[0], layout.Position{Row: 0, Column: 4}); stillThere {
		t.Fatal("want the key gone from the first page")
	}

	moved, found := keyAtPosition(configuration.Pages[1], layout.Position{Row: 2, Column: 2})
	if !found {
		t.Fatalf("want the key on the second page, got %+v", configuration.Pages[1].Keys)
	}

	requireSameKeyElsewhere(t, original, moved, layout.Position{Row: 2, Column: 2})
}

func TestEditor_RefusesAMoveOrCopyToABadDestination(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		form url.Values
		want string
	}{
		{"occupied", moveForm(copyAction, "0", "1", "1"), "already has a key"},
		{"back navigation position", moveForm(moveAction, "0", "3", "0"), "reserved for paging"},
		{"forward navigation position", moveForm(copyAction, "0", "3", "7"), "reserved for paging"},
		{"row off the grid", moveForm(moveAction, "0", "4", "1"), offGridText},
		{"column off the grid", moveForm(moveAction, "0", "1", "8"), offGridText},
		{"page that does not exist", moveForm(moveAction, "5", "1", "1"), offGridText},
		{"same position", moveForm(moveAction, "0", "0", "0"), "where the key already is"},
		{"no key at the source", url.Values{
			pageField: {"0"}, rowField: {"2"}, columnField: {"2"},
			toField: {"0.1.2"}, actionField: {moveAction},
		}, "no key here"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, nil)

			// The occupied case needs a second key in the way.
			if testCase.name == "occupied" {
				harness.post(t, "/key", url.Values{
					pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, typeField: {openURLType},
					urlField: {"https://example.test/other"},
				})
			}

			before := harness.onDisk(t)

			recorder := harness.post(t, movePath, testCase.form)
			body := recorder.Body.String()

			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("%s, got %d: %s", refusedMoveText, recorder.Code, body)
			}

			alert := regexp.MustCompile(`(?s)<div class="errors" role="alert">.*?</div>`).FindString(body)
			if !strings.Contains(alert, "Error: ") || !strings.Contains(alert, testCase.want) {
				t.Fatalf("want an alert starting Error: and saying %q, got %q", testCase.want, alert)
			}

			if !strings.Contains(body, "<title>Error: ") {
				t.Error("want the title to start Error:")
			}

			if !reflect.DeepEqual(harness.onDisk(t), before) {
				t.Error("want the file unchanged after a refused move")
			}
		})
	}
}

func TestEditor_RefusedMoveFocusesTheDestination(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).post(t, movePath, moveForm(moveAction, "0", "3", "0")).Body.String()

	focused := regexp.MustCompile(`<(?:input|select)[^>]*\sautofocus[^>]*>`).FindAllString(body, -1)
	if len(focused) != 1 || !strings.Contains(focused[0], `name="to"`) {
		t.Fatalf("want only the destination list focused, got %q", focused)
	}
}

func TestEditor_RefusesAnUnknownMoveAction(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, movePath, moveForm("swap", "0", "1", "1"))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for an unknown action, got %d", recorder.Code)
	}
}

func TestEditor_MoveWithoutTheTokenIsRefusedByTheGuard(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.postRaw(t, movePath, moveForm(moveAction, "0", "1", "1"), func(request *http.Request) {
		request.Header.Set("Origin", testOrigin)
	})

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("want 403 without the token, got %d", recorder.Code)
	}

	if _, moved := keyAtPosition(harness.onDisk(t).Pages[0], layout.Position{Row: 1, Column: 1}); moved {
		t.Fatal("want nothing moved without the token")
	}
}

func TestEditor_MoveWaitsWhileTheFileOnDiskIsInvalid(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	breakConfigurationFile(t, harness)

	recorder := harness.post(t, movePath, moveForm(moveAction, "0", "1", "1"))
	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), "Fix the file") {
		t.Fatalf("%s while the file is invalid, got %d", refusedMoveText, recorder.Code)
	}
}

// breakConfigurationFile replaces the file with a hand edit the store refuses.
func breakConfigurationFile(t *testing.T, harness *editorHarness) {
	t.Helper()

	writeError := os.WriteFile(harness.configPath, []byte("schema_version: 1\npages: [\n"), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}
}

// TestEditor_MoveListOffersOnlyFreeTiles: the author found the form offering
// the paging tiles, which the server then refused (evaluation, deck testing).
// The one destination list now holds every free tile on every page and nothing
// else: no paging tile, no occupied tile and not the key's own place.
func TestEditor_MoveListOffersOnlyFreeTiles(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	addError := harness.store.Save(func() config.Configuration {
		configuration, _ := harness.store.OnDisk()
		configuration.Pages = append(configuration.Pages, config.Page{Name: "Games", Application: "", Keys: nil})

		return configuration
	}())
	if addError != nil {
		t.Fatal(addError)
	}

	body := harness.get(t, "/key?page=0&row=0&column=0").Body.String()
	offered := regexp.MustCompile(`<option value="(\d+\.\d+\.\d+)"`).FindAllStringSubmatch(body, -1)

	// Two pages of 32 tiles, less two paging tiles on each and the key's own.
	if len(offered) != 2*(32-2)-1 {
		t.Fatalf("want 59 free tiles offered, got %d", len(offered))
	}

	for _, refused := range []string{"0.0.0", "0.3.0", "0.3.7", "1.3.0", "1.3.7"} {
		if strings.Contains(body, `<option value="`+refused+`"`) {
			t.Errorf("want %s left out of the list", refused)
		}
	}

	if !strings.Contains(body, `<optgroup label="Page 2: Games">`) {
		t.Error("want the second page's tiles grouped under its name")
	}
}

func TestEditor_RefusesAMalformedDestination(t *testing.T) {
	t.Parallel()

	form := moveForm(moveAction, "0", "0", "1")
	form.Set(toField, "not-a-tile")

	recorder := newHarness(t, nil).post(t, movePath, form)
	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), offGridText) {
		t.Fatalf("%s for a malformed destination, got %d", refusedMoveText, recorder.Code)
	}
}
