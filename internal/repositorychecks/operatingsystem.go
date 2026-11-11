package repositorychecks

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// permittedOperatingSystemDirectories are the only directories whose Go files
// may contain operating system specific code (specification section 2). Each
// entry covers that directory and everything beneath it, matched on whole path
// segments so a sibling sharing a prefix is not permitted by accident.
//
// Ports live in internal/platform and stay pure; their adapters live in
// internal/platform/systemadapter, so only the adapter directory is listed.
func permittedOperatingSystemDirectories() []string {
	return []string{
		"internal/platform/systemadapter",
		"internal/deck/hidtransport",
	}
}

// unixBuildTag is the build tag matching every Unix like operating system. It
// names operating systems in a build constraint but is not a file name suffix.
const unixBuildTag = "unix"

// knownOperatingSystems and knownArchitectures copy Go's own lists from
// internal/syslist (read from the go1.27.3 source on 11 November 2026), plus the
// unix build tag. They are copied deliberately: internal/syslist cannot be
// imported, and a build tag or file suffix naming any of these makes a file
// operating system specific.
func knownOperatingSystems() map[string]bool {
	return map[string]bool{
		"aix": true, "android": true, "darwin": true, "dragonfly": true, "freebsd": true,
		"hurd": true, "illumos": true, "ios": true, "js": true, "linux": true, "nacl": true,
		"netbsd": true, "openbsd": true, "plan9": true, "solaris": true, "wasip1": true,
		"windows": true, "zos": true, unixBuildTag: true,
	}
}

func knownArchitectures() map[string]bool {
	return map[string]bool{
		"386": true, "amd64": true, "amd64p32": true, "arm": true, "armbe": true, "arm64": true,
		"arm64be": true, "loong64": true, "mips": true, "mipsle": true, "mips64": true,
		"mips64le": true, "mips64p32": true, "mips64p32le": true, "ppc": true, "ppc64": true,
		"ppc64le": true, "riscv": true, "riscv64": true, "s390": true, "s390x": true,
		"sparc": true, "sparc64": true, "wasm": true,
	}
}

// isForbiddenOperatingSystemImport reports whether an import path reaches the
// operating system directly. cgo is included because hidapi through cgo is the
// one permitted native boundary, and os/exec because launching processes is the
// URL opener's and application launcher's job alone.
func isForbiddenOperatingSystemImport(importPath string) bool {
	switch importPath {
	case "syscall", "C", "os/exec", "golang.org/x/sys":
		return true
	default:
		return strings.HasPrefix(importPath, "golang.org/x/sys/")
	}
}

// ScanOperatingSystemUse reports operating system specific code in a Go file
// outside the packages permitted to hold it: an operating system import, a
// reference to runtime.GOOS, an operating system file name suffix, or a build
// constraint naming an operating system.
func ScanOperatingSystemUse(relativePath string, fileContent []byte) []Violation {
	if isInPermittedOperatingSystemDirectory(relativePath) {
		return nil
	}

	violations := scanOperatingSystemFileName(relativePath)

	fileSet := token.NewFileSet()

	parsedFile, parseError := parser.ParseFile(fileSet, relativePath, fileContent, parser.ParseComments)
	if parseError != nil {
		return append(violations, Violation{
			Path:       relativePath,
			LineNumber: 1,
			Rule:       RuleUnparseableGoFile,
			Detail:     fmt.Sprintf("cannot parse, so operating system use cannot be ruled out: %v", parseError),
		})
	}

	violations = append(violations, scanBuildConstraints(relativePath, fileSet, parsedFile)...)
	violations = append(violations, scanOperatingSystemImports(relativePath, fileSet, parsedFile)...)

	return append(violations, scanRuntimeGOOS(relativePath, fileSet, parsedFile)...)
}

func isInPermittedOperatingSystemDirectory(relativePath string) bool {
	fileDirectory := path.Dir(relativePath)

	for _, permittedDirectory := range permittedOperatingSystemDirectories() {
		if fileDirectory == permittedDirectory || strings.HasPrefix(fileDirectory, permittedDirectory+"/") {
			return true
		}
	}

	return false
}

