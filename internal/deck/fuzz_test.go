package deck_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// FuzzDecodeKeyStates feeds arbitrary bytes to the input report decoder, the
// device facing parser (datum 0.10). It must never panic, and must accept only
// reports of exactly the model's input length.
func FuzzDecodeKeyStates(f *testing.F) {
	for _, seed := range [][]byte{nil, keyReport(), keyReport(0, 31), make([]byte, 35), make([]byte, 37)} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, report []byte) {
		model := xlModel(t)

		states, decodeError := deck.DecodeKeyStates(model, report)
		if decodeError != nil {
			return
		}

		if len(report) != model.Protocol.InputReportLength || len(states) != model.Rows*model.Columns {
			t.Fatalf("accepted a %d byte report as %d keys", len(report), len(states))
		}
	})
}

// FuzzDecodeSerialNumber feeds arbitrary feature reports to the text decoder.
func FuzzDecodeSerialNumber(f *testing.F) {
	for _, seed := range [][]byte{nil, make([]byte, 32), []byte("\x06\x00CL12\x00")} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, report []byte) {
		serial, decodeError := deck.DecodeSerialNumber(xlModel(t), report)
		if decodeError != nil {
			return
		}

		if len(serial) > len(report) {
			t.Fatalf("decoded %d characters from %d bytes", len(serial), len(report))
		}
	})
}
