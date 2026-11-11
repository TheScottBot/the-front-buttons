package repositorychecks_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/repositorychecks"
)

// scanTreeWithGitignore builds a tree holding a clean README, the given
// .gitignore and the given files, each file carrying a planted en dash, and scans
// it.
func scanTreeWithGitignore(
	t *testing.T,
	gitignoreContent string,
	plantedPaths []string,
) []repositorychecks.Violation {
	t.Helper()

	treeRoot := t.TempDir()
	writeTreeFile(t, treeRoot, "README.md", []byte(repositorychecks.SettledDescription+"\n"))
	writeTreeFile(t, treeRoot, ".gitignore", []byte(gitignoreContent))

	for _, plantedPath := range plantedPaths {
		writeTreeFile(t, treeRoot, plantedPath, []byte("one "+plantedEnDash+" two\n"))
	}

	violations, scanError := repositorychecks.ScanRepository(treeRoot)
	if scanError != nil {
		t.Fatalf("scan failed: %v", scanError)
	}

	return violations
}

// reportedPaths lists the distinct paths that carry a violation.
func reportedPaths(violations []repositorychecks.Violation) map[string]bool {
	paths := make(map[string]bool, len(violations))
	for _, violation := range violations {
		paths[violation.Path] = true
	}

	return paths
}

func TestScanRepository_SkipsDirectory_WhenGitignoreNamesItAtAnyDepth(t *testing.T) {
	t.Parallel()

	violations := scanTreeWithGitignore(t, ".idea/\n", []string{".idea/workspace.xml", "nested/.idea/misc.xml"})

	if len(violations) != 0 {
		t.Fatalf("want ignored directories skipped, got %v", violations)
	}
}

func TestScanRepository_SkipsFile_WhenGitignoreGlobMatchesItsName(t *testing.T) {
	t.Parallel()

	violations := scanTreeWithGitignore(t, "*.log\n.DS_Store\n", []string{"daemon.log", "deep/dir/.DS_Store"})

	if len(violations) != 0 {
		t.Fatalf("want ignored files skipped, got %v", violations)
	}
}

func TestScanRepository_SkipsOnlyRootPath_WhenGitignorePatternIsAnchored(t *testing.T) {
	t.Parallel()

	violations := scanTreeWithGitignore(t, "/dist/\n", []string{"dist/notes.md", "docs/dist/notes.md"})

	paths := reportedPaths(violations)
	if paths["dist/notes.md"] || !paths["docs/dist/notes.md"] {
		t.Fatalf("want only the root dist skipped, got %v", violations)
	}
}

func TestScanRepository_ScansFile_WhenDirectoryPatternOnlySharesItsName(t *testing.T) {
	t.Parallel()

	violations := scanTreeWithGitignore(t, "cache/\n", []string{"cache"})

	if !reportedPaths(violations)["cache"] {
		t.Fatalf("want a file named like an ignored directory still scanned, got %v", violations)
	}
}

func TestScanRepository_ScansEverything_WhenGitignoreHoldsOnlyCommentsAndBlankLines(t *testing.T) {
	t.Parallel()

	violations := scanTreeWithGitignore(t, "# a comment\n\n   \n", []string{"notes.md"})

	if !reportedPaths(violations)["notes.md"] {
		t.Fatalf("want comments and blank lines to ignore nothing, got %v", violations)
	}
}

func TestScanRepository_ReportsGitignorePattern_WhenGrammarIsUnsupported(t *testing.T) {
	t.Parallel()

	unsupportedPatterns := []string{"!keep.md", "**/generated", "docs/private.md", "trailing\\ space"}

	for _, unsupportedPattern := range unsupportedPatterns {
		t.Run(unsupportedPattern, func(t *testing.T) {
			t.Parallel()

			violations := scanTreeWithGitignore(t, unsupportedPattern+"\n", nil)

			if !containsRule(violations, repositorychecks.RuleUnsupportedIgnorePattern) {
				t.Fatalf("want a %s violation for %q, got %v",
					repositorychecks.RuleUnsupportedIgnorePattern, unsupportedPattern, rulesOf(violations))
			}
		})
	}
}
