package config

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"time"

	"go.yaml.in/yaml/v4"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// The YAML tags a scalar must carry for each kind of value. Values are read
// only by tag, never by YAML's own conversion, so a value YAML would guess is a
// boolean or a number is refused where text is expected rather than coerced.
const (
	textTag    = "!!str"
	integerTag = "!!int"
	booleanTag = "!!bool"
)

// mappingEntryWidth is the number of nodes each mapping entry occupies in a
// node's content: the name, then the value.
const mappingEntryWidth = 2

// decimalInteger is the only spelling of an integer accepted. YAML also reads
// 0x10 and 0o17 as integers, which hand written configuration never means.
var decimalInteger = regexp.MustCompile(`^-?[0-9]+$`)

// problemList collects every problem in a file, so one save in the editor shows
// all of them at once.
type problemList struct {
	problems []Problem
}

func (list *problemList) addf(node *yaml.Node, path string, reason Reason, format string, arguments ...any) {
	line := 0
	if node != nil {
		line = node.Line
	}

	list.problems = append(list.problems, Problem{
		Path:   path,
		Line:   line,
		Reason: reason,
		Detail: fmt.Sprintf(format, arguments...),
	})
}

// mappingEntries returns a mapping's values by key, reporting any key not in
// allowedNames as unknown (datum 0.11). It returns false if the node is not a
// mapping.
func (list *problemList) mappingEntries(
	node *yaml.Node,
	path string,
	allowedNames ...string,
) (map[string]*yaml.Node, bool) {
	if node.Kind != yaml.MappingNode {
		list.addf(node, path, ReasonWrongType, "expected a mapping of names to values")

		return nil, false
	}

	allowed := make(map[string]bool, len(allowedNames))
	for _, allowedName := range allowedNames {
		allowed[allowedName] = true
	}

	entries := make(map[string]*yaml.Node, len(node.Content)/mappingEntryWidth)

	for entryIndex := 0; entryIndex+1 < len(node.Content); entryIndex += mappingEntryWidth {
		nameNode, valueNode := node.Content[entryIndex], node.Content[entryIndex+1]

		entryName, isText := textOf(nameNode)
		if !isText || !allowed[entryName] {
			list.addf(nameNode, path, ReasonUnknownField, "unknown field %q", nameNode.Value)

			continue
		}

		entries[entryName] = valueNode
	}

	return entries, true
}

// requireInteger reads a required integer entry, reporting it missing or of the
// wrong type.
func (list *problemList) requireInteger(
	entries map[string]*yaml.Node,
	parent *yaml.Node,
	path string,
	entryName string,
) (int64, bool) {
	valueNode, present := entries[entryName]
	if !present {
		list.addf(parent, path, ReasonMissing, "missing required field %q", entryName)

		return 0, false
	}

	integerValue, isInteger := integerOf(valueNode)
	if !isInteger {
		list.addf(valueNode, path+"."+entryName, ReasonWrongType, "expected a whole number written in decimal")

		return 0, false
	}

	return integerValue, true
}

func textOf(node *yaml.Node) (string, bool) {
	if node.Kind != yaml.ScalarNode || node.ShortTag() != textTag {
		return "", false
	}

	return node.Value, true
}

func integerOf(node *yaml.Node) (int64, bool) {
	if node.Kind != yaml.ScalarNode || node.ShortTag() != integerTag || !decimalInteger.MatchString(node.Value) {
		return 0, false
	}

	integerValue, parseError := strconv.ParseInt(node.Value, 10, 64)

	return integerValue, parseError == nil
}

func booleanOf(node *yaml.Node) (bool, bool) {
	if node.Kind != yaml.ScalarNode || node.ShortTag() != booleanTag {
		return false, false
	}

	switch node.Value {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

// fieldValueOf converts a node to the Go value a brick field holds, or reports
// what was expected.
func fieldValueOf(field brick.Field, node *yaml.Node) (any, string) {
	convert, supported := fieldConverters()[field.Type]
	if !supported {
		return nil, "unsupported field type"
	}

	return convert(node)
}

// fieldConverters holds one conversion per field type, each returning the value
// or a description of what was expected.
func fieldConverters() map[brick.FieldType]func(node *yaml.Node) (any, string) {
	return map[brick.FieldType]func(node *yaml.Node) (any, string){
		brick.FieldTypeText: func(node *yaml.Node) (any, string) {
			if textValue, isText := textOf(node); isText {
				return textValue, ""
			}

			return nil, "expected text; quote values YAML would read as numbers or booleans"
		},
		brick.FieldTypeInteger: func(node *yaml.Node) (any, string) {
			if integerValue, isInteger := integerOf(node); isInteger {
				return integerValue, ""
			}

			return nil, "expected a whole number written in decimal"
		},
		brick.FieldTypeBoolean: func(node *yaml.Node) (any, string) {
			if booleanValue, isBoolean := booleanOf(node); isBoolean {
				return booleanValue, ""
			}

			return nil, "expected true or false"
		},
		brick.FieldTypeDuration: durationOf,
	}
}

func durationOf(node *yaml.Node) (any, string) {
	const expectedDuration = "expected a duration such as 90s or 5m"

	textValue, isText := textOf(node)
	if !isText {
		return nil, expectedDuration
	}

	duration, parseError := time.ParseDuration(textValue)
	if parseError != nil {
		return nil, expectedDuration
	}

	return duration, ""
}

// containsAlias reports whether any node in the tree is an alias. Aliases are
// refused outright: hand written configuration has no need of them, and refusing
// them removes alias expansion from the attack surface entirely.
func containsAlias(node *yaml.Node) bool {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return true
	}

	return slices.ContainsFunc(node.Content, containsAlias)
}

// firstDuplicateKey returns the first mapping key that appears twice in the same
// mapping anywhere in the tree, or nil. go-yaml v4's unique key option does not
// apply when decoding into a node tree (finding TF11), so the decoder checks
// itself: a duplicated key would otherwise let the last value silently win.
func firstDuplicateKey(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.MappingNode {
		seenKeys := make(map[string]bool, len(node.Content)/mappingEntryWidth)

		for entryIndex := 0; entryIndex+1 < len(node.Content); entryIndex += mappingEntryWidth {
			keyNode := node.Content[entryIndex]
			if seenKeys[keyNode.Value] {
				return keyNode
			}

			seenKeys[keyNode.Value] = true
		}
	}

	for _, child := range node.Content {
		if duplicate := firstDuplicateKey(child); duplicate != nil {
			return duplicate
		}
	}

	return nil
}
