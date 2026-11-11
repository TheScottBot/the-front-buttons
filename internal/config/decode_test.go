package config_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// validConfiguration exercises every field type, both fake kinds, a preset and
// two pages, and is the file the round trip and store tests start from.
const validConfiguration = `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: test_link
        parameters:
          address: https://example.test/docs
          label: Docs
          background: "#102030"
      - row: 0
        column: 1
        kind: test_poller
        parameters:
          source: main-build
          every: 90s
          enabled: true
          retries: 3
          credential: github-work
          icon: brand-github
  - name: Play
    keys:
      - row: 2
        column: 4
        preset: test_channel
        fields:
          server_id: "123"
          channel_id: "456"
          label: Party
`

func newTestDecoder(t *testing.T) config.Decoder {
	t.Helper()

	registry, registryError := testsupport.NewRegistry()
	if registryError != nil {
		t.Fatalf("building the fake registry: %v", registryError)
	}

	presetTable, presetError := testsupport.NewPresetTable(registry)
	if presetError != nil {
		t.Fatalf("building the fake preset table: %v", presetError)
	}

	grid, gridError := layout.NewGrid(4, 8)
	if gridError != nil {
		t.Fatalf("building the grid: %v", gridError)
	}

	return config.NewDecoder(registry, presetTable, grid)
}

// problemsOf returns the problems of a rejection, failing the test if the error
// is anything else.
func problemsOf(t *testing.T, decodeError error) []config.Problem {
	t.Helper()

	rejection, isRejection := errors.AsType[*config.RejectionError](decodeError)
	if !isRejection {
		t.Fatalf("want a *config.RejectionError, got %v", decodeError)
	}

	return rejection.Problems
}

// requireProblem fails unless decoding the file is rejected with a problem of
// the given reason.
func requireProblem(t *testing.T, fileContent string, wantReason config.Reason) config.Problem {
	t.Helper()

	_, decodeError := newTestDecoder(t).Decode([]byte(fileContent))

	var matching config.Problem

	matched := false

	for _, problem := range problemsOf(t, decodeError) {
		if problem.Reason == wantReason {
			matching, matched = problem, true

			break
		}
	}

	if !matched {
		t.Fatalf("want a %s problem, got %v", wantReason, decodeError)
	}

	return matching
}

func decodeValidConfiguration(t *testing.T) config.Configuration {
	t.Helper()

	configuration, decodeError := newTestDecoder(t).Decode([]byte(validConfiguration))
	if decodeError != nil {
		t.Fatalf("decoding: %v", decodeError)
	}

	return configuration
}

func TestDecode_AcceptsEveryFieldType(t *testing.T) {
	t.Parallel()

	configuration := decodeValidConfiguration(t)

	if configuration.SchemaVersion != config.CurrentSchemaVersion || len(configuration.Pages) != 2 {
		t.Fatalf("unexpected shape: %+v", configuration)
	}

	poller, isPoller := configuration.Pages[0].Keys[1].Parameters.(*testsupport.PollerParameters)
	if !isPoller {
		t.Fatalf("want *PollerParameters, got %T", configuration.Pages[0].Keys[1].Parameters)
	}

	wantPoller := testsupport.PollerParameters{
		Icon: "brand-github", Label: "", Background: "",
		Source: "main-build", Every: 90 * time.Second, Enabled: true, Retries: 3, Credential: "github-work",
	}
	if *poller != wantPoller {
		t.Fatalf("poller parameters wrong: %+v", *poller)
	}
}

func TestDecode_ExpandsPresetKey_AndKeepsWhatWasWritten(t *testing.T) {
	t.Parallel()

	presetKey := decodeValidConfiguration(t).Pages[1].Keys[0]

	link, isLink := presetKey.Parameters.(*testsupport.LinkParameters)
	if !isLink || presetKey.Kind != testsupport.LinkKindName {
		t.Fatalf("want the expansion as a test_link key, got %+v", presetKey)
	}

	if link.Address != "https://chat.example.test/channels/123/456" || link.Label != "Party" {
		t.Fatalf("expansion wrong: %+v", link)
	}

	if presetKey.Preset == nil || presetKey.Preset.Name != testsupport.ChannelPresetName {
		t.Fatalf("want the preset source kept, got %+v", presetKey.Preset)
	}

	if presetKey.Position != (layout.Position{Row: 2, Column: 4}) {
		t.Fatalf("position wrong: %+v", presetKey.Position)
	}
}

