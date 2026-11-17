package repositorychecks

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// hiddenProgramHelperPath is the one Windows file allowed to start a program
// itself. The daemon on Windows is a windowed program with no console, so any
// console program it starts, such as powershell.exe, gets a console window of
// its own that flashes on screen; the helper starts every program with that
// window suppressed.
const hiddenProgramHelperPath = "internal/platform/systemadapter/hiddenprogram_windows.go"

// windowsBuildTag is the build tag and file name suffix for Windows.
const windowsBuildTag = "windows"

// processStartingCalls are the functions that start a program, by package.
func processStartingCalls() map[string]map[string]bool {
	return map[string]map[string]bool{
		"os/exec": {"Command": true, "CommandContext": true},
		"os":      {"StartProcess": true},
	}
}

// ScanWindowsProcessStarts reports a program started directly in non test Go
// code built only for Windows, outside the hidden program helper. Such a start
// shows a console window on every call.
func ScanWindowsProcessStarts(relativePath string, fileContent []byte) []Violation {
	if relativePath == hiddenProgramHelperPath || strings.HasSuffix(relativePath, "_test.go") {
		return nil
	}

	fileSet := token.NewFileSet()

	parsedFile, parseError := parser.ParseFile(fileSet, relativePath, fileContent, parser.ParseComments)
	if parseError != nil || !isWindowsOnly(relativePath, parsedFile) {
		return nil
	}

	localNames := processPackageNames(parsedFile)

	var violations []Violation

	ast.Inspect(parsedFile, func(visitedNode ast.Node) bool {
		selectorExpression, isSelector := visitedNode.(*ast.SelectorExpr)
		if !isSelector {
			return true
		}

		qualifier, isIdentifier := selectorExpression.X.(*ast.Ident)
		if !isIdentifier || !processStartingCalls()[localNames[qualifier.Name]][selectorExpression.Sel.Name] {
			return true
		}

		violations = append(violations, Violation{
			Path:       relativePath,
			LineNumber: fileSet.Position(selectorExpression.Pos()).Line,
			Rule:       RuleWindowsVisibleProcess,
			Detail: "starts a program that would flash a console window on Windows; " +
				"start it through hiddenProgram instead",
		})

		return true
	})

	return violations
}

// processPackageNames maps the name each package that starts programs is
// imported under, renamed or not, to its import path.
func processPackageNames(parsedFile *ast.File) map[string]string {
	localNames := map[string]string{}

	for _, importSpecification := range parsedFile.Imports {
		importPath, unquoteError := strconv.Unquote(importSpecification.Path.Value)
		if unquoteError != nil || processStartingCalls()[importPath] == nil {
			continue
		}

		localName := path.Base(importPath)
		if importSpecification.Name != nil {
			localName = importSpecification.Name.Name
		}

		localNames[localName] = importPath
	}

	return localNames
}

// isWindowsOnly reports whether a file is built only for Windows, by its name
// suffix or a build constraint that holds for Windows alone.
func isWindowsOnly(relativePath string, parsedFile *ast.File) bool {
	stem := strings.TrimSuffix(path.Base(relativePath), ".go")
	if strings.HasSuffix(stem, "_windows") || strings.Contains(stem, "_windows_") {
		return true
	}

	for _, commentGroup := range parsedFile.Comments {
		if commentGroup.Pos() >= parsedFile.Package {
			break
		}

		for _, comment := range commentGroup.List {
			expression, parseError := constraint.Parse(comment.Text)
			if parseError != nil {
				continue
			}

			if expression.Eval(func(tag string) bool { return tag == windowsBuildTag }) &&
				!expression.Eval(func(tag string) bool { return tag != windowsBuildTag && knownOperatingSystems()[tag] }) {
				return true
			}
		}
	}

	return false
}
