package editor_test

import (
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The editor follows the system's light or dark setting. Light is the punk
// palette; dark is Thyme's: its panel, its pink time text and outline
// (github.com/TheScottBot/Thyme, internal/theme). Every rule names a colour
// role, never a palette colour, so the dark palette reaches all of them, and
// each role pair a person reads meets WCAG AA in both (engineering datum 0.16).

// The colour roles the tests check, named once.
const (
	pageRole      = "page"
	textRole      = "text"
	blockRole     = "block"
	highlightRole = "highlight"
	fieldRole     = "field"
	ruleRole      = "rule"
	accentRole    = "accent"
)

// The WCAG AA minimums: text, and the parts of a control such as its focus.
const (
	textContrast    = 4.5
	nonTextContrast = 3
)

// themeCookie is the cookie the switch keeps its choice in; noThemeChosen is
// the page's opening tag when there is no choice, so the system's setting holds.
const (
	themeCookie   = "theme"
	noThemeChosen = `<html lang="en-GB">`
)

// maximumReferences bounds following one role through others, so a circle of
// references fails rather than looping.
const maximumReferences = 8

// customProperty matches one custom property's definition; propertyReference
// a value that is only a reference to another; paletteColour a use of one of
// the light palette's own names, which only the role definitions may make.
var (
	customProperty    = regexp.MustCompile(`-{2}([a-z0-9-]+):\s*([^;]+);`)
	propertyReference = regexp.MustCompile(`^var\(-{2}([a-z0-9-]+)\)$`)
	paletteColour     = regexp.MustCompile(`var\(-{2}(pink|ink|yellow|cyan|paper)\)`)
	rootBlock         = regexp.MustCompile(`(?s):root\s*\{.*?\}`)
	firstRootBlock    = regexp.MustCompile(`(?s)^.*?:root\s*\{(.*?)\}`)
	darkRootBlock     = regexp.MustCompile(
		`(?s)@media \(prefers-color-scheme: dark\)\s*\{\s*:root:not\(\[data-theme="light"\]\)\s*\{(.*?)\}`)
	chosenDarkBlock = regexp.MustCompile(`(?s):root\[data-theme="dark"\]\s*\{(.*?)\}`)
)

func TestStylesheet_FollowsTheSystemIntoThymesDarkPalette(t *testing.T) {
	t.Parallel()

	_, dark := stylesheetPalettes(t)

	for role, want := range map[string]string{pageRole: "#12131c", textRole: "#f894f8", ruleRole: "#f59ef3"} {
		if got := dark.resolve(t, role); got != want {
			t.Errorf("dark %s role: want Thyme's %s, got %s", role, want, got)
		}
	}
}

func TestStylesheet_EveryRolePairMeetsContrastInBothPalettes(t *testing.T) {
	t.Parallel()

	// The text and background roles drawn together.
	textPairs := [][2]string{
		{textRole, pageRole},
		{"on-block", blockRole},
		{"on-block-accent", blockRole},
		{"on-block-accent-2", blockRole},
		{textRole, highlightRole},
		{textRole, "notice"},
		{textRole, fieldRole},
	}

	light, dark := stylesheetPalettes(t)

	for name, palette := range map[string]palette{"light": light, "dark": dark} {
		for _, pair := range textPairs {
			ratio := contrast(t, palette.resolve(t, pair[0]), palette.resolve(t, pair[1]))
			if ratio < textContrast {
				t.Errorf("%s: %s on %s is %.2f to 1, want at least %.1f", name, pair[0], pair[1], ratio, textContrast)
			}
		}

		// Focus is an outline in the rule role with a halo in the accent role,
		// so one of the two must show against whatever a control sits on.
		for _, surface := range []string{pageRole, blockRole, highlightRole, fieldRole} {
			behind := palette.resolve(t, surface)
			best := max(contrast(t, palette.resolve(t, ruleRole), behind), contrast(t, palette.resolve(t, accentRole), behind))

			if best < nonTextContrast {
				t.Errorf("%s: focus on %s is %.2f to 1 at best, want at least %d", name, surface, best, nonTextContrast)
			}
		}
	}
}

func TestStylesheet_RulesNameRolesNotPaletteColours(t *testing.T) {
	t.Parallel()

	stylesheet := newHarness(t, nil).get(t, "/static/editor.css").Body.String()

	for _, use := range paletteColour.FindAllString(rootBlock.ReplaceAllString(stylesheet, ""), -1) {
		t.Errorf("a rule names the palette colour %s; name a role so dark mode reaches it", use)
	}
}

// TestStylesheet_ChosenDarkMatchesTheSystemsDark: the switch's dark is the
// same palette as the system's, and the system's gives way to a chosen light.
func TestStylesheet_ChosenDarkMatchesTheSystemsDark(t *testing.T) {
	t.Parallel()

	stylesheet := newHarness(t, nil).get(t, "/static/editor.css").Body.String()

	chosen := chosenDarkBlock.FindStringSubmatch(stylesheet)
	if chosen == nil {
		t.Fatal(`stylesheet: want a :root[data-theme="dark"] block for the switch`)
	}

	_, system := stylesheetPalettes(t)

	if got := customProperties(chosen[1]); !maps.Equal(got, system.properties) {
		t.Errorf("want the chosen dark palette the same as the system's,\ngot  %v\nwant %v", got, system.properties)
	}
}

// TestEditor_ThemeSwitchWaitsForTheScript: the author asked for a switch
// between light and dark. It needs the script, so it is hidden until the
// script shows it, and it is a switch to a screen reader.
func TestEditor_ThemeSwitchWaitsForTheScript(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, path := range editorPages() {
		body := harness.get(t, path).Body.String()

		if !regexp.MustCompile(`<button type="button" class="theme-switch" role="switch" aria-checked="false" hidden>`).
			MatchString(body) {
			t.Errorf("%s: want the theme switch, hidden until the script runs", path)
		}

		if strings.Contains(body, "data-theme") {
			t.Errorf("%s: want no theme chosen without the cookie, so the system's setting holds", path)
		}
	}

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"theme-switch", themeCookie + "=", "prefers-color-scheme: dark"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}
}