// scanOperatingSystemFileName applies Go's file name rule: a name ending in
// _GOOS, _GOARCH or _GOOS_GOARCH, before any _test, is constrained to that
// target. Only the forms naming an operating system are reported.
func scanOperatingSystemFileName(relativePath string) []Violation {
	// A suffix needs a name before it: Go treats a file called windows.go as
	// unconstrained, so a name and a suffix are the fewest parts that count.
	const minimumPartsForSuffix = 2

	fileStem := strings.TrimSuffix(strings.TrimSuffix(path.Base(relativePath), ".go"), "_test")
	nameParts := strings.Split(fileStem, "_")

	operatingSystems := knownOperatingSystems()
	lastPart := nameParts[len(nameParts)-1]

	namesOperatingSystem := len(nameParts) >= minimumPartsForSuffix &&
		operatingSystems[lastPart] && lastPart != unixBuildTag
	if !namesOperatingSystem && len(nameParts) >= minimumPartsForSuffix+1 && knownArchitectures()[lastPart] {
		secondLastPart := nameParts[len(nameParts)-2]
		namesOperatingSystem = operatingSystems[secondLastPart] && secondLastPart != unixBuildTag
	}

	if !namesOperatingSystem {
		return nil
	}

	return []Violation{{
		Path:       relativePath,
		LineNumber: 1,
		Rule:       RuleOperatingSystemFile,
		Detail:     "file name suffix restricts this file to one operating system; such files belong in a permitted package",
	}}
}

func scanBuildConstraints(relativePath string, fileSet *token.FileSet, parsedFile *ast.File) []Violation {
	var violations []Violation

	for _, commentGroup := range parsedFile.Comments {
		if commentGroup.Pos() >= parsedFile.Package {
			break
		}

		for _, comment := range commentGroup.List {
			if !constraint.IsGoBuild(comment.Text) && !constraint.IsPlusBuild(comment.Text) {
				continue
			}

			constraintExpression, constraintError := constraint.Parse(comment.Text)
			if constraintError != nil || !constraintNamesOperatingSystem(constraintExpression) {
				continue
			}

			violations = append(violations, Violation{
				Path:       relativePath,
				LineNumber: fileSet.Position(comment.Pos()).Line,
				Rule:       RuleOperatingSystemFile,
				Detail:     "build constraint names an operating system; such files belong in a permitted package",
			})
		}
	}

	return violations
}

func constraintNamesOperatingSystem(constraintExpression constraint.Expr) bool {
	switch typedExpression := constraintExpression.(type) {
	case *constraint.TagExpr:
		return knownOperatingSystems()[typedExpression.Tag]
	case *constraint.NotExpr:
		return constraintNamesOperatingSystem(typedExpression.X)
	case *constraint.AndExpr:
		return constraintNamesOperatingSystem(typedExpression.X) || constraintNamesOperatingSystem(typedExpression.Y)
	case *constraint.OrExpr:
		return constraintNamesOperatingSystem(typedExpression.X) || constraintNamesOperatingSystem(typedExpression.Y)
	default:
		return false
	}
}

func scanOperatingSystemImports(relativePath string, fileSet *token.FileSet, parsedFile *ast.File) []Violation {
	var violations []Violation

	for _, importSpecification := range parsedFile.Imports {
		importPath, unquoteError := strconv.Unquote(importSpecification.Path.Value)
		if unquoteError != nil || !isForbiddenOperatingSystemImport(importPath) {
			continue
		}

		violations = append(violations, Violation{
			Path:       relativePath,
			LineNumber: fileSet.Position(importSpecification.Pos()).Line,
			Rule:       RuleOperatingSystemImport,
			Detail:     fmt.Sprintf("import %q reaches the operating system; only permitted packages may do so", importPath),
		})
	}

	return violations
}

// scanRuntimeGOOS finds runtime.GOOS under whatever name the runtime package was
// imported as. A dot import of runtime is reported outright, because it would
// let GOOS appear with no qualifier for this check to see.
func scanRuntimeGOOS(relativePath string, fileSet *token.FileSet, parsedFile *ast.File) []Violation {
	runtimeLocalName := ""

	for _, importSpecification := range parsedFile.Imports {
		if importSpecification.Path.Value != strconv.Quote("runtime") {
			continue
		}

		runtimeLocalName = "runtime"
		if importSpecification.Name != nil {
			runtimeLocalName = importSpecification.Name.Name
		}
	}

	if runtimeLocalName == "" {
		return nil
	}

	if runtimeLocalName == "." {
		return []Violation{{
			Path:       relativePath,
			LineNumber: 1,
			Rule:       RuleOperatingSystemCheck,
			Detail:     "dot import of runtime would hide runtime.GOOS from this check",
		}}
	}

	var violations []Violation

	ast.Inspect(parsedFile, func(visitedNode ast.Node) bool {
		selectorExpression, isSelector := visitedNode.(*ast.SelectorExpr)
		if !isSelector || selectorExpression.Sel.Name != "GOOS" {
			return true
		}

		qualifier, isIdentifier := selectorExpression.X.(*ast.Ident)
		if isIdentifier && qualifier.Name == runtimeLocalName {
			violations = append(violations, Violation{
				Path:       relativePath,
				LineNumber: fileSet.Position(selectorExpression.Pos()).Line,
				Rule:       RuleOperatingSystemCheck,
				Detail:     "runtime.GOOS outside a permitted package; inject behaviour through a port instead",
			})
		}

		return true
	})

	return violations
}
