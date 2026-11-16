package editor_test

import (
	"html"
	"strings"
	"testing"
)

// The key form for a Spotify key, for window cycling, and for opening an
// application, whose badge check box does nothing on Windows.
const (
	spotifyKeyPath     = "/key?page=0&row=1&column=1&type=kind:spotify_next"
	windowCycleKeyPath = "/key?page=0&row=1&column=1&type=kind:jetbrains_next_window"
	openAppKeyPath     = "/key?page=0&row=1&column=1&type=kind:open_application"
	notOnWindowsMark   = "(not on Windows)"
)

// TestEditor_OnWindows_SaysWhichKeysDoNotRunThere: on Windows, the drawer and
// the type list mark the kinds that do not run there, and their about says
// why; the key form for one says so above its fields, and window cycling's
// points to Alt+Tab instead.
func TestEditor_OnWindows_SaysWhichKeysDoNotRunThere(t *testing.T) {
	t.Parallel()

	harness := newHarnessOnWindows(t)

	drawer := harness.get(t, "/page?page=0").Body.String()
	for _, want := range []string{
		"<summary>Brick: spotify_next " + notOnWindowsMark + "</summary>",
		"<summary>Brick: jetbrains_next_window " + notOnWindowsMark + "</summary>",
		"Controlling Spotify is only written for macOS so far",
	} {
		if !strings.Contains(drawer, want) {
			t.Errorf("drawer: want %q", want)
		}
	}

	if strings.Contains(drawer, "<summary>Brick: open_url "+notOnWindowsMark) {
		t.Error("drawer: want a kind that runs on Windows left unmarked")
	}

	spotifyForm := harness.get(t, spotifyKeyPath).Body.String()
	for _, want := range []string{
		`<option value="kind:spotify_next" selected>Brick: spotify_next ` + notOnWindowsMark + `</option>`,
		`<p class="not-here" role="note">Not available on Windows: Controlling Spotify is only written for macOS so far`,
	} {
		if !strings.Contains(spotifyForm, want) {
			t.Errorf("Spotify key form: want %q", want)
		}
	}

	// The template writes + as a character reference, which a browser shows
	// as +.
	cycleForm := html.UnescapeString(harness.get(t, windowCycleKeyPath).Body.String())
	if !strings.Contains(cycleForm, "Alt+Tab") {
		t.Error("window cycling key form: want Alt+Tab offered instead")
	}
}

// TestEditor_OnWindows_SaysWhichFieldsDoNotRunThere: on Windows, the open
// application form says beside the badge check box that it does nothing
// there, and nothing above its fields, since the key itself runs.
func TestEditor_OnWindows_SaysWhichFieldsDoNotRunThere(t *testing.T) {
	t.Parallel()

	form := newHarnessOnWindows(t).get(t, openAppKeyPath).Body.String()

	const badgeNote = `<p class="not-here" id="field-show_notifications-not-here">Not available on Windows.</p>`
	if !strings.Contains(form, badgeNote) {
		t.Error("want the badge check box marked not available on Windows")
	}

	if strings.Contains(form, `role="note">Not available on Windows:`) {
		t.Error("want no note above a key that runs on Windows")
	}
}

// TestEditor_OnAMac_MarksNothing: every key runs on a Mac, so nothing is
// marked there.
func TestEditor_OnAMac_MarksNothing(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, path := range []string{"/page?page=0", spotifyKeyPath, windowCycleKeyPath, openAppKeyPath} {
		body := harness.get(t, path).Body.String()
		if strings.Contains(body, "not on Windows") || strings.Contains(body, "Not available on") {
			t.Errorf("%s: want nothing marked on a Mac", path)
		}
	}
}
