// Package appkeys makes an application's own keys from a table of its
// actions: one kind per action, scoped to the application's app set, pressing
// the action's shortcut for the platform the daemon runs on, as the
// application's default keymap has it (evaluation, app screens). The
// application packages, such as jetbrains and gitkraken, hold the tables.
package appkeys

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// ErrBadTable reports an action table a set cannot be made from.
var ErrBadTable = errors.New("action table refused")

// Action is one application action: its name in kind names, its label on a
// key, and its shortcut on each platform.
type Action struct {
	Name    string
	Label   string
	Mac     platform.Shortcut
	Windows platform.Shortcut
}

// The modifiers, as shortcuts write them. On Windows, Alt takes Option's
// place and Ctrl usually Command's.
const (
	Command = iota
	Control
	Option
	Shift
)

// Keys builds a shortcut from a key and its modifiers.
func Keys(key string, held ...int) platform.Shortcut {
	shortcut := platform.Shortcut{Key: key, Control: false, Alt: false, Shift: false, Command: false}

	for _, modifier := range held {
		switch modifier {
		case Command:
			shortcut.Command = true
		case Control:
			shortcut.Control = true
		case Option:
			shortcut.Alt = true
		case Shift:
			shortcut.Shift = true
		}
	}

	return shortcut
}

// Both builds an action whose Windows shortcut is the macOS one with Ctrl in
// Command's place, as most applications' are.
func Both(name string, label string, key string, held ...int) Action {
	windowsHeld := make([]int, 0, len(held))

	for _, modifier := range held {
		if modifier == Command {
			modifier = Control
		}

		windowsHeld = append(windowsHeld, modifier)
	}

	return Action{Name: name, Label: label, Mac: Keys(key, held...), Windows: Keys(key, windowsHeld...)}
}

// Apart builds an action whose shortcuts differ by platform.
func Apart(name string, label string, mac platform.Shortcut, windows platform.Shortcut) Action {
	return Action{Name: name, Label: label, Mac: mac, Windows: windows}
}

// Check refuses a table with a repeated or empty name, an action without a
// label, or a shortcut whose key a shortcut may not name on either platform.
func Check(actions []Action) error {
	var problems []error

	names := make([]string, 0, len(actions))

	for _, row := range actions {
		switch {
		case row.Name == "" || slices.Contains(names, row.Name):
			problems = append(problems, fmt.Errorf("%w: %q is empty or repeated", ErrBadTable, row.Name))
		case row.Label == "":
			problems = append(problems, fmt.Errorf("%w: %s has no label", ErrBadTable, row.Name))
		case !brick.CheckFormat(brick.FormatShortcutKey, row.Mac.Key) ||
			!brick.CheckFormat(brick.FormatShortcutKey, row.Windows.Key):
			problems = append(problems, fmt.Errorf("%w: %s presses a key a shortcut may not name", ErrBadTable, row.Name))
		}

		names = append(names, row.Name)
	}

	return errors.Join(problems...)
}

// Parameters is every action kind's one declaration of its parameters: only
// its appearance.
type Parameters struct {
	brick.Appearance
}

// Register checks a table and registers each action as a kind of an app set,
// scoped to it and named with its prefix, such as jetbrains_build.
func Register(registry *brick.Registry, scope string, prefix string, actions []Action) error {
	checkError := Check(actions)
	if checkError != nil {
		return checkError
	}

	for _, row := range actions {
		kindName := prefix + row.Name
		pair := platform.ShortcutPair{Mac: row.Mac, Windows: row.Windows}
		label := row.Label

		registerError := brick.RegisterScoped(registry, scope, kindName, func(_ Parameters, ports brick.Ports) (any, error) {
			if ports.ShortcutPresser == nil {
				return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, kindName)
			}

			return actionKey{label: label, pair: pair, presser: ports.ShortcutPresser}, nil
		})
		if registerError != nil {
			return registerError //nolint:wrapcheck // RegisterScoped already names the kind
		}
	}

	return nil
}

// actionKey presses one action's shortcut.
type actionKey struct {
	label   string
	pair    platform.ShortcutPair
	presser platform.ShortcutPresser
}

// Press sends this platform's shortcut for the action, naming any failure on
// the key by its category.
func (key actionKey) Press(pressContext context.Context) error {
	pressError := key.presser.PressShortcut(pressContext, key.presser.ForThisPlatform(key.pair))
	if pressError != nil {
		return fmt.Errorf("pressing %s: %w", key.label,
			brick.Categorised(platform.KeyFailureCategory(pressError), pressError))
	}

	return nil
}

// DefaultLabel is the action's label, shown when the key has none of its own.
func (key actionKey) DefaultLabel() string { return key.label }

// SendsKeystrokes marks the key as sending keystrokes.
func (actionKey) SendsKeystrokes() {}
