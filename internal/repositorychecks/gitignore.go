package repositorychecks

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// gitignoreRelativePath is the one ignore file the scan reads. Nested ignore
// files are not read; the only one in the tree today lives inside an ignored
// editor directory, where it can never matter.
const gitignoreRelativePath = ".gitignore"

// ignorePattern is one line of the root .gitignore, in the subset of git's
// grammar this scan supports: a single path segment glob, optionally anchored to
// the repository root with a leading slash, optionally limited to directories
// with a trailing slash.
type ignorePattern struct {
	segmentGlob   string
	anchored      bool
	directoryOnly bool
}

// ignoreRules lets the scan cover the same files git tracks: datum 0.8 asks for
// every tracked text file, and an ignored file is never tracked. The patterns are
// read from .gitignore itself so the list of ignored paths lives in one place.
type ignoreRules struct {
	patterns []ignorePattern
}

// readIgnoreRules reads the root .gitignore, if there is one. A line outside the
// supported subset is reported rather than approximated, because a guessed
// pattern would quietly scan files git ignores or skip files git tracks.
func readIgnoreRules(repositoryFileSystem fs.FS) (ignoreRules, []Violation, error) {
	gitignoreContent, readError := fs.ReadFile(repositoryFileSystem, gitignoreRelativePath)
	if errors.Is(readError, fs.ErrNotExist) {
		return ignoreRules{patterns: nil}, nil, nil
	}

	if readError != nil {
		return ignoreRules{patterns: nil}, nil, fmt.Errorf("reading %s: %w", gitignoreRelativePath, readError)
	}

	rules := ignoreRules{patterns: nil}

	var violations []Violation

	for lineIndex, lineText := range strings.Split(string(gitignoreContent), "\n") {
		trimmedLine := strings.TrimSpace(lineText)
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		pattern, unsupportedReason := parseIgnorePattern(trimmedLine)
		if unsupportedReason != "" {
			violations = append(violations, Violation{
				Path:       gitignoreRelativePath,
				LineNumber: lineIndex + 1,
				Rule:       RuleUnsupportedIgnorePattern,
				Detail: fmt.Sprintf(
					"pattern %q is outside the subset the repository checks read: %s",
					trimmedLine, unsupportedReason,
				),
			})

			continue
		}

		rules.patterns = append(rules.patterns, pattern)
	}

	return rules, violations, nil
}

// parseIgnorePattern returns the pattern, or a reason it is not supported.
func parseIgnorePattern(trimmedLine string) (ignorePattern, string) {
	unsupported := ignorePattern{segmentGlob: "", anchored: false, directoryOnly: false}

	switch {
	case strings.HasPrefix(trimmedLine, "!"):
		return unsupported, "negation is not supported"
	case strings.Contains(trimmedLine, "**"):
		return unsupported, "a double star is not supported"
	case strings.Contains(trimmedLine, "\\"):
		return unsupported, "escapes are not supported"
	}

	pattern := ignorePattern{
		segmentGlob:   strings.TrimSuffix(strings.TrimPrefix(trimmedLine, "/"), "/"),
		anchored:      strings.HasPrefix(trimmedLine, "/"),
		directoryOnly: strings.HasSuffix(trimmedLine, "/"),
	}

	if pattern.segmentGlob == "" || strings.Contains(pattern.segmentGlob, "/") {
		return unsupported, "only a single path segment is supported"
	}

	_, globError := path.Match(pattern.segmentGlob, "")
	if globError != nil {
		return unsupported, "not a valid glob"
	}

	return pattern, ""
}

// ignores reports whether git would ignore the entry at slashPath. A pattern
// without a trailing slash matches files and directories alike, as in git.
func (rules ignoreRules) ignores(slashPath string, isDirectory bool) bool {
	for _, pattern := range rules.patterns {
		if pattern.directoryOnly && !isDirectory {
			continue
		}

		matchedName := path.Base(slashPath)
		if pattern.anchored {
			// An anchored single segment can only match an entry at the root.
			matchedName = slashPath
		}

		if matched, _ := path.Match(pattern.segmentGlob, matchedName); matched {
			return true
		}
	}

	return false
}
