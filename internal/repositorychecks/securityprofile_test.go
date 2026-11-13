package repositorychecks_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/repositorychecks"
)

// TestUnlistedPrograms_FindsAProgramTheProfileMisses: SECURITY.md lists every
// program the daemon runs, so a program added to the code but not to the
// profile fails the build; test code is not the daemon's.
func TestUnlistedPrograms_FindsAProgramTheProfileMisses(t *testing.T) {
	t.Parallel()

	file := func(content string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte(content), Mode: 0, ModTime: time.Time{}, Sys: nil}
	}

	planted := fstest.MapFS{
		"internal/platform/fetch/fetch.go": file("package fetch\n\nconst curl = \"/usr/bin/curl\"\n" +
			"const shell = \"cmd.exe\"\nconst listed = \"/usr/bin/open\"\n"),
		"internal/platform/fetch/fetch_test.go": file("package fetch\n\nconst only = \"/usr/bin/false\"\n"),
		"SECURITY.md":                           file("Runs `/usr/bin/open`.\n"),
		"internal/bricks/apps/app.go":           file("package apps\n\nconst matched = \"goland64.exe\"\n"),
	}

	unlisted, scanError := repositorychecks.UnlistedPrograms(planted)
	if scanError != nil {
		t.Fatal(scanError)
	}

	if !slices.Equal(unlisted, []string{"/usr/bin/curl", "cmd.exe"}) {
		t.Errorf("want the two unlisted programs, got %v", unlisted)
	}
}

// TestSecurityProfile_ListsEveryProgramRun runs the check over this
// repository.
func TestSecurityProfile_ListsEveryProgramRun(t *testing.T) {
	t.Parallel()

	repositoryRoot, rootError := filepath.Abs(filepath.Join("..", ".."))
	if rootError != nil {
		t.Fatal(rootError)
	}

	unlisted, scanError := repositorychecks.UnlistedPrograms(os.DirFS(repositoryRoot))
	if scanError != nil {
		t.Fatal(scanError)
	}

	for _, program := range unlisted {
		t.Errorf("SECURITY.md does not list %s, which the code runs", program)
	}
}
