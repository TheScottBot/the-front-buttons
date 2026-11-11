package brick

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// The struct tags a parameter declaration uses. Each field carries its name and
// flags in the brick tag, and its bounds, format and description in their own
// tags so a description may contain commas.
const (
	brickTagKey       = "brick"
	boundsTagKey      = "bounds"
	formatTagKey      = "format"
	descriptionTagKey = "description"
)

// The flags the brick tag accepts after the name.
const (
	requiredOption = "required"
	secretOption   = "secret"
)

// boundsPartCount is the number of comma separated parts in a bounds tag: a
// minimum and a maximum, both required, so a bound is never half stated.
const boundsPartCount = 2

// Schema is the one description of a brick kind's parameters, derived from its
// parameter struct. Validation, decoding, editor forms and the generated
// reference all read it, so none of them keeps its own list of fields.
type Schema struct {
	parametersType reflect.Type
	fields         []Field
}

// DeriveSchema derives the schema of parameter struct type P.
func DeriveSchema[P any]() (Schema, error) {
	return DeriveSchemaOfType(reflect.TypeFor[P]())
}

// DeriveSchemaOfType derives the schema of a parameter struct type, refusing any
// declaration that breaks a tag rule. A rule broken here would otherwise surface
// as a confusing configuration error long after the code was written.
func DeriveSchemaOfType(parametersType reflect.Type) (Schema, error) {
	if parametersType.Kind() != reflect.Struct {
		return Schema{parametersType: nil, fields: nil}, fmt.Errorf(
			"%w: %s is not a struct", ErrInvalidParameterDeclaration, parametersType)
	}

	fields, fieldsError := deriveFields(parametersType, nil)
	if fieldsError != nil {
		return Schema{parametersType: nil, fields: nil}, fieldsError
	}

	seenNames := make(map[string]bool, len(fields))
	for _, field := range fields {
		if seenNames[field.Name] {
			return Schema{parametersType: nil, fields: nil}, fmt.Errorf(
				"%w: %s declares duplicate field name %q", ErrInvalidParameterDeclaration, parametersType, field.Name)
		}

		seenNames[field.Name] = true
	}

	return Schema{parametersType: parametersType, fields: fields}, nil
}

