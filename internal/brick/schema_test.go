package brick_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// Field names of sampleParameters, named once for every test in this package.
const (
	labelFieldName      = "label"
	iconFieldName       = "icon"
	backgroundFieldName = "background"
	addressFieldName    = "address"
	countFieldName      = "count"
	enabledFieldName    = "enabled"
	intervalFieldName   = "interval"
	credentialFieldName = "credential"
	sampleCredential    = "github-work"
	sampleAddress       = "https://example.test/"
	docsLabel           = "Docs"
)

func deriveSampleSchema(t *testing.T) brick.Schema {
	t.Helper()

	schema, deriveError := brick.DeriveSchema[sampleParameters]()
	if deriveError != nil {
		t.Fatalf("deriving: %v", deriveError)
	}

	return schema
}

func TestDeriveSchema_ListsEveryFieldInDeclarationOrder_IncludingAppearance(t *testing.T) {
	t.Parallel()

	fields := deriveSampleSchema(t).Fields()

	fieldNames := make([]string, 0, len(fields))
	for _, field := range fields {
		fieldNames = append(fieldNames, field.Name)
	}

	want := []string{
		labelFieldName, iconFieldName, backgroundFieldName, "icon_file", addressFieldName,
		countFieldName, enabledFieldName, intervalFieldName, credentialFieldName,
	}
	if !reflect.DeepEqual(fieldNames, want) {
		t.Fatalf("want fields %v, got %v", want, fieldNames)
	}
}

// fieldFacts is the comparable projection of a Field, leaving out the struct
// index the schema keeps privately.
type fieldFacts struct {
	Type        brick.FieldType
	Required    bool
	Bounds      *brick.Bounds
	Format      brick.Format
	Description string
	Secret      bool
}

func factsOf(field brick.Field) fieldFacts {
	return fieldFacts{
		Type:        field.Type,
		Required:    field.Required,
		Bounds:      field.Bounds,
		Format:      field.Format,
		Description: field.Description,
		Secret:      field.Secret,
	}
}

func TestDeriveSchema_CarriesEveryTagFact(t *testing.T) {
	t.Parallel()

	schema := deriveSampleSchema(t)

	wantFacts := map[string]fieldFacts{
		addressFieldName: {
			Type: brick.FieldTypeText, Required: true, Bounds: &brick.Bounds{Minimum: 1, Maximum: 200},
			Format: brick.FormatHTTPSURL, Description: "Where the key goes.", Secret: false,
		},
		intervalFieldName: {
			Type: brick.FieldTypeDuration, Required: false,
			Bounds: &brick.Bounds{Minimum: int64(30 * time.Second), Maximum: int64(time.Hour)},
			Format: "", Description: "How often it runs.", Secret: false,
		},
		credentialFieldName: {
			Type: brick.FieldTypeText, Required: false, Bounds: &brick.Bounds{Minimum: 1, Maximum: 64},
			Format: brick.FormatIdentifier, Description: "Name of a stored credential.", Secret: true,
		},
		enabledFieldName: {
			Type: brick.FieldTypeBoolean, Required: false, Bounds: nil,
			Format: "", Description: "Whether it is on.", Secret: false,
		},
	}

	for fieldName, want := range wantFacts {
		field, fieldFound := schema.Field(fieldName)
		if !fieldFound {
			t.Errorf("want a %s field", fieldName)

			continue
		}

		if got := factsOf(field); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: want %+v, got %+v", fieldName, want, got)
		}
	}
}