// TestEditor_ChosenThemeIsDrawnByTheServer: the choice is kept in a cookie and
// written onto the page as it is served, so a page never flashes the other
// palette before the script runs. Only light and dark are taken.
func TestEditor_ChosenThemeIsDrawnByTheServer(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for value, want := range map[string]string{
		"dark":   `<html lang="en-GB" data-theme="dark">`,
		"light":  `<html lang="en-GB" data-theme="light">`,
		"purple": noThemeChosen,
		"Dark":   noThemeChosen,
	} {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+testAddress+firstPagePath,
			http.NoBody)
		request.Header.Set("Cookie", themeCookie+"="+value)
		withSession(request)

		recorder := httptest.NewRecorder()
		harness.handler.ServeHTTP(recorder, request)

		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("cookie %q: want %s", value, want)
		}
	}
}

// palette is the custom properties in force: its own, then those it falls
// back on.
type palette struct {
	properties map[string]string
	fallback   map[string]string
}

// resolve follows a property through references to a colour.
func (palette palette) resolve(t *testing.T, name string) string {
	t.Helper()

	for range maximumReferences {
		value, found := palette.properties[name]
		if !found {
			value, found = palette.fallback[name]
		}

		if !found {
			t.Fatalf("no %s property defined", name)
		}

		reference := propertyReference.FindStringSubmatch(value)
		if reference == nil {
			return strings.ToLower(value)
		}

		name = reference[1]
	}

	t.Fatalf("the %s property refers round in a circle", name)

	return ""
}

// stylesheetPalettes reads the light :root and the dark one inside
// prefers-color-scheme: dark.
func stylesheetPalettes(t *testing.T) (palette, palette) {
	t.Helper()

	stylesheet := newHarness(t, nil).get(t, "/static/editor.css").Body.String()

	base := firstRootBlock.FindStringSubmatch(stylesheet)
	if base == nil {
		t.Fatal("stylesheet: want a :root block")
	}

	dark := darkRootBlock.FindStringSubmatch(stylesheet)
	if dark == nil {
		t.Fatal("stylesheet: want a :root block inside @media (prefers-color-scheme: dark)")
	}

	light := customProperties(base[1])

	return palette{properties: light, fallback: nil}, palette{properties: customProperties(dark[1]), fallback: light}
}

func customProperties(block string) map[string]string {
	properties := map[string]string{}

	for _, match := range customProperty.FindAllStringSubmatch(block, -1) {
		properties[match[1]] = strings.TrimSpace(match[2])
	}

	return properties
}

// contrast is the WCAG contrast ratio of two #rrggbb colours.
func contrast(t *testing.T, first string, second string) float64 {
	t.Helper()

	lighter, darker := luminance(t, first), luminance(t, second)
	if darker > lighter {
		lighter, darker = darker, lighter
	}

	return (lighter + 0.05) / (darker + 0.05)
}

// luminance is a #rrggbb colour's relative luminance, by WCAG's formula.
func luminance(t *testing.T, hex string) float64 {
	t.Helper()

	if len(hex) != len("#rrggbb") || hex[0] != '#' {
		t.Fatalf("want a #rrggbb colour, got %q", hex)
	}

	total := 0.0

	for index, weight := range []float64{0.2126, 0.7152, 0.0722} {
		channel, parseError := strconv.ParseUint(hex[1+2*index:3+2*index], 16, 8)
		if parseError != nil {
			t.Fatalf("colour %q: %v", hex, parseError)
		}

		value := float64(channel) / math.MaxUint8
		if value <= 0.04045 {
			value /= 12.92
		} else {
			value = math.Pow((value+0.055)/1.055, 2.4)
		}

		total += weight * value
	}

	return total
}
