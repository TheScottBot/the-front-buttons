package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.yaml.in/yaml/v4"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// ErrNotEncodable reports a configuration naming a kind or preset this build does
// not have, which only a programming error can produce.
var ErrNotEncodable = errors.New("configuration cannot be encoded")

// encodingIndent is the indentation of written files, matching what a person
// editing by hand most often writes.
const encodingIndent = 2

// Encode writes a configuration in the file format Decode reads, driven by the
// same schemas and names, so the encoder and the decoder share one grammar
// (datum 0.7). Optional fields at their zero value are left out, because
// absence already means the zero value. A preset key is written as its preset
// name and fields, never its expansion.
//
// Encode does not validate; a caller that built the configuration itself
// decodes the result before saving it.
func (decoder Decoder) Encode(configuration Configuration) ([]byte, error) {
	pagesNode := sequenceNode()

	for _, page := range configuration.Pages {
		keysNode := sequenceNode()

		for _, key := range page.Keys {
			keyNode, keyError := decoder.encodeKey(key)
			if keyError != nil {
				return nil, keyError
			}

			keysNode.Content = append(keysNode.Content, keyNode)
		}

		pageNode := mappingNode(textNode(pageNameName), textNode(page.Name))
		if page.IsAppScreen() {
			pageNode.Content = append(pageNode.Content, textNode(applicationName), textNode(page.Application))
		}

		pageNode.Content = append(pageNode.Content, textNode(keysName), keysNode)
		pagesNode.Content = append(pagesNode.Content, pageNode)
	}

	root := mappingNode(
		textNode(schemaVersionName), integerNode(int64(configuration.SchemaVersion)),
		textNode(pagesName), pagesNode,
	)

	encoded, dumpError := yaml.Dump(root, yaml.WithIndent(encodingIndent))
	if dumpError != nil {
		return nil, fmt.Errorf("encoding configuration: %w", dumpError)
	}

	return encoded, nil
}

func (decoder Decoder) encodeKey(key Key) (*yaml.Node, error) {
	keyNode := mappingNode(
		textNode(rowName), integerNode(int64(key.Position.Row)),
		textNode(columnName), integerNode(int64(key.Position.Column)),
	)

	if key.Preset != nil {
		row, rowFound := decoder.presetTable.Row(key.Preset.Name)
		if !rowFound {
			return nil, fmt.Errorf("%w: unknown preset %q", ErrNotEncodable, key.Preset.Name)
		}

		keyNode.Content = append(keyNode.Content,
			textNode(presetName), textNode(key.Preset.Name),
			textNode(fieldsName), encodeFields(row.FieldsSchema(), key.Preset.Fields))

		return keyNode, nil
	}

	kind, kindFound := decoder.registry.Kind(key.Kind)
	if !kindFound {
		return nil, fmt.Errorf("%w: unknown brick kind %q", ErrNotEncodable, key.Kind)
	}

	keyNode.Content = append(keyNode.Content,
		textNode(kindName), textNode(key.Kind),
		textNode(parametersName), encodeFields(kind.Schema(), key.Parameters))

	return keyNode, nil
}

func encodeFields(schema brick.Schema, parameters any) *yaml.Node {
	fieldsNode := mappingNode()

	for _, field := range schema.Fields() {
		value := schema.Read(parameters, field)

		valueNode, isZero := fieldNode(value)
		if isZero && !field.Required {
			continue
		}

		fieldsNode.Content = append(fieldsNode.Content, textNode(field.Name), valueNode)
	}

	return fieldsNode
}

// fieldNode returns the node for a field value and whether it is the zero value.
func fieldNode(value any) (*yaml.Node, bool) {
	switch typedValue := value.(type) {
	case string:
		return textNode(typedValue), typedValue == ""
	case int64:
		return integerNode(typedValue), typedValue == 0
	case bool:
		return scalarNode(booleanTag, strconv.FormatBool(typedValue)), !typedValue
	case time.Duration:
		return textNode(typedValue.String()), typedValue == 0
	default:
		return scalarNode(textTag, ""), true
	}
}

// The node constructors set only what the file's meaning needs; style, comments
// and positions are left to the emitter.

func scalarNode(tag string, value string) *yaml.Node {
	node := new(yaml.Node)
	node.Kind = yaml.ScalarNode
	node.Tag = tag
	node.Value = value

	return node
}

func textNode(value string) *yaml.Node { return scalarNode(textTag, value) }

func integerNode(value int64) *yaml.Node {
	return scalarNode(integerTag, strconv.FormatInt(value, 10))
}

func mappingNode(content ...*yaml.Node) *yaml.Node {
	node := new(yaml.Node)
	node.Kind = yaml.MappingNode
	node.Content = content

	return node
}

func sequenceNode() *yaml.Node {
	node := new(yaml.Node)
	node.Kind = yaml.SequenceNode

	return node
}
