// Package reference generates the brick reference in docs/BRICKS.md from the brick
// registry and the preset table, so the documentation is derived from the same
// declarations as validation, decoding and the editor's forms (specification
// section 2), and a check fails if the two ever differ.
package reference

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
)

// sectionHeading starts the section of docs/BRICKS.md this package owns.
const sectionHeading = "## Brick reference\n"

// nextSectionStart is how the following section begins.
const nextSectionStart = "\n## "

// ErrNoSection reports a document without the brick reference heading.
var ErrNoSection = errors.New("the document has no brick reference section")

// Markdown describes every kind and preset: each field's name, type, whether it
// is required, its bounds, its format and its description.
func Markdown(registry *brick.Registry, presetTable presets.Table) string {
	var generated strings.Builder

	generated.WriteString("\nGenerated from each kind's and preset's declaration. Do not edit by hand: run\n")
	generated.WriteString("`UPDATE_BRICKS=1 go test ./internal/reference/` after changing a declaration.\n")

	for _, kindName := range registry.KindNames() {
		kind, _ := registry.Kind(kindName)

		fmt.Fprintf(&generated, "\n### `%s`\n\nA brick kind.\n\n", kindName)
		writeFieldTable(&generated, kind.Schema())
	}

	for _, presetName := range presetTable.Names() {
		row, _ := presetTable.Row(presetName)

		fmt.Fprintf(&generated, "\n### `%s`\n\nA preset that expands into `%s`.", presetName, row.TargetKind())
		fmt.Fprintf(&generated, " Its action, which the editor shows, is `%s`.\n\n", row.Action())
		writeFieldTable(&generated, row.FieldsSchema())
	}

	generated.WriteString("\n")

	return generated.String()
}

func writeFieldTable(generated *strings.Builder, schema brick.Schema) {
	generated.WriteString("| Field | Type | Required | Bounds | Format | Description |\n")
	generated.WriteString("|---|---|---|---|---|---|\n")

	for _, field := range schema.Fields() {
		fieldType := string(field.Type)
		if field.Secret {
			fieldType += ", secret"
		}

		required := "optional"
		if field.Required {
			required = "required"
		}

		fmt.Fprintf(generated, "| `%s` | %s | %s | %s | %s | %s |\n",
			field.Name, fieldType, required, describeBounds(field), string(field.Format), field.Description)
	}
}

func describeBounds(field brick.Field) string {
	if field.Bounds == nil {
		return ""
	}

	switch field.Type { //nolint:exhaustive // booleans have no bounds, and the schema refuses bounds on them
	case brick.FieldTypeText:
		return fmt.Sprintf("%d to %d characters", field.Bounds.Minimum, field.Bounds.Maximum)
	case brick.FieldTypeDuration:
		return compactDuration(time.Duration(field.Bounds.Minimum)) + " to " +
			compactDuration(time.Duration(field.Bounds.Maximum))
	default:
		return strconv.FormatInt(field.Bounds.Minimum, 10) + " to " + strconv.FormatInt(field.Bounds.Maximum, 10)
	}
}

// compactDuration writes 1h and 5m rather than Go's 1h0m0s and 5m0s.
func compactDuration(duration time.Duration) string {
	written := duration.String()

	if trimmed, wholeMinutes := strings.CutSuffix(written, "m0s"); wholeMinutes {
		written = trimmed + "m"
	}

	if trimmed, wholeHours := strings.CutSuffix(written, "h0m"); wholeHours {
		written = trimmed + "h"
	}

	return written
}

// Section returns a document's current brick reference.
func Section(document string) (string, error) {
	start, end, findError := sectionBounds(document)
	if findError != nil {
		return "", findError
	}

	return document[start:end], nil
}

// ReplaceSection returns a document with its brick reference replaced.
func ReplaceSection(document string, generated string) (string, error) {
	start, end, findError := sectionBounds(document)
	if findError != nil {
		return "", findError
	}

	return document[:start] + generated + document[end:], nil
}

func sectionBounds(document string) (int, int, error) {
	headingAt := strings.Index(document, sectionHeading)
	if headingAt < 0 {
		return 0, 0, ErrNoSection
	}

	start := headingAt + len(sectionHeading)

	nextAt := strings.Index(document[start:], nextSectionStart)
	if nextAt < 0 {
		return start, len(document), nil
	}

	return start, start + nextAt + 1, nil
}
