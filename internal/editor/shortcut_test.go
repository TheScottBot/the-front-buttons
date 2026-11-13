package editor_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/bricks/pressshortcut"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// shortcutKeyPath is a new shortcut key's editor, and keyField the form name
// of a key brick's key.
const (
	shortcutKeyPath = "/key?page=0&row=1&column=1&type=kind:press_shortcut"
	keyField        = "field.key"
)

// TestEditor_ShortcutFormOffersModifiersForEachPlatform: the author asked to
// click Control, Shift, Option or Alt and Command in any combination, each
// named as the platform names it, with a switch between the Mac's names and
// Windows's (evaluation, deck testing). The key is chosen from a list.
func TestEditor_ShortcutFormOffersModifiersForEachPlatform(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, shortcutKeyPath).Body.String()

	if !regexp.MustCompile(`<select id="field-key" name="field.key"`).MatchString(body) ||
		!strings.Contains(body, `<option value="C">C</option>`) {
		t.Error("want the key chosen from a list")
	}

	for name, labels := range map[string][2]string{
		"control": {"⌃ Control", "Ctrl"},
		"alt":     {"⌥ Option", "Alt"},
		"shift":   {"⇧ Shift", "Shift"},
		"command": {"⌘ Command", "Win"},
	} {
		want := `<label for="field-` + name + `" data-mac="` + labels[0] + `" data-windows="` + labels[1] + `">` +
			labels[0] + `</label>`
		if !strings.Contains(body, want) {
			t.Errorf("want the %s chip named for the Mac, with both names: %s", name, want)
		}
	}

	if !strings.Contains(body, `<div class="modifier-platform" hidden>`) ||
		!strings.Contains(body, `data-platform="windows"`) {
		t.Error("want the platform switch, hidden until the script shows it")
	}

	if strings.Count(body, `class="modifier-platform"`) != 1 {
		t.Error("want one platform switch for all the modifiers")
	}
}

// TestEditor_ShortcutFormStartsWithThisMachinesNames: the Dell's editor shows
// Windows's names first.
func TestEditor_ShortcutFormStartsWithThisMachinesNames(t *testing.T) {
	t.Parallel()

	body := newHarnessOnWindows(t).get(t, shortcutKeyPath).Body.String()

	if !strings.Contains(body, `data-windows="Win">Win</label>`) {
		t.Error("want the Windows names shown first on Windows")
	}
}

// TestEditor_SavesAShortcut: a shortcut saved is its key and the modifiers
// ticked.
func TestEditor_SavesAShortcut(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	response := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, typeField: {"kind:press_shortcut"},
		keyField: {"C"}, "field.command": {"on"}, "field.shift": {"on"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("want the shortcut saved, got %d: %s", response.Code, response.Body.String())
	}

	key, _ := keyAtPosition(harness.onDisk(t).Pages[0], layout.Position{Row: 1, Column: 1})

	parameters, isShortcut := key.Parameters.(*pressshortcut.Parameters)
	if !isShortcut || parameters.Key != "C" || !parameters.Command || !parameters.Shift || parameters.Control ||
		parameters.Alt {
		t.Errorf("want ⇧⌘C saved, got %+v", key.Parameters)
	}

	if !strings.Contains(harness.get(t, "/static/editor.js").Body.String(), "platform-choice") {
		t.Error("script: want it to switch the modifier names")
	}
}
