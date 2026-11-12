package platform_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

func TestCheckLaunchTarget_AcceptsOrdinaryTargets(t *testing.T) {
	t.Parallel()

	for _, target := range []string{
		"https://discord.com/channels/1/2",
		"com.microsoft.teams2",
		`C:\Program Files\Thing\thing.exe`,
		"MSTeams_8wekyb3d8bbwe!MSTeams",
		"Visual Studio Code",
	} {
		checkError := platform.CheckLaunchTarget(target)
		if checkError != nil {
			t.Errorf("want %q accepted, got %v", target, checkError)
		}
	}
}

func TestCheckLaunchTarget_RefusesTargetsACommandCouldMisread(t *testing.T) {
	t.Parallel()

	refused := map[string]string{
		"empty":            "",
		"leading hyphen":   "-a Calculator",
		"newline":          "https://example.test/\nrm",
		"carriage return":  "https://example.test/\r",
		"nul":              "https://example.test/\x00",
		"over the length":  strings.Repeat("a", platform.MaximumLaunchTargetLength+1),
		"tab":              "a\tb",
		"escape":           "a\x1bb",
		"delete character": "a\x7fb",
	}

	for caseName, target := range refused {
		checkError := platform.CheckLaunchTarget(target)
		if !errors.Is(checkError, platform.ErrLaunchRefused) {
			t.Errorf("%s: want ErrLaunchRefused, got %v", caseName, checkError)
		}
	}
}
