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

// updateReadmeVariable rewrites the README's brick reference from the registry
// when set to 1.
const updateReadmeVariable = "UPDATE_README"

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

// readmePath is the repository's README, from this package's directory, where
// go test runs.
const readmePath = "../../README.md"

// TestREADME_BrickReferenceMatchesTheRegistry fails whenever a kind, field or
// preset changes without the README following (specification TE10).
func TestREADME_BrickReferenceMatchesTheRegistry(t *testing.T) {
	t.Parallel()

	readme, readError := os.ReadFile(readmePath)
	if readError != nil {
		t.Fatal(readError)
	}

	generated := builtInReference(t)

	if os.Getenv(updateReadmeVariable) == "1" {
		updated, replaceError := reference.ReplaceSection(string(readme), generated)
		if replaceError != nil {
			t.Fatal(replaceError)
		}

		// A constant path: the taint rule follows the README's contents, not the path.
		writeError := os.WriteFile(readmePath, []byte(updated), 0o600) //nolint:gosec // see above
		if writeError != nil {
			t.Fatal(writeError)
		}

		return
	}

	current, sectionError := reference.Section(string(readme))
	if sectionError != nil {
		t.Fatal(sectionError)
	}

	if current != generated {
		t.Fatalf("the README's brick reference is out of date; run %s=1 go test ./internal/reference/", updateReadmeVariable)
	}
}