func TestEncode_RoundTripsThroughDecode_KeepingPresetFields(t *testing.T) {
	t.Parallel()

	decoder := newTestDecoder(t)

	first, firstError := decoder.Decode([]byte(validConfiguration))
	if firstError != nil {
		t.Fatalf("decoding: %v", firstError)
	}

	encoded, encodeError := decoder.Encode(first)
	if encodeError != nil {
		t.Fatalf("encoding: %v", encodeError)
	}

	encodedText := string(encoded)
	if !strings.Contains(encodedText, "preset: test_channel") || strings.Contains(encodedText, "chat.example.test") {
		t.Fatalf("want the preset stored by name and fields, not expanded:\n%s", encoded)
	}

	second, secondError := decoder.Decode(encoded)
	if secondError != nil {
		t.Fatalf("decoding the encoded file: %v\n%s", secondError, encoded)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("round trip changed the configuration:\nfirst  %+v\nsecond %+v", first, second)
	}
}

func TestDecode_RejectsUnknownFieldsAtEveryLevel(t *testing.T) {
	t.Parallel()

	// insertAfter adds an extra line after the first occurrence of a line of the
	// valid configuration.
	insertAfter := func(existingLine string, extraLine string) string {
		return strings.Replace(validConfiguration, existingLine+"\n", existingLine+"\n"+extraLine+"\n", 1)
	}

	unknownFieldCases := map[string]string{
		"top level":    insertAfter("schema_version: 1", "colour: red"),
		"page":         insertAfter("  - name: Work", "    theme: dark"),
		"key":          insertAfter("        kind: test_link", "        size: 2"),
		"parameter":    insertAfter("          label: Docs", "          lable: Typo"),
		"preset field": insertAfter("          label: Party", "          guild: x"),
	}

	for caseName, fileContent := range unknownFieldCases {
		t.Run(caseName, func(t *testing.T) {
			t.Parallel()

			problem := requireProblem(t, fileContent, config.ReasonUnknownField)
			if problem.Line == 0 {
				t.Fatalf("want the problem to carry a line number, got %+v", problem)
			}
		})
	}
}

func TestDecode_RejectsUnknownKindAndUnknownPreset(t *testing.T) {
	t.Parallel()

	unknownKind := strings.Replace(validConfiguration, "kind: test_link", "kind: no_such_kind", 1)
	requireProblem(t, unknownKind, config.ReasonUnknownKind)

	unknownPreset := strings.Replace(validConfiguration, "preset: test_channel", "preset: no_such_preset", 1)
	requireProblem(t, unknownPreset, config.ReasonUnknownPreset)
}

func TestDecode_RejectsKeyNamingBothOrNeitherKindAndPreset(t *testing.T) {
	t.Parallel()

	both := strings.Replace(validConfiguration, "        preset: test_channel\n",
		"        preset: test_channel\n        kind: test_link\n", 1)
	requireProblem(t, both, config.ReasonKindOrPreset)

	neither := strings.Replace(validConfiguration, "        kind: test_link\n", "", 1)
	requireProblem(t, neither, config.ReasonKindOrPreset)
}

func TestDecode_RejectsReservedNavigationPositions(t *testing.T) {
	t.Parallel()

	for _, reservedColumn := range []string{"0", "7"} {
		fileContent := strings.Replace(validConfiguration, "      - row: 2\n        column: 4\n",
			"      - row: 3\n        column: "+reservedColumn+"\n", 1)

		problem := requireProblem(t, fileContent, config.ReasonReservedPosition)
		if !strings.Contains(problem.Detail, "navigation") {
			t.Errorf("want the reason to say the position is for navigation, got %q", problem.Detail)
		}
	}
}

func TestDecode_RejectsPositionOutsideTheGrid(t *testing.T) {
	t.Parallel()

	requireProblem(t, strings.Replace(validConfiguration, "        column: 4\n", "        column: 8\n", 1),
		config.ReasonOutsideGrid)
}

func TestDecode_RejectsTwoKeysAtOnePosition(t *testing.T) {
	t.Parallel()

	requireProblem(t, strings.Replace(validConfiguration, "        column: 1\n", "        column: 0\n", 1),
		config.ReasonDuplicatePosition)
}

func TestDecode_RejectsMissingRequiredParameter(t *testing.T) {
	t.Parallel()

	requireProblem(t, strings.Replace(validConfiguration, "          address: https://example.test/docs\n", "", 1),
		config.ReasonMissing)
}

