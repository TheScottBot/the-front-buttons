// Package pressshortcut is the press_shortcut brick: pressing its key sends a
// key with any combination of modifiers held, such as ⌘C on a Mac or Ctrl+C
// on Windows, to whatever has the keyboard focus. The author asked for it to
// set up each machine's shortcuts in the same places (evaluation, deck
// testing): each machine's daemon reads its own configuration.
package pressshortcut

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// KindName is the kind's name in configuration.
const KindName = "press_shortcut"

// Parameters is the kind's one declaration of its parameters. Each modifier
// is named as each platform names it, for the editor's chips.
type Parameters struct {
	brick.Appearance

	Key     string `brick:"key,required" format:"shortcut_key" description:"The key to press: a letter, a digit, F1 to F20, or a named key."`
	Control bool   `brick:"control"      mac:"⌃ Control"       windows:"Ctrl"                                                                description:"Hold Control."`
	Alt     bool   `brick:"alt"          mac:"⌥ Option"        windows:"Alt"                                                                 description:"Hold Option on macOS, Alt on Windows."`
	Shift   bool   `brick:"shift"        mac:"⇧ Shift"         windows:"Shift"                                                               description:"Hold Shift."`
	Command bool   `brick:"command"      mac:"⌘ Command"       windows:"Win"                                                                 description:"Hold Command on macOS, the Windows key on Windows."`
}

// Register adds the kind to a registry.
func Register(registry *brick.Registry) error {
	return brick.Register(registry, KindName, construct) //nolint:wrapcheck // Register already names the kind
}

func construct(parameters Parameters, ports brick.Ports) (any, error) {
	if ports.ShortcutPresser == nil {
		return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, KindName)
	}

	return shortcutKey{
		shortcut: platform.Shortcut{
			Key: parameters.Key, Control: parameters.Control, Alt: parameters.Alt, Shift: parameters.Shift,
			Command: parameters.Command,
		},
		presser: ports.ShortcutPresser,
	}, nil
}

// shortcutKey presses one shortcut.
type shortcutKey struct {
	shortcut platform.Shortcut
	presser  platform.ShortcutPresser
}

// Press sends the shortcut, and names any failure on the key by its category.
func (key shortcutKey) Press(pressContext context.Context) error {
	pressError := key.presser.PressShortcut(pressContext, key.shortcut)
	if pressError != nil {
		return fmt.Errorf("pressing a shortcut: %w",
			brick.Categorised(platform.KeyFailureCategory(pressError), pressError))
	}

	return nil
}

// DefaultLabel is the shortcut as this platform writes it, shown when the key
// has no label of its own.
func (key shortcutKey) DefaultLabel() string {
	return key.presser.DescribeShortcut(key.shortcut)
}

// SendsKeystrokes marks the key as sending keystrokes.
func (shortcutKey) SendsKeystrokes() {}
