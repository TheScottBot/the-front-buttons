package repositorychecks_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/repositorychecks"
)

// writeTreeFile writes one file into a temporary tree, creating its directories.
func writeTreeFile(t *testing.T, treeRoot string, relativePath string, fileContent []byte) {
	t.Helper()

	absolutePath := filepath.Join(treeRoot, filepath.FromSlash(relativePath))

	makeDirectoryError := os.MkdirAll(filepath.Dir(absolutePath), 0o750)
	if makeDirectoryError != nil {
		t.Fatalf("creating directory for %s: %v", relativePath, makeDirectoryError)
	}

	writeError := os.WriteFile(absolutePath, fileContent, 0o600)
	if writeError != nil {
		t.Fatalf("writing %s: %v", relativePath, writeError)
	}
}

func TestScanRepository_ReportsEveryCheck_WhenTreeHasPlantedViolations(t *testing.T) {
	t.Parallel()

	treeRoot := t.TempDir()
	writeTreeFile(t, treeRoot, "README.md", []byte("A changed first line\n"))
	writeTreeFile(t, treeRoot, "docs/notes.md", []byte("one "+plantedEnDash+" two\n"))
	writeTreeFile(t, treeRoot, "internal/live/hub.go", goSourceUsingRuntimeGOOS())

	violations, scanError := repositorychecks.ScanRepository(treeRoot)
	if scanError != nil {
		t.Fatalf("scan failed: %v", scanError)
	}

	wantedRules := []repositorychecks.RuleName{
		repositorychecks.RuleReadmeFirstLine,
		repositorychecks.RuleDashCharacter,
		repositorychecks.RuleOperatingSystemCheck,
	}

	for _, wantedRule := range wantedRules {
		if !containsRule(violations, wantedRule) {
			t.Errorf("want a %s violation, got %v", wantedRule, violations)
		}
	}
}

func TestScanRepository_ReportsSlashSeparatedRelativePaths(t *testing.T) {
	t.Parallel()

	treeRoot := t.TempDir()
	writeTreeFile(t, treeRoot, "README.md", []byte(repositorychecks.SettledDescription+"\n"))
	writeTreeFile(t, treeRoot, "docs/deep/notes.md", []byte("one "+plantedEnDash+" two\n"))

	violations, scanError := repositorychecks.ScanRepository(treeRoot)
	if scanError != nil {
		t.Fatalf("scan failed: %v", scanError)
	}

	if len(violations) != 1 || violations[0].Path != "docs/deep/notes.md" {
		t.Fatalf("want one violation at docs/deep/notes.md, got %v", violations)
	}
}

func TestScanRepository_ReportsMissingReadme(t *testing.T) {
	t.Parallel()

	treeRoot := t.TempDir()
	writeTreeFile(t, treeRoot, "docs/notes.md", []byte("Clean.\n"))

	violations, scanError := repositorychecks.ScanRepository(treeRoot)
	if scanError != nil {
		t.Fatalf("scan failed: %v", scanError)
	}

	if !containsRule(violations, repositorychecks.RuleReadmeFirstLine) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleReadmeFirstLine, violations)
	}
}

func TestScanRepository_IgnoresGitDirectoryAndBinaryFiles(t *testing.T) {
	t.Parallel()

	treeRoot := t.TempDir()
	writeTreeFile(t, treeRoot, "README.md", []byte(repositorychecks.SettledDescription+"\n"))
	writeTreeFile(t, treeRoot, ".git/COMMIT_EDITMSG", []byte("one "+plantedEnDash+" two\n"))
	writeTreeFile(t, treeRoot, "assets/icon.png", append([]byte{0x89, 'P', 'N', 'G', 0x00}, []byte(plantedEnDash)...))

	violations, scanError := repositorychecks.ScanRepository(treeRoot)
	if scanError != nil {
		t.Fatalf("scan failed: %v", scanError)
	}

	if len(violations) != 0 {
		t.Fatalf("want no violations, got %v", violations)
	}
}

func TestScanRepository_SkipsExactVendoredPathsOnly(t *testing.T) {
	t.Parallel()

	treeRoot := t.TempDir()
	writeTreeFile(t, treeRoot, "README.md", []byte(repositorychecks.SettledDescription+"\n"))
	writeTreeFile(t, treeRoot, "third_party/upstream/notes.md", []byte("one "+plantedEnDash+" two\n"))
	writeTreeFile(t, treeRoot, "third_party/upstream/notes.md.orig", []byte("one "+plantedEnDash+" two\n"))

	violations, scanError := repositorychecks.ScanRepositoryExcluding(treeRoot, []string{"third_party/upstream/notes.md"})
	if scanError != nil {
		t.Fatalf("scan failed: %v", scanError)
	}

	if len(violations) != 1 || violations[0].Path != "third_party/upstream/notes.md.orig" {
		t.Fatalf("want exactly the unlisted file reported, got %v", violations)
	}
}

func TestScanRepository_ReturnsError_WhenRootDoesNotExist(t *testing.T) {
	t.Parallel()

	_, scanError := repositorychecks.ScanRepository(filepath.Join(t.TempDir(), "absent"))

	if !errors.Is(scanError, os.ErrNotExist) {
		t.Fatalf("want an error wrapping os.ErrNotExist, got %v", scanError)
	}
}

// TestRepository_HasNoViolations runs every check over this repository. It is
// the test continuous integration relies on, and it is trusted only because the
// tests above prove each check fails on a planted violation.
func TestRepository_HasNoViolations(t *testing.T) {
	t.Parallel()

	repositoryRoot, rootError := filepath.Abs(filepath.Join("..", ".."))
	if rootError != nil {
		t.Fatalf("resolving repository root: %v", rootError)
	}

	violations, scanError := repositorychecks.ScanRepository(repositoryRoot)
	if scanError != nil {
		t.Fatalf("scan failed: %v", scanError)
	}

	for _, violation := range violations {
		t.Error(violation.String())
	}
}
