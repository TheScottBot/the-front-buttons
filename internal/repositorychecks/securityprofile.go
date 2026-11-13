package repositorychecks

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

// securityProfilePath is the document that says what the daemon does and
// does not do; anyone may check it against the code.
const securityProfilePath = "SECURITY.md"

// The Go folders the daemon is built from.
var programRoots = []string{"internal", "cmd"} //nolint:gochecknoglobals // a fixed list

// UnlistedPrograms lists the programs the daemon's Go code names, as string
// literals of a fixed path under /usr/bin or /bin or a Windows .exe, that
// SECURITY.md does not mention. Test files are left out, as tests are not the
// daemon.
func UnlistedPrograms(repositoryFileSystem fs.FS) ([]string, error) {
	profile, readError := fs.ReadFile(repositoryFileSystem, securityProfilePath)
	if readError != nil {
		return nil, fmt.Errorf("reading %s: %w", securityProfilePath, readError)
	}

	var unlisted []string

	for _, root := range programRoots {
		programs, walkError := programsUnder(repositoryFileSystem, root)
		if walkError != nil {
			return nil, walkError
		}

		for _, program := range programs {
			if !strings.Contains(string(profile), program) && !slices.Contains(unlisted, program) {
				unlisted = append(unlisted, program)
			}
		}
	}

	slices.Sort(unlisted)

	return unlisted, nil
}

// programsUnder lists the programs named by the Go files under a folder,
// leaving out tests; a folder that is not there names none.
func programsUnder(repositoryFileSystem fs.FS, root string) ([]string, error) {
	var programs []string

	walkError := fs.WalkDir(repositoryFileSystem, root, func(path string, entry fs.DirEntry, entryError error) error {
		switch {
		case errors.Is(entryError, fs.ErrNotExist) && path == root:
			return fs.SkipDir
		case entryError != nil:
			return entryError
		case entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go"):
			return nil
		}

		named, namedError := programsNamed(repositoryFileSystem, path)
		programs = append(programs, named...)

		return namedError
	})
	if walkError != nil {
		return nil, fmt.Errorf("walking %s: %w", root, walkError)
	}

	return programs, nil
}

// programsNamed lists the program paths one Go file's string literals name.
func programsNamed(repositoryFileSystem fs.FS, path string) ([]string, error) {
	source, readError := fs.ReadFile(repositoryFileSystem, path)
	if readError != nil {
		return nil, fmt.Errorf("reading %s: %w", path, readError)
	}

	parsed, parseError := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if parseError != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, parseError)
	}

	var programs []string

	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, isLiteral := node.(*ast.BasicLit)
		if !isLiteral || literal.Kind != token.STRING {
			return true
		}

		value, unquoteError := strconv.Unquote(literal.Value)
		if unquoteError == nil && isProgramPath(value) {
			programs = append(programs, value)
		}

		return true
	})

	return programs, nil
}

// isProgramPath reports a single fixed program: a path under /usr/bin or /bin,
// or a file name ending .exe, with no space.
func isProgramPath(value string) bool {
	if strings.ContainsAny(value, " \n\t") {
		return false
	}

	for _, prefix := range []string{"/usr/bin/", "/bin/"} {
		if name, under := strings.CutPrefix(value, prefix); under {
			return name != "" && !strings.Contains(name, "/")
		}
	}

	return strings.HasSuffix(value, ".exe") && len(value) > len(".exe") && !strings.ContainsAny(value, "/\\")
}