func TestDecode_RejectsValuesOfTheWrongYAMLType(t *testing.T) {
	t.Parallel()

	wrongTypeCases := map[string]string{
		"number for text":    strings.Replace(validConfiguration, "label: Docs", "label: 12", 1),
		"word for boolean":   strings.Replace(validConfiguration, "enabled: true", "enabled: yes", 1),
		"text for integer":   strings.Replace(validConfiguration, "retries: 3", "retries: three", 1),
		"hex for integer":    strings.Replace(validConfiguration, "retries: 3", "retries: 0x3", 1),
		"number for row":     strings.Replace(validConfiguration, "      - row: 2\n", "      - row: two\n", 1),
		"bad duration":       strings.Replace(validConfiguration, "every: 90s", "every: soon", 1),
		"mapping for text":   strings.Replace(validConfiguration, "label: Docs", "label: {text: Docs}", 1),
		"sequence for pages": "schema_version: 1\npages: Work\n",
	}

	for caseName, fileContent := range wrongTypeCases {
		t.Run(caseName, func(t *testing.T) {
			t.Parallel()

			requireProblem(t, fileContent, config.ReasonWrongType)
		})
	}
}

func TestDecode_RejectsParameterOutsideItsBoundsOrFormat(t *testing.T) {
	t.Parallel()

	requireProblem(t, strings.Replace(validConfiguration, "retries: 3", "retries: 9", 1), config.ReasonOutOfBounds)
	requireProblem(t, strings.Replace(validConfiguration, "https://example.test/docs", "http://example.test/docs", 1),
		config.ReasonBadFormat)
}

func TestDecode_RejectsSchemaVersionThatIsMissingOrNewer(t *testing.T) {
	t.Parallel()

	requireProblem(t, strings.Replace(validConfiguration, "schema_version: 1\n", "", 1), config.ReasonMissing)
	requireProblem(t, strings.Replace(validConfiguration, "schema_version: 1", "schema_version: 2", 1),
		config.ReasonUnsupportedVersion)
	requireProblem(t, strings.Replace(validConfiguration, "schema_version: 1", "schema_version: 0", 1),
		config.ReasonUnsupportedVersion)
}

func TestDecode_RefusesFileOverTheByteBound_BeforeParsing(t *testing.T) {
	t.Parallel()

	// Unbalanced brackets would be a parse error; a too large rejection proves
	// the bound is checked before the parser ever sees the bytes.
	oversized := strings.Repeat("[", config.MaximumFileBytes+1)

	requireProblem(t, oversized, config.ReasonTooLarge)
}

func TestDecode_RejectsMalformedDuplicateAndMultipleDocuments(t *testing.T) {
	t.Parallel()

	requireProblem(t, "schema_version: [1\n", config.ReasonMalformed)
	requireProblem(t, "schema_version: 1\nschema_version: 1\npages: []\n", config.ReasonMalformed)
	requireProblem(t, validConfiguration+"\n...\n"+validConfiguration, config.ReasonMalformed)
	requireProblem(t, "", config.ReasonMalformed)
}

func TestDecode_RejectsMorePagesThanTheBound(t *testing.T) {
	t.Parallel()

	var fileContent strings.Builder

	fileContent.WriteString("schema_version: 1\npages:\n")

	for range config.MaximumPages + 1 {
		fileContent.WriteString("  - name: Page\n    keys: []\n")
	}

	requireProblem(t, fileContent.String(), config.ReasonTooMany)
}

func TestDecode_RejectsPageWithoutAName(t *testing.T) {
	t.Parallel()

	requireProblem(t, strings.Replace(validConfiguration, "  - name: Play\n    keys:\n", "  - keys:\n", 1),
		config.ReasonMissing)
	requireProblem(t, strings.Replace(validConfiguration, "name: Play", `name: ""`, 1), config.ReasonOutOfBounds)
}

func TestDecode_ReportsEveryProblemNotOnlyTheFirst(t *testing.T) {
	t.Parallel()

	fileContent := strings.Replace(validConfiguration, "retries: 3", "retries: 9", 1)
	fileContent = strings.Replace(fileContent, "kind: test_link", "kind: no_such_kind", 1)

	_, decodeError := newTestDecoder(t).Decode([]byte(fileContent))

	if problems := problemsOf(t, decodeError); len(problems) < 2 {
		t.Fatalf("want at least two problems, got %v", problems)
	}
}

func TestDecode_RejectionNeverEchoesAValue(t *testing.T) {
	t.Parallel()

	colleagueURL := "http://example.test/?users=colleague@example.test"
	fileContent := strings.Replace(validConfiguration, "https://example.test/docs", colleagueURL, 1)

	_, decodeError := newTestDecoder(t).Decode([]byte(fileContent))

	if decodeError == nil || strings.Contains(decodeError.Error(), "colleague") {
		t.Fatalf("want a rejection that names the field but not the value, got %v", decodeError)
	}
}
