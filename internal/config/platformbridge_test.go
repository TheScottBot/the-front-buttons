package config_test

import (
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// withBridge is a valid file with a platform_bridge line before its pages.
func withBridge(value string) string {
	return strings.Replace(validConfiguration, "pages:\n", "platform_bridge: "+value+"\npages:\n", 1)
}

// TestDecode_ReadsPlatformBridge: platform_bridge names the bridge the daemon
// reaches the system through, cgo or purego, and is read as written.
func TestDecode_ReadsPlatformBridge(t *testing.T) {
	t.Parallel()

	for _, want := range []config.PlatformBridge{config.PlatformBridgeCgo, config.PlatformBridgePurego} {
		configuration, decodeError := newTestDecoder(t).Decode([]byte(withBridge(string(want))))
		if decodeError != nil {
			t.Fatal(decodeError)
		}

		if configuration.PlatformBridge != want || configuration.PlatformBridgeOrDefault() != want {
			t.Errorf("want %q, got %q (or default %q)",
				want, configuration.PlatformBridge, configuration.PlatformBridgeOrDefault())
		}
	}
}

// TestDecode_LeavesPlatformBridgeAbsent_MeaningCgo: a file without
// platform_bridge decodes with none, so saving it never writes one in, and
// the daemon takes cgo.
func TestDecode_LeavesPlatformBridgeAbsent_MeaningCgo(t *testing.T) {
	t.Parallel()

	configuration := decodeValidConfiguration(t)
	if configuration.PlatformBridge != "" {
		t.Fatalf("want no bridge decoded, got %q", configuration.PlatformBridge)
	}

	if got := configuration.PlatformBridgeOrDefault(); got != config.PlatformBridgeCgo {
		t.Fatalf("want cgo by default, got %q", got)
	}
}

// TestDecode_RefusesAnUnknownPlatformBridge: any value but cgo or purego
// refuses the whole file, naming the field, with the line it is on.
func TestDecode_RefusesAnUnknownPlatformBridge(t *testing.T) {
	t.Parallel()

	for name, check := range map[string]struct {
		value  string
		reason config.Reason
	}{
		"another bridge":   {"swift", config.ReasonUnknownBridge},
		"capitals":         {"Purego", config.ReasonUnknownBridge},
		"padded":           {`" cgo"`, config.ReasonUnknownBridge},
		"empty text":       {`""`, config.ReasonUnknownBridge},
		"a number":         {"1", config.ReasonWrongType},
		"a list":           {"[cgo]", config.ReasonWrongType},
		"nothing after it": {"", config.ReasonWrongType},
	} {
		problem := requireProblem(t, withBridge(check.value), check.reason)
		if problem.Path != "platform_bridge" || problem.Line == 0 || !strings.Contains(problem.Detail, "cgo") {
			t.Errorf("%s: want %s at platform_bridge with its line and the values allowed, got %+v",
				name, check.reason, problem)
		}
	}
}

// TestEncode_WritesPlatformBridgeOnlyWhenSet: the bridge round trips through
// the encoder, before the pages, and a file without it is written without it.
func TestEncode_WritesPlatformBridgeOnlyWhenSet(t *testing.T) {
	t.Parallel()

	decoder := newTestDecoder(t)

	for _, check := range []struct {
		file string
		want config.PlatformBridge
	}{
		{withBridge("purego"), config.PlatformBridgePurego},
		{withBridge("cgo"), config.PlatformBridgeCgo},
		{validConfiguration, ""},
	} {
		configuration, decodeError := decoder.Decode([]byte(check.file))
		if decodeError != nil {
			t.Fatal(decodeError)
		}

		encoded, encodeError := decoder.Encode(configuration.Clone())
		if encodeError != nil {
			t.Fatal(encodeError)
		}

		written := string(encoded)
		if strings.Contains(written, "platform_bridge:") != (check.want != "") ||
			strings.Index(written, "platform_bridge:") > strings.Index(written, "pages:") {
			t.Errorf("want the bridge %q written only when set, before the pages, got:\n%s", check.want, written)
		}

		reread, rereadError := decoder.Decode(encoded)
		if rereadError != nil {
			t.Fatal(rereadError)
		}

		if reread.PlatformBridge != check.want {
			t.Errorf("want %q after a round trip, got %q", check.want, reread.PlatformBridge)
		}
	}
}
