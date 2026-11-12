package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// editorPages are every page a person can reach in the editor.
func editorPages() []string {
	return []string{
		"/page?page=0",
		"/key?page=0&row=0&column=0",
		"/key?page=0&row=1&column=1&type=kind:pipeline_status",
		"/key?page=0&row=1&column=1&type=preset:teams_channel",
		brightnessKeyPath,
		"/credentials",
		"/diagnostics",
	}
}

func TestEditor_EveryControlHasALabel(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	controlPattern := regexp.MustCompile(`<(input|select|textarea)\b[^>]*>`)

	for _, path := range editorPages() {
		body := harness.get(t, path).Body.String()

		for _, control := range controlPattern.FindAllString(body, -1) {
			if strings.Contains(control, `type="hidden"`) || strings.Contains(control, `type="submit"`) {
				continue
			}

			identifier := regexp.MustCompile(`\bid="([^"]+)"`).FindStringSubmatch(control)
			if identifier == nil || !strings.Contains(body, `for="`+identifier[1]+`"`) {
				t.Errorf("%s: control without a label: %s", path, control)
			}
		}
	}
}

func TestEditor_EveryPageHasASkipLinkAMainLandmarkAndALiveRegion(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, path := range editorPages() {
		body := harness.get(t, path).Body.String()

		for _, want := range []string{
			`<html lang="en-GB">`,
			`<a class="skip" href="#main">`,
			`<main id="main"`,
			`role="status" aria-live="polite"`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: want %s", path, want)
			}
		}

		if regexp.MustCompile(`tabindex="[1-9]`).MatchString(body) {
			t.Errorf("%s: a positive tabindex breaks the natural keyboard order", path)
		}
	}
}

func TestEditor_GridIsOperableFromTheKeyboard(t *testing.T) {
	t.Parallel()

	body := newHarness(t, nil).get(t, "/page?page=0").Body.String()

	grid := regexp.MustCompile(`(?s)<table class="grid".*?</table>`).FindString(body)
	links := regexp.MustCompile(`<a\b[^>]*>`).FindAllString(grid, -1)

	// 32 keys less the two reserved navigation keys, each a link the keyboard
	// reaches with Tab and follows with Enter, each named for a screen reader.
	if len(links) != 30 {
		t.Fatalf("want 30 key links in the grid, got %d", len(links))
	}

	for _, link := range links {
		if !strings.Contains(link, "aria-label=") {
			t.Errorf("grid link without a name: %s", link)
		}
	}
}

func TestEditor_StylesShowFocusRespectReducedMotionAndNeverUseColourAlone(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	stylesheet := harness.get(t, "/static/editor.css").Body.String()

	for _, want := range []string{":focus-visible", "outline:", "prefers-reduced-motion"} {
		if !strings.Contains(stylesheet, want) {
			t.Errorf("stylesheet: want %s", want)
		}
	}

	errorPage := harness.post(t, "/key", map[string][]string{
		pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, typeField: {openURLType}, urlField: {"http://x.test/"},
	}).Body.String()

	if !strings.Contains(errorPage, "Error:") {
		t.Fatal("want errors announced in words, not colour alone")
	}
}
