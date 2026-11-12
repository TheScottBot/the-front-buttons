package editor_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// listedIcons returns the icon names a gallery page lists, in order.
func listedIcons(body string) []string {
	matches := regexp.MustCompile(`<code>([a-z0-9-]+)</code>`).FindAllStringSubmatch(body, -1)
	names := make([]string, 0, len(matches))

	for _, match := range matches {
		names = append(names, match[1])
	}

	return names
}

// TestEditor_IconGalleryListsEveryIcon_WithAPreview: the author had no way to
// know which icon names exist (evaluation, deck testing). The gallery lists
// them in order, each drawn as it would be on a key, and says how many there
// are.
func TestEditor_IconGalleryListsEveryIcon_WithAPreview(t *testing.T) {
	t.Parallel()

	recorder := newHarness(t, nil).get(t, "/icons")
	body := recorder.Body.String()

	if recorder.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", recorder.Code)
	}

	if got := strings.Join(listedIcons(body), ","); got != "brand-discord,dice-5,dragon,sword" {
		t.Fatalf("want every icon in order, got %s", got)
	}

	if !strings.Contains(body, `src="/icon.png?name=dragon"`) || !strings.Contains(body, "4 icons") {
		t.Fatal("want each icon previewed and the count stated")
	}
}

func TestEditor_IconGallerySearchNarrowsTheList(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for query, want := range map[string]string{
		"dra":         "dragon",
		"DICE":        "dice-5",
		"brand disc":  "brand-discord",
		"no-such-one": "",
	} {
		body := harness.get(t, "/icons?q="+strings.ReplaceAll(query, " ", "+")).Body.String()

		if got := strings.Join(listedIcons(body), ","); got != want {
			t.Errorf("search %q: want %q, got %q", query, want, got)
		}
	}
}

func TestEditor_IconPreviewDrawsAKnownIcon_AndRefusesAnyOtherName(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	known := harness.get(t, "/icon.png?name=dragon")
	if known.Code != http.StatusOK || known.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("want a PNG for a known icon, got %d %s", known.Code, known.Header().Get("Content-Type"))
	}

	for _, name := range []string{"not-an-icon", "", "../config"} {
		if unknown := harness.get(t, "/icon.png?name="+name); unknown.Code != http.StatusNotFound {
			t.Errorf("icon %q: want 404, got %d", name, unknown.Code)
		}
	}
}

func TestEditor_KeyFormLinksTheIconFieldToTheGallery(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, "/key?page=0&row=1&column=1&type="+openURLType).Body.String()

	if !regexp.MustCompile(`id="field-icon-help"[^>]*>[^<]*<a href="/icons"`).MatchString(body) {
		t.Fatal("want the icon field's help to link to the icon gallery")
	}
}
