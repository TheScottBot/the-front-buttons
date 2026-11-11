package deck

import (
	"errors"
	"time"
)

// ErrDisconnected reports a deck that can no longer be used: unplugged, its dock
// switched to the other machine, or closed. The dock moving between machines is
// normal operation (specification), so callers treat this as a state, not a
// fault, and wait for the watcher to see the deck again.
var ErrDisconnected = errors.New("deck disconnected")

// ErrDeckBusy reports a deck that is attached but cannot be opened, most often
// because other software, such as Elgato's own, holds it.
var ErrDeckBusy = errors.New("deck busy")

// Connection identifies one attached deck. Path is the operating system's name
// for it, which changes when it is unplugged and plugged back in.
type Connection struct {
	Path      string
	VendorID  uint16
	ProductID uint16
}

// DeckDevice is the port that moves bytes to and from one open deck. Its adapter
// is hidtransport; it understands no protocol, which lives in this package's pure
// encoders and decoders (specification section 2). Every error after a deck goes
// away wraps ErrDisconnected.
//
//nolint:revive // the name is the specification's, from its ports table
type DeckDevice interface {
	// Write sends one output report.
	Write(report []byte) error
	// SendFeatureReport sends one feature report.
	SendFeatureReport(report []byte) error
	// GetFeatureReport reads the feature report with the given ID, at most
	// length bytes plus the one byte the macOS quirk allows (evaluation 3.6).
	GetFeatureReport(reportID byte, length int) ([]byte, error)
	// ReadInputReport waits up to timeout for one input report and returns its
	// length, or zero if none arrived.
	ReadInputReport(buffer []byte, timeout time.Duration) (int, error)
	// Close releases the deck. Closing twice is safe.
	Close() error
}

// DeckWatcher is the port that finds attached decks and opens them. Its adapter
// is hidtransport's enumeration; neither hidapi release offers hotplug callbacks
// (evaluation section 4), so the core polls Enumerate.
//
//nolint:revive // the name is the specification's, from its ports table
type DeckWatcher interface {
	// Enumerate lists attached decks of every model in the model table.
	Enumerate() ([]Connection, error)
	// Open opens one listed deck.
	Open(connection Connection) (DeckDevice, error)
}
