package editor_test

import (
	"bytes"
	"fmt"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openurl"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

func TestEditor_ShowsTheGrid_WithTheNavigationKeysReserved(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	for _, want := range []string{
		`aria-label="Row 4, column 1: reserved for paging back"`,
		`aria-label="Row 4, column 8: reserved for paging forward"`,
		`href="/key?page=0&amp;row=0&amp;column=0"`,
		`Docs`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s in the page", want)
		}
	}
}

// formFieldsFor fetches a key form for a type and returns its body.
func formFieldsFor(t *testing.T, harness *editorHarness, typeChoice string) string {
	t.Helper()

	recorder := harness.get(t, "/key?page=0&row=1&column=1&type="+url.QueryEscape(typeChoice))
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s: want 200, got %d", typeChoice, recorder.Code)
	}

	return recorder.Body.String()
}

func requireFieldsInForm(t *testing.T, typeChoice string, body string, schema brick.Schema) {
	t.Helper()

	for _, field := range schema.Fields() {
		control := regexp.MustCompile(`<(input|select)\b[^>]*\bname="field\.` + field.Name + `"[^>]*>`).FindString(body)
		if control == "" {
			t.Errorf("%s: no control for field %s", typeChoice, field.Name)

			continue
		}

		identifier := regexp.MustCompile(`\bid="([^"]+)"`).FindStringSubmatch(control)
		if identifier == nil || !strings.Contains(body, `for="`+identifier[1]+`"`) {
			t.Errorf("%s: field %s has no label", typeChoice, field.Name)
		}

		if field.Required && !strings.Contains(control, "required") {
			t.Errorf("%s: required field %s is not marked required", typeChoice, field.Name)
		}
	}
}

func TestEditor_GeneratesAFormForEveryKindAndPreset_FromTheirSchemas(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	page := harness.get(t, "/key?page=0&row=1&column=1").Body.String()

	optionPattern := regexp.MustCompile(`<option value="((?:kind|preset):[a-z_]+)"`)

	for _, typeChoice := range optionPattern.FindAllStringSubmatch(page, -1) {
		body := formFieldsFor(t, harness, typeChoice[1])

		if !strings.Contains(body, `name="field.label"`) {
			t.Errorf("%s: want the shared appearance fields", typeChoice[1])
		}
	}

	offersEverything := strings.Contains(page, `value="kind:pipeline_status"`) &&
		strings.Contains(page, `value="preset:discord_channel"`)
	if !offersEverything {
		t.Fatal("want every kind and preset offered")
	}
}

func TestEditor_AddingAKindAddsItsForm_WithNoEditorChange(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, func(registry *brick.Registry) error {
		return testsupport.RegisterButtonKind(registry, testsupport.NewPressRecorder())
	})

	schema, deriveError := brick.DeriveSchema[testsupport.ButtonParameters]()
	if deriveError != nil {
		t.Fatal(deriveError)
	}

	body := formFieldsFor(t, harness, "kind:"+testsupport.ButtonKindName)
	requireFieldsInForm(t, "kind:"+testsupport.ButtonKindName, body, schema)
}

func TestEditor_SavesAKindKey_AndTheFileRoundTrips(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"2"}, typeField: {openURLType},
		urlField: {"https://discord.com/channels/1/2"}, "field.label": {"Chat"},
	})
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("want a redirect after saving, got %d: %s", recorder.Code, recorder.Body.String())
	}

	saved := harness.onDisk(t).Pages[0].Keys

	parameters, isOpenURL := saved[len(saved)-1].Parameters.(*openurl.Parameters)
	if !isOpenURL || parameters.URL != "https://discord.com/channels/1/2" || parameters.Label != "Chat" ||
		saved[len(saved)-1].Position != (layout.Position{Row: 1, Column: 2}) {
		t.Fatalf("want the key saved at row 1 column 2, got %+v", saved[len(saved)-1])
	}
}

func TestEditor_SavesAPresetKey_ByItsPresetAndFields(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"0"}, columnField: {"3"}, typeField: {"preset:discord_channel"},
		"field.server_id": {"123"}, "field.channel_id": {"456"}, "field.label": {"Party"},
	})
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("want a redirect after saving, got %d: %s", recorder.Code, recorder.Body.String())
	}

	fileContent, readError := os.ReadFile(harness.configPath)
	if readError != nil || !bytes.Contains(fileContent, []byte("preset: discord_channel")) {
		t.Fatalf("want the preset saved by name, got:\n%s", fileContent)
	}
}

