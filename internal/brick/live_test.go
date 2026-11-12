package brick_test

import (
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
