package repositorychecks

import (
	"os"
)

// ScanRepositoryExcluding exposes the walk with a caller supplied exclusion list
// to the external test package, so the exact path exclusion can be proven before
// anything is vendored.
func ScanRepositoryExcluding(repositoryRoot string, excludedPaths []string) ([]Violation, error) {
	return scanFileSystem(os.DirFS(repositoryRoot), excludedPaths)
}
