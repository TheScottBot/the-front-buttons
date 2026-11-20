package config_test

import (
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// withNavigationBackground is a valid file with a navigation_background line
// before its pages.
func withNavigationBackground(value string) string {
	return strings.Replace(validConfiguration, "pages:\n", "navigation_background: "+value+"\npages:\n", 1)
}

// TestDecode_ReadsNavigationBackground: navigation_background is the colour
// of both navigation keys on every page, read as written.
func TestDecode_ReadsNavigationBackground(t *testing.T) {
	t.Parallel()

	for _, want := range []string{"#1F6FEB", "#a0b1c2"} {
		configuration, decodeError := newTestDecoder(t).Decode([]byte(withNavigationBackground(`"` + want + `"`)))
		if decodeError != nil {
			t.Fatal(decodeError)
		}

		if configuration.NavigationBackground != want {
			t.Errorf("want %q, got %q", want, configuration.NavigationBackground)
		}
	}
}

// TestDecode_LeavesNavigationBackgroundAbsent: a file without it decodes
// with none, so saving it never writes one in and the keys keep their grey.
func TestDecode_LeavesNavigationBackgroundAbsent(t *testing.T) {
	t.Parallel()

	if got := decodeValidConfiguration(t).NavigationBackground; got != "" {
		t.Fatalf("want no navigation background decoded, got %q", got)
	}
}

// TestDecode_RefusesABadNavigationBackground: anything not written #RRGGBB
// refuses the whole file, naming the field, with the line it is on: text in
// another form as a bad format, anything but text as the wrong type.
func TestDecode_RefusesABadNavigationBackground(t *testing.T) {
	t.Parallel()

	refusals := map[config.Reason][]string{
		config.ReasonBadFormat: {"red", `"#F00"`, `"FF0000"`, `""`},
		config.ReasonWrongType: {"1", `["#FF0000"]`, ""},
	}

	for reason, values := range refusals {
		for _, value := range values {
			problem := requireProblem(t, withNavigationBackground(value), reason)
			if problem.Path != "navigation_background" || problem.Line == 0 ||
				!strings.Contains(problem.Detail, "#RRGGBB") {
				t.Errorf("%q: want %s at navigation_background with its line and the form allowed, got %+v",
					value, reason, problem)
			}
		}
	}
}

// TestEncode_WritesNavigationBackgroundOnlyWhenSet: the colour is written
// before the pages and round trips through a copy and the encoder, and a file
// without it is written without it.
func TestEncode_WritesNavigationBackgroundOnlyWhenSet(t *testing.T) {
	t.Parallel()

	decoder := newTestDecoder(t)

	coloured, decodeError := decoder.Decode([]byte(withNavigationBackground(`"#1F6FEB"`)))
	if decodeError != nil {
		t.Fatal(decodeError)
	}

	written, encodeError := decoder.Encode(coloured.Clone())
	if encodeError != nil {
		t.Fatal(encodeError)
	}

	at := strings.Index(string(written), "navigation_background:")
	if at < 0 || at > strings.Index(string(written), "pages:") {
		t.Fatalf("want the colour written before the pages, got:\n%s", written)
	}

	reread, rereadError := decoder.Decode(written)
	if rereadError != nil || reread.NavigationBackground != "#1F6FEB" {
		t.Fatalf("want #1F6FEB after a round trip, got %q (%v)", reread.NavigationBackground, rereadError)
	}

	plain, plainError := decoder.Encode(decodeValidConfiguration(t))
	if plainError != nil || strings.Contains(string(plain), "navigation_background") {
		t.Fatalf("want a file without the colour written without it, got (%v):\n%s", plainError, plain)
	}
}