func TestDeriveSchema_EnforcesEveryTagRule(t *testing.T) {
	t.Parallel()

	const (
		boundsMessage  = "bounds"
		defaultMessage = "default"
	)

	declarationCases := []struct {
		name           string
		parametersType reflect.Type
		wantInMessage  string
	}{
		{"not a struct", reflect.TypeFor[string](), "struct"},
		{"missing brick tag", reflect.TypeFor[missingBrickTag](), "brick tag"},
		{"unexported field", reflect.TypeFor[unexportedField](), "unexported"},
		{"bad field name", reflect.TypeFor[badFieldName](), "name"},
		{"duplicate field name", reflect.TypeFor[duplicateFieldName](), "duplicate"},
		{"duplicate through appearance", reflect.TypeFor[duplicateOfAppearanceField](), "duplicate"},
		{"missing description", reflect.TypeFor[missingDescription](), "description"},
		{"unsupported type", reflect.TypeFor[unsupportedType](), "type"},
		{"unknown tag option", reflect.TypeFor[unknownTagOption](), "option"},
		{"bounds on boolean", reflect.TypeFor[boundsOnBoolean](), boundsMessage},
		{"malformed bounds", reflect.TypeFor[malformedBounds](), boundsMessage},
		{"inverted bounds", reflect.TypeFor[invertedBounds](), boundsMessage},
		{"negative text bounds", reflect.TypeFor[negativeTextBounds](), boundsMessage},
		{"format on integer", reflect.TypeFor[formatOnInteger](), "format"},
		{"unknown format", reflect.TypeFor[unknownFormat](), "format"},
		{"secret on integer", reflect.TypeFor[secretOnInteger](), "secret"},
		{"unknown suggestions", reflect.TypeFor[unknownSuggestions](), "suggest"},
		{"suggestions on integer", reflect.TypeFor[suggestionsOnInteger](), "suggest"},
		{"replaces an unknown field", reflect.TypeFor[replacesUnknownField](), "replaces"},
		{"replaces on text", reflect.TypeFor[replacesOnText](), "replaces"},
		{"untagged embedded non appearance struct", reflect.TypeFor[embeddedOtherStruct](), "embedded"},
		{"default on text", reflect.TypeFor[defaultOnText](), defaultMessage},
		{"default out of bounds", reflect.TypeFor[defaultOutOfBounds](), defaultMessage},
		{"default not a number", reflect.TypeFor[defaultNotANumber](), defaultMessage},
		{"platform labels on text", reflect.TypeFor[platformLabelOnText](), "platform"},
		{"a Mac label alone", reflect.TypeFor[macLabelAlone](), "platform"},
	}

	for _, declarationCase := range declarationCases {
		t.Run(declarationCase.name, func(t *testing.T) {
			t.Parallel()

			_, deriveError := brick.DeriveSchemaOfType(declarationCase.parametersType)

			if !errors.Is(deriveError, brick.ErrInvalidParameterDeclaration) {
				t.Fatalf("want ErrInvalidParameterDeclaration, got %v", deriveError)
			}

			if !strings.Contains(deriveError.Error(), declarationCase.wantInMessage) {
				t.Fatalf("want %q in the message, got %v", declarationCase.wantInMessage, deriveError)
			}
		})
	}
}

func TestSchemaValidateValue_EnforcesBoundsAndFormats(t *testing.T) {
	t.Parallel()

	schema := deriveSampleSchema(t)

	valueCases := []struct {
		fieldName  string
		value      any
		wantReason brick.RejectionReason
	}{
		{addressFieldName, sampleAddress + "path", ""},
		{addressFieldName, "http://example.test/", brick.ReasonBadFormat},
		{addressFieldName, "https://", brick.ReasonBadFormat},
		{addressFieldName, "", brick.ReasonOutOfBounds},
		{addressFieldName, sampleAddress + strings.Repeat("a", 200), brick.ReasonOutOfBounds},
		{countFieldName, int64(1), ""},
		{countFieldName, int64(0), brick.ReasonOutOfBounds},
		{countFieldName, int64(11), brick.ReasonOutOfBounds},
		{countFieldName, "three", brick.ReasonWrongType},
		{enabledFieldName, true, ""},
		{enabledFieldName, "yes", brick.ReasonWrongType},
		{intervalFieldName, time.Minute, ""},
		{intervalFieldName, 10 * time.Second, brick.ReasonOutOfBounds},
		{credentialFieldName, sampleCredential, ""},
		{credentialFieldName, "Has Spaces", brick.ReasonBadFormat},
		{backgroundFieldName, "#1a2B3c", ""},
		{backgroundFieldName, "red", brick.ReasonBadFormat},
		{labelFieldName, strings.Repeat("é", 24), ""},
		{labelFieldName, strings.Repeat("é", 25), brick.ReasonOutOfBounds},
	}

	for _, valueCase := range valueCases {
		field, fieldFound := schema.Field(valueCase.fieldName)
		if !fieldFound {
			t.Fatalf("no field %s", valueCase.fieldName)
		}

		validateError := schema.ValidateValue(field, valueCase.value)

		gotReason := brick.RejectionReason("")

		if fieldError, isFieldError := errors.AsType[*brick.FieldError](validateError); isFieldError {
			gotReason = fieldError.Reason
		} else if validateError != nil {
			t.Errorf("%s=%v: want a FieldError, got %v", valueCase.fieldName, valueCase.value, validateError)

			continue
		}

		if gotReason != valueCase.wantReason {
			t.Errorf("%s=%v: want reason %q, got %q (%v)",
				valueCase.fieldName, valueCase.value, valueCase.wantReason, gotReason, validateError)
		}
	}
}

