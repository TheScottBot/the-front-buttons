package brick

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

// FieldType is the kind of value a parameter holds. The set is deliberately
// small: every type here has one unambiguous spelling in the configuration file
// and one editor control.
type FieldType string

// The supported field types.
const (
	FieldTypeText     FieldType = "text"
	FieldTypeInteger  FieldType = "integer"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeDuration FieldType = "duration"
)

// Format names a rule a text value must satisfy beyond its length.
type Format string

// The supported formats. Each is checked by exactly one function in
// formatCheckers, so the rule lives in one place for configuration, editor and
// documentation alike.
const (
	// FormatHTTPSURL is an absolute https URL with a host.
	FormatHTTPSURL Format = "https_url"
	// FormatIdentifier is a lower case name such as a credential or icon name.
	FormatIdentifier Format = "identifier"
	// FormatDigits is one or more decimal digits, as in Discord's IDs.
	FormatDigits Format = "digits"
	// FormatColour is a colour written #RRGGBB.
	FormatColour Format = "colour"
)

// Bounds limits a value: the length in characters for text, the value for an
// integer, and the length of time in nanoseconds for a duration.
type Bounds struct {
	Minimum int64
	Maximum int64
}

// Field is one parameter as declared by a brick kind's parameter struct.
type Field struct {
	Name        string
	Type        FieldType
	Required    bool
	Bounds      *Bounds
	Format      Format
	Description string
	// Secret marks a field whose value is the name of a credential in the
	// credential store. The secret itself never enters configuration
	// (specification section 2); the editor offers write only entry for it.
	Secret bool

	structFieldIndex []int
}

// RejectionReason categorises why a value was refused, so rejections can be
// counted by reason rather than logged with their content (datum 0.15).
type RejectionReason string

// The reasons a value is refused.
const (
	ReasonMissing      RejectionReason = "missing"
	ReasonUnknownField RejectionReason = "unknown_field"
	ReasonWrongType    RejectionReason = "wrong_type"
	ReasonOutOfBounds  RejectionReason = "out_of_bounds"
	ReasonBadFormat    RejectionReason = "bad_format"
)

// FieldError reports a value refused for one field. It names the field and the
// reason but never echoes the value, which may be a URL holding a colleague's
// address (specification section 7).
type FieldError struct {
	Field  string
	Reason RejectionReason
	Detail string
}

func (fieldError *FieldError) Error() string {
	return fmt.Sprintf("field %s: %s: %s", fieldError.Field, fieldError.Reason, fieldError.Detail)
}

// ErrInvalidParameterDeclaration reports a parameter struct that breaks a tag
// rule. It is a programming error found at registration, before any
// configuration is read.
var ErrInvalidParameterDeclaration = errors.New("invalid brick parameter declaration")

// ErrInvalidKind reports a kind name that is malformed or already registered.
var ErrInvalidKind = errors.New("invalid brick kind")

// ErrParametersMismatch reports parameters of a different kind's type, a
// programming error between the configuration loader and the registry.
var ErrParametersMismatch = errors.New("parameters do not belong to this schema")

// lowerSnakeCaseName is the spelling every kind and field name must use, so a
// configuration file reads the same everywhere and no two names differ only by
// case.
var lowerSnakeCaseName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var (
	identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`)
	digitsPattern     = regexp.MustCompile(`^[0-9]+$`)
	colourPattern     = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// formatCheckers holds the one check per format.
func formatCheckers() map[Format]func(textValue string) bool {
	return map[Format]func(textValue string) bool{
		FormatHTTPSURL:   isHTTPSURL,
		FormatIdentifier: identifierPattern.MatchString,
		FormatDigits:     digitsPattern.MatchString,
		FormatColour:     colourPattern.MatchString,
	}
}

func isHTTPSURL(textValue string) bool {
	parsedURL, parseError := url.Parse(textValue)

	return parseError == nil && parsedURL.Scheme == "https" && parsedURL.Host != "" && parsedURL.User == nil
}
