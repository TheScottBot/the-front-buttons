package systemadapter

import (
	"errors"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// Bridge names the way the macOS adapters call the system: through cgo, or
// through purego with no C compiler. Windows takes it and ignores it.
type Bridge string

// The bridges.
const (
	BridgeCgo    Bridge = "cgo"
	BridgePurego Bridge = "purego"
)

// ErrBridgeUnavailable reports a purego adapter not written yet, or cgo
// asked for in a build without it.
var ErrBridgeUnavailable = errors.New("not available through this bridge")

// AskingKeyPresser is what NewKeyPresser returns: a KeyPresser and
// ShortcutPresser that can ask for the Accessibility permission it needs, so
// the window keys share its prompt and its restart.
type AskingKeyPresser interface {
	platform.KeyPresser
	platform.ShortcutPresser
	RequestAccessibility()
}
