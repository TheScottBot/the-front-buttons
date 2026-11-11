package repositorychecks

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
)

// binaryDetectionPrefixLength is how much of a file is examined for a NUL byte
// when deciding whether it is binary, the same heuristic git uses, so the scan
// treats as text exactly what a diff would show as text.
const binaryDetectionPrefixLength = 8000

// gitDirectoryName is never scanned: it holds repository state, not tracked
// files.
const gitDirectoryName = ".git"

// vendoredPaths lists, file by file, the vendored third party sources exempt from
// the prose scan (engineering datum 0.8). It is an explicit list rather than a
// pattern because a pattern eventually exempts something that is not vendored.
// Nothing is vendored yet; TE4 adds the icon font and TE9 adds htmx.
func vendoredPaths() []string {
	return []string{}
}

// ScanRepository walks the tree under repositoryRoot and applies every check to
// every text file git would track: the prose scan to all of them, the operating
// system check to Go files, and the README check to the root README, whose absence
// is itself a violation. Paths the root .gitignore names are skipped.
func ScanRepository(repositoryRoot string) ([]Violation, error) {
	return scanFileSystem(os.DirFS(repositoryRoot), vendoredPaths())
}

// scanFileSystem does the walk over a filesystem rooted at the repository, so
// every path it sees is already relative and slash separated.
func scanFileSystem(repositoryFileSystem fs.FS, excludedPaths []string) ([]Violation, error) {
	rules, ignoreViolations, ignoreError := readIgnoreRules(repositoryFileSystem)
	if ignoreError != nil {
		return nil, ignoreError
	}

	scan := repositoryScan{
		repositoryFileSystem: repositoryFileSystem,
		excludedPathSet:      make(map[string]bool, len(excludedPaths)),
		rules:                rules,
		violations:           ignoreViolations,
		readmeFound:          false,
	}

	for _, excludedPath := range excludedPaths {
		scan.excludedPathSet[excludedPath] = true
	}

	walkError := fs.WalkDir(repositoryFileSystem, ".", scan.visitEntry)
	if walkError != nil {
		return nil, fmt.Errorf("walking repository: %w", walkError)
	}

	if !scan.readmeFound {
		scan.violations = append(scan.violations, missingReadmeViolation())
	}

	return scan.violations, nil
}

// repositoryScan accumulates what one walk of the tree finds.
type repositoryScan struct {
	repositoryFileSystem fs.FS
	excludedPathSet      map[string]bool
	rules                ignoreRules
	violations           []Violation
	readmeFound          bool
}

func (scan *repositoryScan) visitEntry(slashPath string, directoryEntry fs.DirEntry, entryError error) error {
	if entryError != nil {
		return entryError
	}

	if directoryEntry.IsDir() {
		if directoryEntry.Name() == gitDirectoryName || scan.rules.ignores(slashPath, true) {
			return fs.SkipDir
		}

		return nil
	}

	if scan.excludedPathSet[slashPath] || !directoryEntry.Type().IsRegular() || scan.rules.ignores(slashPath, false) {
		return nil
	}

	fileViolations, scanError := scanOneFile(scan.repositoryFileSystem, slashPath)
	if scanError != nil {
		return scanError
	}

	scan.violations = append(scan.violations, fileViolations...)
	scan.readmeFound = scan.readmeFound || slashPath == readmeRelativePath

	return nil
}

func missingReadmeViolation() Violation {
	return Violation{
		Path:       readmeRelativePath,
		LineNumber: 0,
		Rule:       RuleReadmeFirstLine,
		Detail:     "README.md is missing from the repository root",
	}
}

// scanOneFile applies every check that concerns the file at slashPath. Binary
// files have no prose and are skipped.
func scanOneFile(repositoryFileSystem fs.FS, slashPath string) ([]Violation, error) {
	fileContent, readError := fs.ReadFile(repositoryFileSystem, slashPath)
	if readError != nil {
		return nil, fmt.Errorf("reading %s: %w", slashPath, readError)
	}

	if isBinaryContent(fileContent) {
		return nil, nil
	}

	violations := ScanProse(slashPath, fileContent)

	if path.Ext(slashPath) == ".go" {
		violations = append(violations, ScanOperatingSystemUse(slashPath, fileContent)...)
	}

	if slashPath == readmeRelativePath {
		violations = append(violations, CheckReadmeFirstLine(fileContent)...)
	}

	return violations, nil
}

func isBinaryContent(fileContent []byte) bool {
	examinedPrefix := fileContent[:min(len(fileContent), binaryDetectionPrefixLength)]

	return bytes.IndexByte(examinedPrefix, 0) >= 0
}
