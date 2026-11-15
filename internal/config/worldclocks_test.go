package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// withClocks is a valid file with a world_clocks section before its pages.
func withClocks(clocks string) string {
	return strings.Replace(validConfiguration, "pages:\n", "world_clocks:\n"+clocks+"pages:\n", 1)
}

// berlinAndAraraquara are two world clocks as the author might write them.
const berlinAndAraraquara = `  - label: BER
    zone: Europe/Berlin
  - label: ARQ
    zone: America/Sao_Paulo
`

// TestDecode_ReadsWorldClocks: world_clocks lists the world view's clocks,
// each a label and a zone, in the order written.
func TestDecode_ReadsWorldClocks(t *testing.T) {
	t.Parallel()

	configuration, decodeError := newTestDecoder(t).Decode([]byte(withClocks(berlinAndAraraquara)))
	if decodeError != nil {
		t.Fatal(decodeError)
	}

	want := []config.WorldClock{{Label: "BER", Zone: "Europe/Berlin"}, {Label: "ARQ", Zone: "America/Sao_Paulo"}}
	if !slices.Equal(configuration.WorldClocks, want) || !slices.Equal(configuration.WorldClocksOrDefault(), want) {
		t.Fatalf("want %v, got %v", want, configuration.WorldClocks)
	}
}

// TestDecode_LeavesWorldClocksAbsent_MeaningTheDefaults: a file without
// world_clocks decodes with none, so saving it never writes them in, and the
// world view takes UTC, London, Berlin and Sao Paulo.
func TestDecode_LeavesWorldClocksAbsent_MeaningTheDefaults(t *testing.T) {
	t.Parallel()

	configuration := decodeValidConfiguration(t)
	if configuration.WorldClocks != nil {
		t.Fatalf("want no world clocks decoded, got %v", configuration.WorldClocks)
	}

	labels := make([]string, 0, config.MaximumWorldClocks)
	for _, clock := range configuration.WorldClocksOrDefault() {
		labels = append(labels, clock.Label)
	}

	if !slices.Equal(labels, []string{"UTC", "LDN", "BER", "SAO"}) {
		t.Fatalf("want the four defaults, got %v", labels)
	}
}

// TestDecode_RefusesWorldClocksThatCannotBeShown: each clock the world view
// could not show, or a zone that is not one, refuses the whole file, by path
// and reason.
func TestDecode_RefusesWorldClocksThatCannotBeShown(t *testing.T) {
	t.Parallel()

	const (
		clocksPath = "world_clocks"
		firstClock = clocksPath + "[0]"
		zonePath   = firstClock + ".zone"
	)

	clock := func(label, zone string) string { return "  - label: " + label + "\n    zone: " + zone + "\n" }

	for name, check := range map[string]struct {
		clocks string
		reason config.Reason
		path   string
	}{
		"an empty list":      {"  []\n", config.ReasonOutOfBounds, clocksPath},
		"five clocks":        {strings.Repeat(clock("UTC", "UTC"), 5), config.ReasonTooMany, clocksPath},
		"a long label":       {clock("BERLN", "Europe/Berlin"), config.ReasonOutOfBounds, "world_clocks[0].label"},
		"punctuation":        {clock("B-R", "Europe/Berlin"), config.ReasonBadFormat, "world_clocks[0].label"},
		"no label":           {"  - zone: UTC\n", config.ReasonMissing, firstClock},
		"no zone":            {"  - label: UTC\n", config.ReasonMissing, firstClock},
		"an unknown zone":    {clock("ATL", "Europe/Atlantis"), config.ReasonUnknownZone, zonePath},
		"the machine's zone": {clock("HOME", "Local"), config.ReasonUnknownZone, zonePath},
		"a path":             {clock("ETC", "../etc/passwd"), config.ReasonUnknownZone, zonePath},
		"a number":           {clock("ONE", "1"), config.ReasonWrongType, zonePath},
		"an unknown field":   {clock("UTC", "UTC") + "    city: Berlin\n", config.ReasonUnknownField, firstClock},
		"a repeated label": {
			clock("BER", "Europe/Berlin") + clock("BER", "UTC"), config.ReasonDuplicateLabel, clocksPath + "[1].label",
		},
		"not a list": {"  label: UTC\n", config.ReasonWrongType, clocksPath},
	} {
		problem := requireProblem(t, withClocks(check.clocks), check.reason)
		if problem.Path != check.path || problem.Line == 0 {
			t.Errorf("%s: want %s at %s with its line, got %+v", name, check.reason, check.path, problem)
		}
	}
}

// TestDecode_ReportsEveryBadWorldClock: two bad clocks are both reported.
func TestDecode_ReportsEveryBadWorldClock(t *testing.T) {
	t.Parallel()

	_, decodeError := newTestDecoder(t).Decode([]byte(withClocks(
		"  - label: ATL\n    zone: Europe/Atlantis\n  - label: TOOLONG\n    zone: UTC\n")))

	if problems := problemsOf(t, decodeError); len(problems) != 2 {
		t.Fatalf("want both clocks reported, got %v", problems)
	}
}

// TestEncode_WritesWorldClocksOnlyWhenThereAreSome: world clocks round trip
// through the encoder, before the pages, and a file without them is written
// without them.
func TestEncode_WritesWorldClocksOnlyWhenThereAreSome(t *testing.T) {
	t.Parallel()

	decoder := newTestDecoder(t)

	for _, check := range []struct {
		file      string
		wantClock bool
	}{{withClocks(berlinAndAraraquara), true}, {validConfiguration, false}} {
		configuration, decodeError := decoder.Decode([]byte(check.file))
		if decodeError != nil {
			t.Fatal(decodeError)
		}

		encoded, encodeError := decoder.Encode(configuration)
		if encodeError != nil {
			t.Fatal(encodeError)
		}

		written := string(encoded)
		if strings.Contains(written, "world_clocks:") != check.wantClock ||
			(check.wantClock && strings.Index(written, "world_clocks:") > strings.Index(written, "pages:")) {
			t.Errorf("want world clocks written %v, before the pages, got:\n%s", check.wantClock, written)
		}

		again, againError := decoder.Decode(encoded)
		if againError != nil || !slices.Equal(again.WorldClocks, configuration.WorldClocks) {
			t.Errorf("want the world clocks to round trip, got %v (%v)", again.WorldClocks, againError)
		}
	}
}

// TestClone_CopiesTheWorldClocks: a copy shares no world clocks with the
// configuration it was copied from, and keeps none as none.
func TestClone_CopiesTheWorldClocks(t *testing.T) {
	t.Parallel()

	configuration, decodeError := newTestDecoder(t).Decode([]byte(withClocks(berlinAndAraraquara)))
	if decodeError != nil {
		t.Fatal(decodeError)
	}

	cloned := configuration.Clone()
	cloned.WorldClocks[0].Zone = "UTC"

	if configuration.WorldClocks[0].Zone != "Europe/Berlin" || len(cloned.WorldClocks) != 2 {
		t.Fatalf("want an independent copy, got %v and %v", configuration.WorldClocks, cloned.WorldClocks)
	}

	if decodeValidConfiguration(t).Clone().WorldClocks != nil {
		t.Fatal("want no world clocks cloned as none")
	}
}
