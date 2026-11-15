// Package mural is the keys for Mural, the online whiteboard, for running
// sessions on a board. mural_share_link pastes a board's link, with a message
// saying what it is for, wherever the person is typing, such as a meeting's
// chat; the timer keys (timer.go) start and add to a board's timer.
package mural

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// ShareKindName is the share key's name in configuration.
const ShareKindName = "mural_share_link"

// clipboardFailedCategory is what the share key shows when the clipboard
// would not take its text.
const clipboardFailedCategory = "clipboard_failed"

// shareLabel is the share key's label when it has none of its own.
const shareLabel = "Mural link"

// ShareParameters is the share key's one declaration of its parameters.
type ShareParameters struct {
	brick.Appearance

	Link    string `brick:"link,required" bounds:"1,2048" format:"mural_link"                                                                                             description:"The mural's link, as its browser shows it or as Mural's Share gives it."`
	Message string `brick:"message"       bounds:"0,500"  description:"What to paste before the link, such as what the board is for; left out, the link is pasted alone."`
}

// Register adds the Mural kinds to a registry.
func Register(registry *brick.Registry) error {
	shareError := brick.Register(registry, ShareKindName, constructShare)
	if shareError != nil {
		return shareError //nolint:wrapcheck // Register already names the kind
	}

	return registerTimers(registry)
}

func constructShare(parameters ShareParameters, ports brick.Ports) (any, error) {
	if ports.ClipboardWriter == nil || ports.ShortcutPresser == nil {
		return nil, fmt.Errorf("%w: %s needs the clipboard and key presses", brick.ErrPortMissing, ShareKindName)
	}

	text := parameters.Link
	if parameters.Message != "" {
		text = parameters.Message + " " + parameters.Link
	}

	return shareKey{text: text, clipboard: ports.ClipboardWriter, presser: ports.ShortcutPresser}, nil
}

// shareKey pastes its text where the person is typing.
type shareKey struct {
	text      string
	clipboard platform.ClipboardWriter
	presser   platform.ShortcutPresser
}

// pasteShortcut is the paste command: Command V on a Mac, Ctrl V on Windows.
func pasteShortcut() platform.ShortcutPair {
	return platform.ShortcutPair{
		Mac:     platform.Shortcut{Key: "V", Control: false, Alt: false, Shift: false, Command: true},
		Windows: platform.Shortcut{Key: "V", Control: true, Alt: false, Shift: false, Command: false},
	}
}

// Press puts the text on the clipboard and pastes it. The text is left on the
// clipboard, to paste again.
func (key shareKey) Press(pressContext context.Context) error {
	writeError := key.clipboard.WriteClipboard(pressContext, key.text)
	if writeError != nil {
		return fmt.Errorf("putting the link on the clipboard: %w", brick.Categorised(clipboardFailedCategory, writeError))
	}

	pressError := key.presser.PressShortcut(pressContext, key.presser.ForThisPlatform(pasteShortcut()))
	if pressError != nil {
		return fmt.Errorf("pasting the link: %w", brick.Categorised(platform.KeyFailureCategory(pressError), pressError))
	}

	return nil
}

// DefaultLabel names the key when it has no label of its own.
func (shareKey) DefaultLabel() string { return shareLabel }

// SendsKeystrokes marks the key as sending keystrokes.
func (shareKey) SendsKeystrokes() {}
