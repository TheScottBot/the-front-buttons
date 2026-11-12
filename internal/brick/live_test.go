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
