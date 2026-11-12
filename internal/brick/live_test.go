package brick_test

import (
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

func TestCheckFormat_RepositoryWorkflowAndBranch(t *testing.T) {
	t.Parallel()

	formatCases := []struct {
		format brick.Format
		value  string
		want   bool
	}{
		{brick.FormatRepository, "octo-org/hello-world", true},
		{brick.FormatRepository, "octo-org/hello.world_2", true},
		{brick.FormatRepository, "octo-org", false},
		{brick.FormatRepository, "../etc", false},
		{brick.FormatRepository, "octo/org/extra", false},
		{brick.FormatToken, "ci.yml", true},
		{brick.FormatToken, "12345", true},
		{brick.FormatToken, "a/b", false},
		{brick.FormatToken, ".hidden", false},
		{brick.FormatBranch, "main", true},
		{brick.FormatBranch, "feature/new-key", true},
		{brick.FormatBranch, "feature/../main", false},
		{brick.FormatBranch, "has space", false},
	}

	for _, formatCase := range formatCases {
		if got := brick.CheckFormat(formatCase.format, formatCase.value); got != formatCase.want {
			t.Errorf("%s %q: want %v, got %v", formatCase.format, formatCase.value, formatCase.want, got)
		}
	}
}

func TestCheckFormat_RefusesAnUnknownFormat(t *testing.T) {
	t.Parallel()

	if brick.CheckFormat("no_such_format", "anything") {
		t.Fatal("want an unknown format to refuse every value")
	}
}

// TestCheckFormat_LinkURL_AcceptsHTTPSAndTheDiscordAppOnly covers the format a
// key's link must meet: https, or Discord's own scheme, which opens the app
// where an https link opens the browser (the author's decision amending TD13,
// evaluation, deck testing). Every other scheme stays refused.
func TestCheckFormat_LinkURL_AcceptsHTTPSAndTheDiscordAppOnly(t *testing.T) {
	t.Parallel()

	linkCases := map[string]bool{
		"https://example.com/":                     true,
		"https://discord.com/channels/1/2":         true,
		"discord://discord.com/channels/1/2":       true,
		"http://example.com/":                      false,
		"javascript:alert(1)":                      false,
		"file:///etc/passwd":                       false,
		"msteams://teams.microsoft.com/l/chat/0/0": false,
		"discord:channels/1/2":                     false,
		"discord:///channels/1/2":                  false,
		"discord://someone@discord.com/channels/1": false,
		"": false,
	}

	for value, want := range linkCases {
		if got := brick.CheckFormat(brick.FormatLinkURL, value); got != want {
			t.Errorf("CheckFormat(link_url, %q) = %v, want %v", value, got, want)
		}
	}
}

// TestCheckFormat_PNGFile_AcceptsAPlainFileNameOnly covers icon_file: a PNG in
// the icons folder beside the configuration, named by its file name alone, so a
// value can never reach any other file (the author's request, evaluation, deck
// testing).
func TestCheckFormat_PNGFile_AcceptsAPlainFileNameOnly(t *testing.T) {
	t.Parallel()

	fileCases := map[string]bool{
		"yeku.png":           true,
		"Uri_2.PNG":          true,
		"d-and-d.beyond.png": true,
		"../config.png":      false,
		"icons/yeku.png":     false,
		`icons\yeku.png`:     false,
		".hidden.png":        false,
		"yeku.jpg":           false,
		"yeku.png.exe":       false,
		"yeku..png":          false,
		".png":               false,
		"":                   false,
	}

	for value, want := range fileCases {
		if got := brick.CheckFormat(brick.FormatPNGFile, value); got != want {
			t.Errorf("CheckFormat(png_file, %q) = %v, want %v", value, got, want)
		}
	}
}

// TestCheckFormat_FunctionKey_AcceptsF13ToF24 covers the key a press_key key
// sends: the extended function keys the author asked for, which no keyboard
// has, so a press cannot collide with anything typed (evaluation, deck
// testing).
func TestCheckFormat_FunctionKey_AcceptsF13ToF24(t *testing.T) {
	t.Parallel()

	keyCases := map[string]bool{
		"F13": true, "F19": true, "F20": true, "F24": true,
		"F12": false, "F25": false, "f13": false, "F013": false, "F1": false, "": false,
	}

	if brick.CheckFormat(brick.FormatFunctionKey, "F13\n") {
		t.Error("want a trailing newline refused")
	}

	for value, want := range keyCases {
		if got := brick.CheckFormat(brick.FormatFunctionKey, value); got != want {
			t.Errorf("CheckFormat(function_key, %q) = %v, want %v", value, got, want)
		}
	}
}

// TestFormatChoices_ListsAFormatsEveryValue: a format with a short, fixed set
// of values lists them, so the editor can offer a choice rather than a text
// field where a typing slip breaks the key (the author's request, evaluation,
// deck testing). Every listed value passes its own format.
func TestFormatChoices_ListsAFormatsEveryValue(t *testing.T) {
	t.Parallel()

	choices := brick.FormatChoices(brick.FormatFunctionKey)

	want := []string{"F13", "F14", "F15", "F16", "F17", "F18", "F19", "F20", "F21", "F22", "F23", "F24"}
	if !slices.Equal(choices, want) {
		t.Fatalf("want %v, got %v", want, choices)
	}

	for _, choice := range choices {
		if !brick.CheckFormat(brick.FormatFunctionKey, choice) {
			t.Errorf("listed %s fails its own format", choice)
		}
	}

	if brick.FormatChoices(brick.FormatHTTPSURL) != nil {
		t.Error("want no list for an open ended format")
	}
}