func TestEditor_ShowsAValidationErrorAtItsField(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, typeField: {openURLType},
		urlField: {"http://insecure.test/"},
	})

	body := recorder.Body.String()
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", recorder.Code)
	}

	urlInput := regexp.MustCompile(`<input[^>]*name="field\.url"[^>]*>`).FindString(body)
	describedBy := regexp.MustCompile(`aria-describedby="([^"]*)"`).FindStringSubmatch(urlInput)

	if !strings.Contains(urlInput, `aria-invalid="true"`) || describedBy == nil {
		t.Fatalf("want the url input marked invalid and described, got %s", urlInput)
	}

	errorPattern := regexp.MustCompile(`id="` + regexp.QuoteMeta(strings.Fields(describedBy[1])[0]) + `"[^>]*>[^<]*Error:`)
	if !errorPattern.MatchString(body) {
		t.Fatal("want the field's error text, starting Error:, linked from the input")
	}
}

func TestEditor_RefusesAKeyOnANavigationPosition(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"3"}, columnField: {"0"}, typeField: {openURLType},
		urlField: {"https://example.test/"},
	})

	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), "navigation") {
		t.Fatalf("want 422 explaining the position is for navigation, got %d", recorder.Code)
	}
}

func TestEditor_DeletesAKey(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, "/key/delete", url.Values{pageField: {"0"}, rowField: {"0"}, columnField: {"0"}})

	if recorder.Code != http.StatusSeeOther || len(harness.onDisk(t).Pages[0].Keys) != 0 {
		t.Fatalf("want the key deleted, got %d", recorder.Code)
	}
}

func TestEditor_AddsRenamesAndDeletesPages(t *testing.T) {
	t.Parallel()

	const gamesPage = "Games"

	harness := newHarness(t, nil)

	steps := []struct {
		path      string
		form      url.Values
		wantPages []string
	}{
		{"/page/add", url.Values{nameField: {playName}}, []string{workName, playName}},
		{"/page/rename", url.Values{pageField: {"1"}, nameField: {gamesPage}}, []string{workName, gamesPage}},
		{"/page/delete", url.Values{pageField: {"0"}}, []string{gamesPage}},
	}

	for _, step := range steps {
		recorder := harness.post(t, step.path, step.form)
		if recorder.Code != http.StatusSeeOther {
			t.Fatalf("%s: want a redirect, got %d: %s", step.path, recorder.Code, recorder.Body.String())
		}

		pagesOnDisk := harness.onDisk(t).Pages

		names := make([]string, 0, len(pagesOnDisk))
		for _, page := range pagesOnDisk {
			names = append(names, page.Name)
		}

		if fmt.Sprint(names) != fmt.Sprint(step.wantPages) {
			t.Fatalf("%s: want pages %v, got %v", step.path, step.wantPages, names)
		}
	}
}

// The credential the credential tests store, and the secret's form field.
const (
	workCredential = "github-work"
	secretField    = "secret"
)

func TestEditor_StoresACredential_AndNeverShowsItBack(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	const secret = "github_pat_SHOWN_TO_NO_ONE_0123" //nolint:gosec // made up, to catch a secret shown back

	recorder := harness.post(t, credentialsPath, url.Values{nameField: {workCredential}, secretField: {secret}})
	page := harness.get(t, credentialsPath)

	stored, readError := harness.credentials.ReadCredential(workCredential)
	if readError != nil || stored != secret {
		t.Fatalf("want the secret stored, got an error %v", readError)
	}

	if strings.Contains(recorder.Body.String(), secret) || strings.Contains(page.Body.String(), secret) ||
		strings.Contains(recorder.Header().Get("Location"), secret) {
		t.Fatal("the secret must never be shown back")
	}
}

// TestEditor_BindsACredentialToItsHost: storing a token records the host it
// may be sent to, api.github.com unless an Enterprise host is given, and the
// page says which; a host that is not a plain host name is refused.
func TestEditor_BindsACredentialToItsHost(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	harness.post(t, credentialsPath, url.Values{nameField: {workCredential}, secretField: {"one"}})

	host, _ := platform.CredentialHost(harness.credentials, workCredential)
	if host != "api.github.com" {
		t.Errorf("want api.github.com when no host is given, got %q", host)
	}

	harness.post(t, credentialsPath, url.Values{
		nameField: {workCredential}, secretField: {"two"}, "host": {"github.example.test"},
	})

	host, _ = platform.CredentialHost(harness.credentials, workCredential)
	if host != "github.example.test" {
		t.Errorf("want the Enterprise host bound, got %q", host)
	}

	refused := harness.post(t, credentialsPath, url.Values{
		nameField: {workCredential}, secretField: {"three"}, "host": {"https://elsewhere.test/api"},
	})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Errorf("want a host with a scheme or path refused, got %d", refused.Code)
	}

	if stored, _ := harness.credentials.ReadCredential(workCredential); stored != "two" {
		t.Error("want a refused host to leave the stored secret alone")
	}
}