func deriveFields(structType reflect.Type, parentIndex []int) ([]Field, error) {
	var fields []Field

	for fieldPosition := range structType.NumField() {
		structField := structType.Field(fieldPosition)
		fieldIndex := append(append([]int(nil), parentIndex...), fieldPosition)

		if structField.Anonymous {
			embeddedFields, embeddedError := deriveEmbeddedFields(structType, structField, fieldIndex)
			if embeddedError != nil {
				return nil, embeddedError
			}

			fields = append(fields, embeddedFields...)

			continue
		}

		field, fieldError := deriveField(structType, structField, fieldIndex)
		if fieldError != nil {
			return nil, fieldError
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// deriveEmbeddedFields admits only Appearance as an embedded struct: it is the
// one set of shared parameters, and embedding anything else would let fields
// arrive in a schema without anyone declaring them for that kind.
func deriveEmbeddedFields(structType reflect.Type, structField reflect.StructField, fieldIndex []int) ([]Field, error) {
	if structField.Type != reflect.TypeFor[Appearance]() {
		return nil, fmt.Errorf("%w: %s embedded %s; only brick.Appearance may be embedded",
			ErrInvalidParameterDeclaration, structType, structField.Type)
	}

	return deriveFields(structField.Type, fieldIndex)
}

func deriveField(structType reflect.Type, structField reflect.StructField, fieldIndex []int) (Field, error) {
	locate := func(cause error) error {
		return fmt.Errorf("%s.%s: %w", structType, structField.Name, cause)
	}

	if !structField.IsExported() {
		return Field{}, locate(declarationFault("unexported field; every field must be a declared parameter"))
	}

	brickTag, hasBrickTag := structField.Tag.Lookup(brickTagKey)
	if !hasBrickTag {
		return Field{}, locate(declarationFault("missing brick tag"))
	}

	field, optionsError := parseBrickTag(brickTag, fieldIndex, structField.Tag.Get(descriptionTagKey))
	if optionsError != nil {
		return Field{}, locate(optionsError)
	}

	if field.Description == "" {
		return Field{}, locate(declarationFault("missing description; the editor and generated reference need one"))
	}

	fieldType, typeSupported := fieldTypeOf(structField.Type)
	if !typeSupported {
		return Field{}, locate(declarationFault("unsupported type %s", structField.Type))
	}

	field.Type = fieldType

	attributesError := applyBoundsFormatAndSecret(&field, structField.Tag)
	if attributesError != nil {
		return Field{}, locate(attributesError)
	}

	return field, nil
}

// declarationFault describes one broken tag rule, wrapping
// ErrInvalidParameterDeclaration so callers can recognise it.
func declarationFault(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidParameterDeclaration, fmt.Sprintf(format, arguments...))
}

func parseBrickTag(brickTag string, fieldIndex []int, description string) (Field, error) {
	tagParts := strings.Split(brickTag, ",")

	// Type, bounds and format are filled in by the caller once the Go type is
	// known; the zero values here are placeholders, never defaults that escape.
	field := Field{
		Name:             tagParts[0],
		Type:             "",
		Required:         false,
		Bounds:           nil,
		Format:           "",
		Description:      description,
		Secret:           false,
		structFieldIndex: fieldIndex,
	}
	if !lowerSnakeCaseName.MatchString(field.Name) {
		return Field{}, declarationFault("field name %q must be lower snake case", field.Name)
	}

	for _, option := range tagParts[1:] {
		switch option {
		case requiredOption:
			field.Required = true
		case secretOption:
			field.Secret = true
		default:
			return Field{}, declarationFault("unknown brick tag option %q", option)
		}
	}

	return field, nil
}

func fieldTypeOf(goType reflect.Type) (FieldType, bool) {
	if goType == reflect.TypeFor[time.Duration]() {
		return FieldTypeDuration, true
	}

	switch goType.Kind() { //nolint:exhaustive // every other kind is unsupported, which the default reports
	case reflect.String:
		return FieldTypeText, true
	case reflect.Int:
		return FieldTypeInteger, true
	case reflect.Bool:
		return FieldTypeBoolean, true
	default:
		return "", false
	}
}

func applyBoundsFormatAndSecret(field *Field, structTag reflect.StructTag) error {
	if boundsTag, hasBounds := structTag.Lookup(boundsTagKey); hasBounds {
		bounds, boundsError := parseBounds(field.Type, boundsTag)
		if boundsError != nil {
			return boundsError
		}

		field.Bounds = &bounds
	}

	if formatTag, hasFormat := structTag.Lookup(formatTagKey); hasFormat {
		if field.Type != FieldTypeText {
			return declarationFault("format applies only to text, not %s", field.Type)
		}

		if _, formatKnown := formatCheckers()[Format(formatTag)]; !formatKnown {
			return declarationFault("unknown format %q", formatTag)
		}

		field.Format = Format(formatTag)
	}

	if field.Secret && field.Type != FieldTypeText {
		return declarationFault("secret applies only to text, because it holds a credential name, not %s", field.Type)
	}

	return nil
}

func parseBounds(fieldType FieldType, boundsTag string) (Bounds, error) {
	if fieldType == FieldTypeBoolean {
		return Bounds{}, declarationFault("bounds do not apply to %s", fieldType)
	}

	boundsParts := strings.Split(boundsTag, ",")
	if len(boundsParts) != boundsPartCount {
		return Bounds{}, declarationFault("bounds %q must be written minimum,maximum", boundsTag)
	}

	parseBound := func(boundText string) (int64, error) {
		if fieldType == FieldTypeDuration {
			duration, durationError := time.ParseDuration(boundText)

			return int64(duration), durationError
		}

		return strconv.ParseInt(boundText, 10, 64)
	}

	minimum, minimumError := parseBound(boundsParts[0])
	maximum, maximumError := parseBound(boundsParts[1])

	switch {
	case minimumError != nil || maximumError != nil:
		return Bounds{}, declarationFault("bounds %q are not both %s values", boundsTag, fieldType)
	case minimum > maximum:
		return Bounds{}, declarationFault("bounds %q have the minimum above the maximum", boundsTag)
	case fieldType == FieldTypeText && minimum < 0:
		return Bounds{}, declarationFault("bounds %q give a negative length", boundsTag)
	}

	return Bounds{Minimum: minimum, Maximum: maximum}, nil
}

// Fields returns the schema's fields in declaration order, Appearance first.
func (schema Schema) Fields() []Field {
	return append([]Field(nil), schema.fields...)
}

// Field returns the field with the given name.
func (schema Schema) Field(fieldName string) (Field, bool) {
	for _, field := range schema.fields {
		if field.Name == fieldName {
			return field, true
		}
	}

	var absentField Field

	return absentField, false
}

// ValidateValue checks one value against its field: its Go type, then its
// bounds, then its format. The value's types are string for text, int64 for an
// integer, bool for a boolean and time.Duration for a duration.
func (schema Schema) ValidateValue(field Field, value any) error {
	refuse := func(reason RejectionReason, format string, arguments ...any) error {
		return &FieldError{Field: field.Name, Reason: reason, Detail: fmt.Sprintf(format, arguments...)}
	}

	measure, typeMatches := measureValue(field.Type, value)
	if !typeMatches {
		return refuse(ReasonWrongType, "expected %s", field.Type)
	}

	if field.Bounds != nil && (measure < field.Bounds.Minimum || measure > field.Bounds.Maximum) {
		return refuse(ReasonOutOfBounds, "must be between %s and %s",
			describeBound(field.Type, field.Bounds.Minimum), describeBound(field.Type, field.Bounds.Maximum))
	}

	if field.Format != "" {
		textValue, _ := value.(string)
		if !formatCheckers()[field.Format](textValue) {
			return refuse(ReasonBadFormat, "must be %s", field.Format)
		}
	}

	return nil
}

// measureValue returns the number bounds are checked against, and whether the
// value has the field's Go type.
func measureValue(fieldType FieldType, value any) (int64, bool) {
	switch fieldType {
	case FieldTypeText:
		textValue, isText := value.(string)

		return int64(utf8.RuneCountInString(textValue)), isText
	case FieldTypeInteger:
		integerValue, isInteger := value.(int64)

		return integerValue, isInteger
	case FieldTypeBoolean:
		_, isBoolean := value.(bool)

		return 0, isBoolean
	case FieldTypeDuration:
		durationValue, isDuration := value.(time.Duration)

		return int64(durationValue), isDuration
	default:
		return 0, false
	}
}

func describeBound(fieldType FieldType, bound int64) string {
	switch fieldType { //nolint:exhaustive // only durations and text read differently from a plain number
	case FieldTypeDuration:
		return time.Duration(bound).String()
	case FieldTypeText:
		return strconv.FormatInt(bound, 10) + " characters"
	default:
		return strconv.FormatInt(bound, 10)
	}
}

// NewParameters returns a pointer to a zero value of the kind's parameter
// struct, ready for Assign.
func (schema Schema) NewParameters() any {
	return reflect.New(schema.parametersType).Interface()
}

// Assign validates a value and stores it in the parameters, which must be a
// pointer returned by NewParameters.
func (schema Schema) Assign(parameters any, field Field, value any) error {
	validateError := schema.ValidateValue(field, value)
	if validateError != nil {
		return validateError
	}

	target, targetError := schema.fieldTarget(parameters, field)
	if targetError != nil {
		return targetError
	}

	if field.Type == FieldTypeInteger {
		integerValue, _ := value.(int64)
		if target.OverflowInt(integerValue) {
			return &FieldError{Field: field.Name, Reason: ReasonOutOfBounds, Detail: "too large for this platform"}
		}

		target.SetInt(integerValue)

		return nil
	}

	target.Set(reflect.ValueOf(value))

	return nil
}

// Read returns a field's value from the parameters, in the same Go type
// ValidateValue accepts.
func (schema Schema) Read(parameters any, field Field) any {
	target, targetError := schema.fieldTarget(parameters, field)
	if targetError != nil {
		return nil
	}

	if field.Type == FieldTypeInteger {
		return target.Int()
	}

	return target.Interface()
}

// ParametersType is the Go type of the kind's parameter struct.
func (schema Schema) ParametersType() reflect.Type {
	return schema.parametersType
}

func (schema Schema) fieldTarget(parameters any, field Field) (reflect.Value, error) {
	pointer := reflect.ValueOf(parameters)
	if pointer.Kind() != reflect.Pointer || pointer.Elem().Type() != schema.parametersType {
		return reflect.Value{}, fmt.Errorf("%w: parameters are %T, not *%s",
			ErrParametersMismatch, parameters, schema.parametersType)
	}

	return pointer.Elem().FieldByIndex(field.structFieldIndex), nil
}
