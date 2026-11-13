package reference_test

import (
	"os"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/allbricks"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
	"github.com/TheScottBot/the-front-buttons/internal/reference"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// updateBricksVariable rewrites the brick reference in docs/BRICKS.md from
// the registry when set to 1.
const updateBricksVariable = "UPDATE_BRICKS"

func builtInReference(t *testing.T) string {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := allbricks.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	presetTable, presetError := presets.BuiltIn(registry)
	if presetError != nil {
		t.Fatal(presetError)
	}

	return reference.Markdown(registry, presetTable)
}

func TestMarkdown_DescribesEveryKindFieldAndPreset_FromTheirSchemas(t *testing.T) {
	t.Parallel()

	registry, registryError := testsupport.NewRegistry()
	if registryError != nil {
		t.Fatal(registryError)
	}

	presetTable, presetError := testsupport.NewPresetTable(registry)
	if presetError != nil {
		t.Fatal(presetError)
	}

	generated := reference.Markdown(registry, presetTable)

	for _, want := range []string{
		"### `test_link`", "### `test_poller`", "| `address` | text | required |", "| `every` | duration |",
		"30s to 1h", "Where the fake link goes.", "### `test_channel`", "expands into `test_link`",
		"Its action, which the editor shows, is `open_test_channel`.",
		"| `credential` | text, secret |",
	} {
		if !strings.Contains(generated, want) {
			t.Errorf("want %q in the reference", want)
		}
	}
}

// bricksPath is the brick reference's document, from this package's
// directory, where go test runs. It moved out of the README to keep the README
// short.
const bricksPath = "../../docs/BRICKS.md"

// TestBrickReference_MatchesTheRegistry fails whenever a kind, field or preset
// changes without docs/BRICKS.md following (specification TE10).
func TestBrickReference_MatchesTheRegistry(t *testing.T) {
	t.Parallel()

	document, readError := os.ReadFile(bricksPath)
	if readError != nil {
		t.Fatal(readError)
	}

	generated := builtInReference(t)

	if os.Getenv(updateBricksVariable) == "1" {
		updated, replaceError := reference.ReplaceSection(string(document), generated)
		if replaceError != nil {
			t.Fatal(replaceError)
		}

		// A constant path: the taint rule follows the document's contents, not the path.
		writeError := os.WriteFile(bricksPath, []byte(updated), 0o600) //nolint:gosec // see above
		if writeError != nil {
			t.Fatal(writeError)
		}

		return
	}

	current, sectionError := reference.Section(string(document))
	if sectionError != nil {
		t.Fatal(sectionError)
	}

	if current != generated {
		t.Fatalf("docs/BRICKS.md is out of date; run %s=1 go test ./internal/reference/", updateBricksVariable)
	}
}
