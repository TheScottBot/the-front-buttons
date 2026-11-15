package editor_test

import (
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// The world clocks form's route and where a save lands.
const (
	clocksPath  = "/dashboard/clocks"
	clocksSaved = "/dashboard?saved=clocks"
)

// clocksForm is the form's fields for four rows of label and zone, the rest
// left blank.
func clocksForm(rows ...string) url.Values {
	form := url.Values{}

	for row := range 4 {
		label, zone := "", ""
		if 2*row+1 < len(rows) {
			label, zone = rows[2*row], rows[2*row+1]
		}

		form.Set("label."+string(rune('0'+row)), label)
		form.Set("zone."+string(rune('0'+row)), zone)
	}

	return form
}

// TestEditor_WorldClocksFormShowsTheFilesClocks: under the world clocks'
// preview, a form has a row for each of up to four clocks, the file's filled
// in and the rest blank; with none in the file, it says so and fills in the
// defaults the deck shows.
func TestEditor_WorldClocksFormShowsTheFilesClocks(t *testing.T) {
	t.Parallel()

	body := newDashboardHarness(t, dashboardConfiguration).get(t, dashboardPath).Body.String()
	for _, want := range []string{
		`<form method="post" action="/dashboard/clocks" class="clocks-form">`,
		`<input type="hidden" name="token" value="` + testToken + `">`,
		`<input id="clock-label-0" name="label.0" type="text" value="BER" maxlength="4"`,
		`<input id="clock-zone-0" name="zone.0" type="text" value="Europe/Berlin" maxlength="64"`,
		`<input id="clock-label-1" name="label.1" type="text" value="ARQ" maxlength="4"`,
		`<input id="clock-label-3" name="label.3" type="text" value="" maxlength="4"`,
		`<button type="submit">Save clocks</button>`,
		`<button type="submit" name="defaults" value="on">Use the defaults</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("form: want %s", want)
		}
	}

	if strings.Contains(body, "The file names no clocks") {
		t.Error("with clocks in the file: want no word that it has none")
	}

	defaults := newDashboardHarness(t, copyKeysConfiguration).get(t, dashboardPath).Body.String()
	for _, want := range []string{
		"The file names no clocks",
		`name="label.0" type="text" value="UTC"`,
		`name="zone.3" type="text" value="America/Sao_Paulo"`,
	} {
		if !strings.Contains(defaults, want) {
			t.Errorf("without clocks: want %s", want)
		}
	}
}

// TestEditor_SavesTheWorldClocksFromTheForm: the form's rows are saved as the
// file's clocks, in order, a blank row left out and the pages untouched;
// Undo puts the clocks back.
func TestEditor_SavesTheWorldClocksFromTheForm(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, dashboardConfiguration)
	before := harness.onDisk(t)

	response := harness.post(t, clocksPath, clocksForm("TYO", " Asia/Tokyo ", "", "", "NYC", "America/New_York"))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != clocksSaved {
		t.Fatalf("want a redirect to %s, got %d to %q", clocksSaved, response.Code, response.Header().Get("Location"))
	}

	saved := harness.onDisk(t)
	want := []config.WorldClock{{Label: "TYO", Zone: "Asia/Tokyo"}, {Label: "NYC", Zone: "America/New_York"}}

	if !slices.Equal(saved.WorldClocks, want) {
		t.Errorf("want %v, got %v", want, saved.WorldClocks)
	}

	if !reflect.DeepEqual(saved.Pages, before.Pages) {
		t.Error("want the pages untouched")
	}

	if body := harness.get(t, clocksSaved).Body.String(); !strings.Contains(body, "Clocks saved.") {
		t.Error("want the save announced")
	}

	harness.post(t, "/undo", url.Values{})

	if undone := harness.onDisk(t); !slices.Equal(undone.WorldClocks, before.WorldClocks) {
		t.Errorf("undo: want %v back, got %v", before.WorldClocks, undone.WorldClocks)
	}
}

// TestEditor_RefusesBadWorldClocks_KeepingWhatWasTyped: clocks the file
// would refuse are not saved; the form comes back with what was typed and
// each problem beside its row's field, the rows counted as shown though a
// blank row comes first.
func TestEditor_RefusesBadWorldClocks_KeepingWhatWasTyped(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, dashboardConfiguration)
	before := harness.onDisk(t)

	response := harness.post(t, clocksPath, clocksForm("", "", "MOON", "Mars/Base", "PARIS", "Europe/Paris"))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want the form refused, got %d", response.Code)
	}

	if after := harness.onDisk(t); !reflect.DeepEqual(after, before) {
		t.Error("want the file unchanged")
	}

	body := response.Body.String()
	for _, want := range []string{
		`name="label.1" type="text" value="MOON"`,
		`name="zone.1" type="text" value="Mars/Base" maxlength="64" aria-invalid="true"`,
		`<p class="error" id="clock-zone-1-error">Error: not a zone in the zone database, such as Europe/Berlin</p>`,
		`name="label.2" type="text" value="PARIS" maxlength="4" aria-invalid="true"`,
		`<p class="error" id="clock-label-2-error">Error: must be between 1 and 4 characters</p>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("refused form: want %s", want)
		}
	}

	if strings.Contains(body, `id="clock-label-1-error"`) {
		t.Error("want no problem beside a good label")
	}
}

// TestEditor_RefusedClocksStillSayTheFileHasNone: a refused save on a file
// with no clocks still says the deck shows the defaults.
func TestEditor_RefusedClocksStillSayTheFileHasNone(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, copyKeysConfiguration)

	response := harness.post(t, clocksPath, clocksForm("UTC", "Mars/Base"))
	if response.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(response.Body.String(), "The file names no clocks") {
		t.Errorf("want it refused, still saying the file has none, got %d", response.Code)
	}
}

// TestEditor_WorldClocksFormNeedsAClockOrTheDefaults: every row left blank
// is refused, as an empty list is; Use the defaults takes the clocks out of
// the file, so the deck shows the defaults.
func TestEditor_WorldClocksFormNeedsAClockOrTheDefaults(t *testing.T) {
	t.Parallel()

	harness := newDashboardHarness(t, dashboardConfiguration)

	blank := harness.post(t, clocksPath, clocksForm())
	if blank.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(blank.Body.String(), "there must be between 1 and 4") {
		t.Errorf("every row blank: want it refused with why, got %d", blank.Code)
	}

	form := clocksForm("TYO", "Asia/Tokyo")
	form.Set("defaults", "on")

	response := harness.post(t, clocksPath, form)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("defaults: want a redirect, got %d", response.Code)
	}

	if saved := harness.onDisk(t); saved.WorldClocks != nil {
		t.Errorf("defaults: want no clocks in the file, got %v", saved.WorldClocks)
	}
}
