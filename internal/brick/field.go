package brick

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
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
	// FormatLinkURL is what a key may open: an https URL, or a link in Discord's
	// own discord:// scheme on discord.com, which opens the app where an https
	// link opens the browser (the author's decision amending TD13). No other
	// scheme is accepted.
	FormatLinkURL Format = "link_url"
	// FormatIdentifier is a lower case name such as a credential or icon name.
	FormatIdentifier Format = "identifier"
	// FormatDigits is one or more decimal digits, as in Discord's IDs.
	FormatDigits Format = "digits"
	// FormatColour is a colour written #RRGGBB.
	FormatColour Format = "colour"
	// FormatRepository is a GitHub repository written owner/name.
	FormatRepository Format = "repository"
	// FormatToken is one path segment such as a workflow file name or ID: no
	// slashes, and not starting with a dot.
	FormatToken Format = "token"
	// FormatBranch is a branch name, which may hold slashes but never "..".
	FormatBranch Format = "branch"
	// FormatPNGFile is a PNG's plain file name: letters, digits, underscores and
	// hyphens in dot separated parts, ending .png, with no slash, no leading dot
	// and no empty part, so it can only name a file in the folder it is read
	// from.
	FormatPNGFile Format = "png_file"
	// FormatFunctionKey is an extended function key, F13 to F24, which a
	// press_key key sends; no keyboard has them, so a press never collides with
	// typing.
	FormatFunctionKey Format = "function_key"
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
	// Suggestions names a list the editor offers values from, such as the
	// installed applications; empty for none.
	Suggestions Suggestions
	// Replaces names the fields a check box stands in for while it is ticked,
	// as use_app_icon does for icon and icon_file.
	Replaces []string

	structFieldIndex []int
}

// Suggestions names a list of values the editor can offer for a field. The
// list itself comes from outside the schema, from a platform adapter.
type Suggestions string

// SuggestApplications is the installed applications, for the app picker the
// author asked for (evaluation, deck testing).
const SuggestApplications Suggestions = "applications"

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
	// GitHub's owner and repository name limits: an owner of up to 39
	// characters of letters, digits and hyphens; a name of up to 100.
	repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}/[A-Za-z0-9_][A-Za-z0-9._-]{0,99}$`)
	tokenPattern      = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)
	branchPattern     = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._/-]*$`)
	functionKeyRule   = regexp.MustCompile(`^F(1[3-9]|2[0-4])$`)
	pngFilePattern    = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_-]*(\.[A-Za-z0-9_-]+)*\.[pP][nN][gG]$`)
)

// formatCheckers holds the one check per format.
func formatCheckers() map[Format]func(textValue string) bool {
	return map[Format]func(textValue string) bool{
		FormatHTTPSURL:    isHTTPSURL,
		FormatLinkURL:     isLinkURL,
		FormatIdentifier:  identifierPattern.MatchString,
		FormatDigits:      digitsPattern.MatchString,
		FormatColour:      colourPattern.MatchString,
		FormatRepository:  repositoryPattern.MatchString,
		FormatToken:       tokenPattern.MatchString,
		FormatBranch:      isBranchName,
		FormatPNGFile:     pngFilePattern.MatchString,
		FormatFunctionKey: functionKeyRule.MatchString,
	}
}

// The extended function keys a function_key value names.
const (
	firstFunctionKey = 13
	lastFunctionKey  = 24
)

// FormatChoices lists every value of a format whose values are few and fixed,
// so the editor can offer a choice instead of a text field where a typing slip
// breaks the key (the author's request, evaluation, deck testing). It returns
// nil for an open ended format.
func FormatChoices(format Format) []string {
	if format != FormatFunctionKey {
		return nil
	}

	choices := make([]string, 0, lastFunctionKey-firstFunctionKey+1)
	for number := firstFunctionKey; number <= lastFunctionKey; number++ {
		choices = append(choices, fmt.Sprintf("F%d", number))
	}

	return choices
}

// isBranchName refuses "..", which the pattern alone cannot, because it would
// let a branch name climb out of its path.
func isBranchName(textValue string) bool {
	return branchPattern.MatchString(textValue) && !strings.Contains(textValue, "..")
}

// CheckFormat reports whether a value satisfies a format. It is the one check
// per format, used by the schema and by any adapter guarding a value it was
// given, so the two can never disagree (datum 0.7). An unknown format refuses
// every value.
func CheckFormat(format Format, textValue string) bool {
	checker, known := formatCheckers()[format]

	return known && checker(textValue)
}

func isHTTPSURL(textValue string) bool {
	parsedURL, parseError := url.Parse(textValue)

	return parseError == nil && parsedURL.Scheme == "https" && parsedURL.Host != "" && parsedURL.User == nil
}

// discordAppScheme and discordAppHost are the one app link a key may open
// besides https, seen working on the author's MacBook (evaluation, deck testing).
const (
	discordAppScheme = "discord"
	discordAppHost   = "discord.com"
)

func isLinkURL(textValue string) bool {
	if isHTTPSURL(textValue) {
		return true
	}

	parsedURL, parseError := url.Parse(textValue)

	return parseError == nil && parsedURL.Scheme == discordAppScheme && parsedURL.Host == discordAppHost &&
		parsedURL.User == nil
}