func TestSchemaAssignAndRead_RoundTripEveryFieldType(t *testing.T) {
	t.Parallel()

	schema := deriveSampleSchema(t)

	values := map[string]any{
		labelFieldName:      docsLabel,
		addressFieldName:    sampleAddress,
		countFieldName:      int64(3),
		enabledFieldName:    true,
		intervalFieldName:   2 * time.Minute,
		credentialFieldName: sampleCredential,
	}

	parameters := schema.NewParameters()
	for fieldName, value := range values {
		field, _ := schema.Field(fieldName)

		assignError := schema.Assign(parameters, field, value)
		if assignError != nil {
			t.Fatalf("assigning %s: %v", fieldName, assignError)
		}
	}

	want := &sampleParameters{
		Label:      docsLabel,
		Icon:       "",
		Background: "",
		IconFile:   "",
		Address:    sampleAddress,
		Count:      3,
		Enabled:    true,
		Interval:   2 * time.Minute,
		Credential: sampleCredential,
	}
	if !reflect.DeepEqual(parameters, want) {
		t.Fatalf("want %+v, got %+v", want, parameters)
	}

	for fieldName, value := range values {
		field, _ := schema.Field(fieldName)
		if read := schema.Read(parameters, field); read != value {
			t.Errorf("reading %s: want %v, got %v", fieldName, value, read)
		}
	}
}

func TestAppearanceFieldNames_MatchTheDerivedAppearanceSchema(t *testing.T) {
	t.Parallel()

	appearanceSchema, deriveError := brick.DeriveSchema[brick.Appearance]()
	if deriveError != nil {
		t.Fatalf("deriving Appearance: %v", deriveError)
	}

	derivedNames := make([]string, 0, len(appearanceSchema.Fields()))
	for _, field := range appearanceSchema.Fields() {
		derivedNames = append(derivedNames, field.Name)
	}

	if got := brick.AppearanceFieldNames(); !reflect.DeepEqual(got, derivedNames) {
		t.Fatalf("want %v, got %v", derivedNames, got)
	}
}

// TestDeriveSchema_ReadsAFieldsSuggestions: a field may say where the editor
// finds values to offer for it. The applications list feeds the author's app
// picker (evaluation, deck testing).
func TestDeriveSchema_ReadsAFieldsSuggestions(t *testing.T) {
	t.Parallel()

	schema, deriveError := brick.DeriveSchemaOfType(reflect.TypeFor[suggestedApplication]())
	if deriveError != nil {
		t.Fatal(deriveError)
	}

	field, found := schema.Field("application")
	if !found || field.Suggestions != brick.SuggestApplications {
		t.Fatalf("want the applications list, got %+v", field)
	}
}

// TestDeriveSchema_ReadsWhatACheckBoxReplaces: a check box may stand in for
// other fields, as use_app_icon does for icon and icon_file, so the editor can
// set them aside while it is ticked (the author's request, evaluation, deck
// testing).
func TestDeriveSchema_ReadsWhatACheckBoxReplaces(t *testing.T) {
	t.Parallel()

	schema, deriveError := brick.DeriveSchemaOfType(reflect.TypeFor[replacingCheckBox]())
	if deriveError != nil {
		t.Fatal(deriveError)
	}

	field, _ := schema.Field("own")
	if !slices.Equal(field.Replaces, []string{"icon", "icon_file"}) {
		t.Fatalf("want icon and icon_file replaced, got %v", field.Replaces)
	}
}

// TestDeriveSchema_ReadsANumbersDefault: a new key's form starts at a number's
// default, as the author asked for a brightness key's level (evaluation, deck
// testing).
func TestDeriveSchema_ReadsANumbersDefault(t *testing.T) {
	t.Parallel()

	schema, deriveError := brick.DeriveSchemaOfType(reflect.TypeFor[defaultedCount]())
	if deriveError != nil {
		t.Fatal(deriveError)
	}

	field, found := schema.Field("count")
	if !found || field.Default != "5" {
		t.Fatalf("want the default 5, got %+v", field)
	}
}

// TestDeriveSchema_ReadsACheckBoxsPlatformLabels: a modifier key has a name
// on each platform, Command on a Mac and Win on Windows, so the editor can show
// either (the author's request, evaluation, deck testing).
func TestDeriveSchema_ReadsACheckBoxsPlatformLabels(t *testing.T) {
	t.Parallel()

	schema, deriveError := brick.DeriveSchemaOfType(reflect.TypeFor[labelledModifier]())
	if deriveError != nil {
		t.Fatal(deriveError)
	}

	field, found := schema.Field("command")
	if !found || field.MacLabel != "Command" || field.WindowsLabel != "Win" {
		t.Fatalf("want the Mac and Windows labels, got %+v", field)
	}
}
