package deck

import (
	"errors"
	"fmt"
	"strings"
)

// ErrMalformedReport reports bytes from the device that do not have the shape
// the model's protocol gives them. Bytes from the device are untrusted (datum
// 0.10), so a report of the wrong length is refused, never guessed at.
var ErrMalformedReport = errors.New("malformed report from the deck")

// The printable ASCII range a device text field is shown in.
const (
	firstPrintable = 0x20
	lastPrintable  = 0x7E
)

// unprintablePlaceholder stands in for a byte outside printable ASCII, as the
// reference's ascii decoding with errors replaced does.
const unprintablePlaceholder = '?'

// featureQuirkAllowance is one byte of slack on a feature report's length: the
// reference oversizes macOS feature reads by one byte for a hidapi 0.9.0 bug
// (evaluation 3.6), so a report one byte long is not malformed.
const featureQuirkAllowance = 1

// KeyChange is one key going down or coming up.
type KeyChange struct {
	KeyIndex int
	Pressed  bool
}

// DecodeKeyStates reads an input report: the model's header, which is not
// interpreted (on the XL it read 01 00 20 00 in experiment 1: report ID 1, then
// the key count, 32, little endian), then one byte per key,
// non zero meaning pressed as in the reference. Only a report of exactly the
// model's input length is accepted, and the parser never reads past it.
func DecodeKeyStates(model Model, report []byte) ([]bool, error) {
	protocol := model.Protocol
	if len(report) != protocol.InputReportLength {
		return nil, fmt.Errorf("%w: input report of %d bytes, expected %d",
			ErrMalformedReport, len(report), protocol.InputReportLength)
	}

	keyBytes := report[protocol.InputHeaderLength:]
	states := make([]bool, len(keyBytes))

	for keyIndex, keyByte := range keyBytes {
		states[keyIndex] = keyByte != 0
	}

	return states, nil
}

// KeyChanges lists the keys whose state differs between two reports, in key
// order. Reports of different lengths compare only their common keys.
func KeyChanges(previous []bool, current []bool) []KeyChange {
	var changes []KeyChange

	for keyIndex := range min(len(previous), len(current)) {
		if previous[keyIndex] != current[keyIndex] {
			changes = append(changes, KeyChange{KeyIndex: keyIndex, Pressed: current[keyIndex]})
		}
	}

	return changes
}

// DecodeSerialNumber reads the serial number from its feature report.
func DecodeSerialNumber(model Model, report []byte) (string, error) {
	return decodeFeatureText(model, report, model.Protocol.SerialTextOffset)
}

// DecodeFirmwareVersion reads the firmware version from its feature report.
func DecodeFirmwareVersion(model Model, report []byte) (string, error) {
	return decodeFeatureText(model, report, model.Protocol.FirmwareTextOffset)
}

// decodeFeatureText reads ASCII from an offset up to the first NUL, trimmed, as
// the reference's _extract_string does, showing unprintable bytes as ?.
func decodeFeatureText(model Model, report []byte, textOffset int) (string, error) {
	maximumLength := model.Protocol.FeatureReportLength + featureQuirkAllowance
	if len(report) <= textOffset || len(report) > maximumLength {
		return "", fmt.Errorf("%w: feature report of %d bytes, expected more than %d and at most %d",
			ErrMalformedReport, len(report), textOffset, maximumLength)
	}

	var text strings.Builder

	for _, textByte := range report[textOffset:] {
		if textByte == 0 {
			break
		}

		if textByte < firstPrintable || textByte > lastPrintable {
			text.WriteByte(unprintablePlaceholder)

			continue
		}

		text.WriteByte(textByte)
	}

	return strings.TrimSpace(text.String()), nil
}