func TestEditor_ShowsTheDiagnosticCounters(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.counters.Increment("key_presses.open_url")

	body := harness.get(t, "/diagnostics").Body.String()
	if !strings.Contains(body, "key_presses.open_url") {
		t.Fatalf("want the counter listed, got:\n%s", body)
	}
}

func TestEditor_PreviewsAKeyWithTheRenderer(t *testing.T) {
	t.Parallel()

	recorder := newHarness(t, nil).get(t, "/preview.png?page=0&row=0&column=0")

	preview, decodeError := png.Decode(recorder.Body)
	if decodeError != nil || recorder.Header().Get("Content-Type") != "image/png" || preview.Bounds().Dx() != 96 {
		t.Fatalf("want a 96 pixel PNG, got %v (%v)", recorder.Header().Get("Content-Type"), decodeError)
	}
}

func TestEditor_ExplainsAHandEditTheStoreRefused(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	writeError := os.WriteFile(harness.configPath, []byte("schema_version: 1\npages: [\n"), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}

	body := harness.get(t, firstPagePath).Body.String()
	if !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "malformed") {
		t.Fatalf("want the refusal shown with its reason, got:\n%s", body)
	}
}

// TestEditor_TakesARejectedFormToItsFirstError checks where a keyboard or
// screen reader user lands after a refused save. In a dry run of the keyboard
// gate the error page opened with focus at the top and an unchanged title, so
// nothing said a save had failed (evaluation, deck testing). Now the title
// starts "Error:" and the first field in error, and only that one, is focused,
// which reads its label and its error.
func TestEditor_TakesARejectedFormToItsFirstError(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, typeField: {"preset:discord_channel"},
		"field.server_id": {"abc"}, "field.channel_id": {"xyz"},
	})

	body := recorder.Body.String()
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", recorder.Code)
	}

	if !regexp.MustCompile(`<title>Error: `).MatchString(body) {
		t.Error("want the title to start Error:")
	}

	focused := regexp.MustCompile(`<input[^>]*\sautofocus[^>]*>`).FindAllString(body, -1)
	if len(focused) != 1 || !strings.Contains(focused[0], `name="field.server_id"`) {
		t.Fatalf("want only the first field in error focused, got %q", focused)
	}
}

// TestEditor_CaptionCountsPagesFromOne pins the grid's caption. It read "Page
// 0 plus one of 1": the template held the words where the sum belonged.
func TestEditor_CaptionCountsPagesFromOne(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, firstPagePath).Body.String()

	if !regexp.MustCompile(`<caption>Page 1 of \d+: `).MatchString(body) {
		t.Fatalf("want the caption to count from one, got %s",
			regexp.MustCompile(`<caption>[^<]*</caption>`).FindString(body))
	}
}

func TestEditor_GridNamesAPresetKeyByItsAction_AndAKindKeyByItsKind(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, form := range []url.Values{
		{
			pageField: {"0"}, rowField: {"0"}, columnField: {"3"}, typeField: {"preset:" + discordPreset},
			serverIDField: {"123"}, channelIDField: {"456"}, labelField: {"Raid"},
		},
		{
			pageField: {"0"}, rowField: {"0"}, columnField: {"4"}, typeField: {"preset:teams_chat_by_id"},
			"field.chat_id": {"19:abc@thread.v2"},
		},
	} {
		recorder := harness.post(t, "/key", form)
		if recorder.Code != http.StatusSeeOther {
			t.Fatalf("want a redirect after saving, got %d: %s", recorder.Code, recorder.Body.String())
		}
	}

	body := harness.get(t, firstPagePath).Body.String()

	for _, want := range []string{
		`aria-label="Row 1, column 4: Raid (open_server)"`,
		`Raid<span class="kind">open_server</span>`,
		`aria-label="Row 1, column 5: open_chat"`,
		`aria-label="Row 1, column 1: Docs (open_url)"`,
		`Docs<span class="kind">open_url</span>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s in the page", want)
		}
	}

	if strings.Contains(body, "Raid (open_url)") || strings.Contains(body, `Raid<span class="kind">open_url`) {
		t.Error("want a preset key named by its action, not its target kind")
	}
}

func TestEditor_TypeListNamesEachPresetsAction(t *testing.T) {
	t.Parallel()

	page := newHarness(t, nil).get(t, "/key?page=0&row=1&column=1").Body.String()

	for _, want := range []string{
		`>Preset: discord_channel (open_server)</option>`,
		`>Preset: teams_shared_channel (open_channel)</option>`,
		`>Brick: open_url</option>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("want %s in the type list", want)
		}
	}
}
