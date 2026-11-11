package deck_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// keyReport builds an XL input report as StreamDeckXL._read_control_states reads
// it: 4 header bytes, then one byte per key. The header's content is unknown
// until experiment 1, so the decoder must not depend on it.
func keyReport(pressedKeys ...int) []byte {
	report := make([]byte, 36)
	report[0] = 0x01

	for _, keyIndex := range pressedKeys {
		report[4+keyIndex] = 1
	}

	return report
}

func TestDecodeKeyStates_ReadsOneByteperKeyAfterTheHeader(t *testing.T) {
	t.Parallel()

	states, decodeError := deck.DecodeKeyStates(xlModel(t), keyReport(0, 24, 31))
	if decodeError != nil {
		t.Fatalf("decoding: %v", decodeError)
	}

	for keyIndex, pressed := range states {
		wantPressed := keyIndex == 0 || keyIndex == 24 || keyIndex == 31
		if pressed != wantPressed {
			t.Errorf("key %d: want pressed %v, got %v", keyIndex, wantPressed, pressed)
		}
	}
}

func TestDecodeKeyStates_DecodesEveryKeyPressedAndReleased(t *testing.T) {
	t.Parallel()

	for keyIndex := range 32 {
		pressed, pressedError := deck.DecodeKeyStates(xlModel(t), keyReport(keyIndex))
		released, releasedError := deck.DecodeKeyStates(xlModel(t), keyReport())

		if pressedError != nil || releasedError != nil || !pressed[keyIndex] || released[keyIndex] {
			t.Fatalf("key %d: pressed %v released %v (%v %v)", keyIndex, pressed[keyIndex], released[keyIndex],
				pressedError, releasedError)
		}
	}
}

func TestDecodeKeyStates_TreatsAnyNonZeroByteAsPressed_AsTheReferenceDoes(t *testing.T) {
	t.Parallel()

	report := keyReport()
	report[4+3] = 0x7F

	states, decodeError := deck.DecodeKeyStates(xlModel(t), report)
	if decodeError != nil || !states[3] {
		t.Fatalf("want key 3 pressed, got %v (%v)", states, decodeError)
	}
}

func TestDecodeKeyStates_RefusesShortAndOversizedReports(t *testing.T) {
	t.Parallel()

	for _, length := range []int{0, 4, 35, 37, 1024} {
		_, decodeError := deck.DecodeKeyStates(xlModel(t), make([]byte, length))
		if !errors.Is(decodeError, deck.ErrMalformedReport) {
			t.Errorf("length %d: want ErrMalformedReport, got %v", length, decodeError)
		}
	}
}

func TestKeyChanges_ListsOnlyKeysWhoseStateChanged(t *testing.T) {
	t.Parallel()

	previous := make([]bool, 32)
	current := make([]bool, 32)
	previous[2], current[2] = true, true
	current[5] = true
	previous[9] = true

	changes := deck.KeyChanges(previous, current)

	want := []deck.KeyChange{{KeyIndex: 5, Pressed: true}, {KeyIndex: 9, Pressed: false}}
	if len(changes) != len(want) || changes[0] != want[0] || changes[1] != want[1] {
		t.Fatalf("want %v, got %v", want, changes)
	}
}

func TestDecodeFeatureText_ReadsTheSerialAndFirmwareAsTheReferenceDoes(t *testing.T) {
	t.Parallel()

	model := xlModel(t)

	serialReport := make([]byte, 32)
	copy(serialReport[2:], "CL12K1A00042")

	serial, serialError := deck.DecodeSerialNumber(model, serialReport)
	if serialError != nil || serial != "CL12K1A00042" {
		t.Fatalf("want the serial read from offset 2, got %q (%v)", serial, serialError)
	}

	firmwareReport := make([]byte, 32)
	copy(firmwareReport[6:], "1.01.000\x00junk")

	firmware, firmwareError := deck.DecodeFirmwareVersion(model, firmwareReport)
	if firmwareError != nil || firmware != "1.01.000" {
		t.Fatalf("want the firmware read from offset 6 up to the NUL, got %q (%v)", firmware, firmwareError)
	}
}

func TestDecodeFeatureText_ReplacesUnprintableBytes_AndRefusesShortReports(t *testing.T) {
	t.Parallel()

	model := xlModel(t)

	report := make([]byte, 32)
	copy(report[2:], "AB\x07C\xffD")

	serial, _ := deck.DecodeSerialNumber(model, report)
	if serial != "AB?C?D" {
		t.Fatalf("want unprintable bytes shown as ?, got %q", serial)
	}

	_, shortError := deck.DecodeSerialNumber(model, make([]byte, 2))
	if !errors.Is(shortError, deck.ErrMalformedReport) {
		t.Fatalf("want ErrMalformedReport for a short report, got %v", shortError)
	}
}
