// Package deck holds the deck model table, the pure encoders and decoders for the
// device protocol, and the DeckDevice and DeckWatcher ports.
//
// It is pure on purpose: every byte the protocol produces or accepts can be tested
// on a development machine with no device attached, and only hidtransport moves
// those bytes to real hardware.
package deck
