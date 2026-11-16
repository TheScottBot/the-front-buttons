//nolint:testpackage // the bridge's choice is seen in the unexported adapters' types
package systemadapter

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// cgoBuildLine is the constraint every file calling macOS through cgo carries,
// so a build without cgo leaves it out and takes the purego adapter instead.
const cgoBuildLine = "darwin && cgo"

// bridgeChoice is one macOS constructor, made through a bridge, and a test
// of whether what it made is the purego adapter.
type bridgeChoice struct {
	name   string
	make   func(t *testing.T, bridge Bridge) any
	isPure func(made any) bool
}

func isType[T any](made any) bool {
	_, matches := made.(T)

	return matches
}

func bridgeChoices() []bridgeChoice {
	return []bridgeChoice{
		{
			name:   "NewApplicationIcons",
			make:   func(_ *testing.T, bridge Bridge) any { return NewApplicationIcons(bridge) },
			isPure: isType[pureApplicationIcons],
		},
		{
			name:   "NewApplicationActivator",
			make:   func(_ *testing.T, bridge Bridge) any { return NewApplicationActivator(bridge) },
			isPure: isType[pureApplicationActivator],
		},
		{
			name:   "NewApplicationBadges",
			make:   func(_ *testing.T, bridge Bridge) any { return NewApplicationBadges(bridge) },
			isPure: isType[pureApplicationBadges],
		},
		{
			name:   "NewWindowCycler",
			make:   func(_ *testing.T, bridge Bridge) any { return NewWindowCycler(func() {}, bridge) },
			isPure: isType[pureWindowCycler],
		},
		{
			name: "NewKeyPresser",
			make: func(t *testing.T, bridge Bridge) any {
				t.Helper()

				return NewKeyPresser(t.Context(), func() {}, bridge)
			},
			isPure: isType[pureKeyPresser],
		},
		{
			name:   "NewClipboardWriter",
			make:   func(_ *testing.T, bridge Bridge) any { return NewClipboardWriter(bridge) },
			isPure: isType[pureClipboardWriter],
		},
		{
			name:   "NewApplicationQuitter",
			make:   func(_ *testing.T, bridge Bridge) any { return NewApplicationQuitter(bridge) },
			isPure: isType[pureApplicationQuitter],
		},
		{
			name: "AccessibilityTrusted",
			make: func(_ *testing.T, bridge Bridge) any { return accessibilityTrustedThrough(bridge) },
			isPure: func(made any) bool {
				return reflect.ValueOf(made).Pointer() == reflect.ValueOf(pureAccessibilityTrusted).Pointer()
			},
		},
	}
}

// TestConstructors_FollowTheBridge: purego always gives the purego adapter;
// cgo gives the cgo adapter in a build with cgo, and the purego adapter in a
// build without it. Making an adapter changes nothing on the desktop.
func TestConstructors_FollowTheBridge(t *testing.T) {
	t.Parallel()

	for _, choice := range bridgeChoices() {
		t.Run(choice.name, func(t *testing.T) {
			t.Parallel()

			if !choice.isPure(choice.make(t, BridgePurego)) {
				t.Errorf("purego: want the purego adapter, got %T", choice.make(t, BridgePurego))
			}

			if choice.isPure(choice.make(t, BridgeCgo)) == CgoAvailable {
				t.Errorf("cgo with CgoAvailable %v: got %T", CgoAvailable, choice.make(t, BridgeCgo))
			}
		})
	}
}

// TestPuregoKeyPresser_ChoosesAndWritesShortcuts_AsTheCgoAdapterDoes: these
// reach no system call, so the stub answers them already.
func TestPuregoKeyPresser_ChoosesAndWritesShortcuts_AsTheCgoAdapterDoes(t *testing.T) {
	t.Parallel()

	presser := newPureKeyPresser(t.Context(), func() {})
	shortcut := platform.Shortcut{Key: "F9", Control: false, Alt: false, Shift: true, Command: true}
	windowsHalf := platform.Shortcut{Key: "F9", Control: true, Alt: false, Shift: true, Command: false}

	pair := platform.ShortcutPair{Mac: shortcut, Windows: windowsHalf}
	if presser.ForThisPlatform(pair) != shortcut {
		t.Errorf("ForThisPlatform: want the macOS half, got %v", presser.ForThisPlatform(pair))
	}

	if presser.DescribeShortcut(shortcut) != MacOSShortcutText(shortcut) {
		t.Errorf("DescribeShortcut: want %q, got %q", MacOSShortcutText(shortcut), presser.DescribeShortcut(shortcut))
	}
}

// TestCgoFiles_CarryTheCgoBuildLine: Go itself leaves out a file importing
// "C" when cgo is off, so the build alone cannot tell a cgo file that lost its
// constraint. Every such file here must say so in its build line.
func TestCgoFiles_CarryTheCgoBuildLine(t *testing.T) {
	t.Parallel()

	sources, globError := filepath.Glob("*.go")
	if globError != nil {
		t.Fatal(globError)
	}

	var cgoFiles []string

	for _, source := range sources {
		file, parseError := parser.ParseFile(token.NewFileSet(), source, nil, parser.ImportsOnly|parser.ParseComments)
		if parseError != nil {
			t.Fatalf("%s: %v", source, parseError)
		}

		importsC := slices.ContainsFunc(file.Imports, func(spec *ast.ImportSpec) bool { return spec.Path.Value == `"C"` })
		if !importsC {
			continue
		}

		cgoFiles = append(cgoFiles, source)

		if line := buildLine(file); line != cgoBuildLine {
			t.Errorf("%s imports C: want the build line %q, got %q", source, cgoBuildLine, line)
		}
	}

	want := []string{
		"activation_darwin.go", "appicons_darwin.go", "badges_darwin.go", "clipboard_darwin.go",
		"keys_darwin.go", "quit_darwin.go", "windows_darwin.go",
	}
	if !slices.Equal(cgoFiles, want) {
		t.Errorf("want the cgo files %v, got %v", want, cgoFiles)
	}
}

// buildLine is a file's go:build constraint as written, or empty.
func buildLine(file *ast.File) string {
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}

		for _, comment := range group.List {
			if constraint.IsGoBuild(comment.Text) {
				expression, parseError := constraint.Parse(comment.Text)
				if parseError != nil {
					return comment.Text
				}

				return expression.String()
			}
		}
	}

	return ""
}

// TestPackage_BuildsAndVetsWithoutCgo vets this package, tests included, for
// macOS with cgo off, which compiles it without a C compiler: the cgo files
// left out and the purego adapters taking their place.
func TestPackage_BuildsAndVetsWithoutCgo(t *testing.T) {
	t.Parallel()

	goTool, lookError := exec.LookPath("go")
	if lookError != nil {
		t.Skipf("the go tool is not on the path: %v", lookError)
	}

	// The go tool found on the path, with fixed arguments and no shell.
	vet := exec.CommandContext(t.Context(), goTool, "vet", ".") //nolint:gosec // see above

	vet.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=darwin", "GOFLAGS=")

	output, vetError := vet.CombinedOutput()
	if vetError != nil {
		t.Fatalf("go vet with cgo off: %v\n%s", vetError, strings.TrimSpace(string(output)))
	}
}
