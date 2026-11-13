package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestEditor_OffersInstalledApplications_OnTheApplicationFieldOnly: the author
// asked for an app picker (evaluation, deck testing). The open_application
// key's form lists the installed applications by name under its field, each
// choice holding the identifier the field takes; no other field gets one.
func TestEditor_OffersInstalledApplications_OnTheApplicationFieldOnly(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, "/key?page=0&row=1&column=1&type=kind:open_application").Body.String()

	picker := regexp.MustCompile(`(?s)<select[^>]*class="app-picker"[^>]*>.*?</select>`).FindString(body)
	if picker == "" {
		t.Fatal("want an application picker on the application field")
	}

	for _, want := range []string{
		`data-fills="field-application"`,
		`<option value="com.apple.Terminal">Terminal</option>`,
		`<option value="com.spotify.client">Spotify</option>`,
	} {
		if !strings.Contains(picker, want) {
			t.Errorf("want %s in the picker", want)
		}
	}

	if !strings.Contains(body, `<label for="field-application-picker">`) {
		t.Error("want the picker labelled")
	}

	if strings.Count(body, `class="app-picker"`) != 1 {
		t.Error("want the picker on the application field only")
	}
}

func TestEditor_OffersNoPicker_ForOtherKinds(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, "/key?page=0&row=1&column=1&type=kind:open_url").Body.String()

	if strings.Contains(body, `class="app-picker"`) {
		t.Fatal("want no application picker on an open_url key")
	}
}

// TestEditor_EveryPageLoadsTheEditorScript: the picker needs the script on a
// key's own page as well as in the grid's dialog; with the script only on the
// grid page, choosing an application on a key's page filled nothing.
func TestEditor_EveryPageLoadsTheEditorScript(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, path := range []string{firstPagePath, "/key?page=0&row=1&column=1", "/icons", credentialsPath} {
		if body := harness.get(t, path).Body.String(); strings.Count(body, `src="/static/editor.js"`) != 1 {
			t.Errorf("%s: want the editor script loaded once", path)
		}
	}
}

// TestEditor_UseAppIconSetsTheIconFieldsAside: ticking use_app_icon should
// stop the Icon and Icon file fields being used, as the author asked; the
// check box names them for the script to set aside while it is ticked.
func TestEditor_UseAppIconSetsTheIconFieldsAside(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, "/key?page=0&row=1&column=1&type=kind:open_application").Body.String()

	checkBox := regexp.MustCompile(`<input id="field-use_app_icon"[^>]*>`).FindString(body)
	if !strings.Contains(checkBox, `data-replaces="field-icon field-icon_file"`) {
		t.Fatalf("want the check box to name the icon fields, got %s", checkBox)
	}
}
