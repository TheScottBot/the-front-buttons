package systemadapter_test

import (
	"reflect"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// TestParseBadges_ReadsTheDocksRecords: each application's badge is read by
// its identifier in lower case, a pip as well as a count, and anything that
// is not a badge a key could show is left out.
func TestParseBadges_ReadsTheDocksRecords(t *testing.T) {
	t.Parallel()

	records := "com.hnc.Discord\x1f•\x1e" +
		"com.apple.mail\x1f12\x1e" +
		"com.example.long\x1f123456789012\x1e" +
		"com.example.blank\x1f \t\x1e" +
		"com.example.control\x1f4\a\x1e" +
		"\x1f3\x1e" +
		"no separator\x1e"

	want := map[string]string{
		"com.hnc.discord":     "•",
		"com.apple.mail":      "12",
		"com.example.long":    "12345678",
		"com.example.control": "4",
	}

	if got := systemadapter.ParseBadges(records); !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	if got := systemadapter.ParseBadges(""); len(got) != 0 {
		t.Errorf("want no badges from nothing, got %v", got)
	}
}
