package mural_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/mural"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// boardLink is a mural's own address, as its browser shows it.
const boardLink = "https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d"

var errRefused = errors.New("refused")

// noAppearance is a key's appearance with nothing set.
func noAppearance() brick.Appearance {
	return brick.Appearance{Label: "", Icon: "", Background: "", IconFile: ""}
}

// presserFake records the shortcuts pressed, choosing a Mac's half of a pair.
type presserFake struct {
	pressed []platform.Shortcut
	failure error
}

func (fake *presserFake) PressShortcut(_ context.Context, shortcut platform.Shortcut) error {
	fake.pressed = append(fake.pressed, shortcut)

	return fake.failure
}

func (*presserFake) ForThisPlatform(pair platform.ShortcutPair) platform.Shortcut { return pair.Mac }

func (*presserFake) DescribeShortcut(platform.Shortcut) string { return "" }

// shareKey builds the share key with a message, over the clipboard and
// presser given.
//
//nolint:ireturn // the brick is exercised through the interface the daemon uses
func shareKey(
	t *testing.T, message string, clipboard platform.ClipboardWriter, presser platform.ShortcutPresser,
) brick.Pressable {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.ShareKindName)
	parameters := &mural.ShareParameters{Appearance: noAppearance(), Link: boardLink, Message: message}

	constructed, constructError := kind.Construct(parameters, brick.Ports{
		ClipboardWriter: clipboard, ShortcutPresser: presser,
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	pressable, _ := constructed.(brick.Pressable)

	return pressable
}

// TestShareLink_PastesTheMessageAndLinkWhereTyping: the author asked for a
// key that pastes a mural's link with what it is for: a press puts the
// message and the link on the clipboard and pastes them into whatever has the
// keyboard focus, as Command V on a Mac and Ctrl V on Windows do; with no
// message, the link alone.
func TestShareLink_PastesTheMessageAndLinkWhereTyping(t *testing.T) {
	t.Parallel()

	for message, want := range map[string]string{
		"Retro board for today:": "Retro board for today: " + boardLink,
		"":                       boardLink,
	} {
		clipboard := testsupport.NewClipboardTestDouble()
		presser := &presserFake{pressed: nil, failure: nil}

		pressError := shareKey(t, message, clipboard, presser).Press(t.Context())
		if pressError != nil {
			t.Fatal(pressError)
		}

		if got := clipboard.Written(); len(got) != 1 || got[0] != want {
			t.Errorf("%q: want %q on the clipboard, got %q", message, want, got)
		}

		paste := platform.Shortcut{Key: "V", Control: false, Alt: false, Shift: false, Command: true}
		if len(presser.pressed) != 1 || presser.pressed[0] != paste {
			t.Errorf("%q: want one paste, got %+v", message, presser.pressed)
		}
	}
}

// TestShareLink_ShowsWhyItCouldNotPaste: a clipboard that will not take the
// text shows clipboard_failed and pastes nothing; a paste that will not send
// shows why, as any key press does.
func TestShareLink_ShowsWhyItCouldNotPaste(t *testing.T) {
	t.Parallel()

	clipboard := testsupport.NewClipboardTestDouble()
	clipboard.FailNext(errRefused)

	presser := &presserFake{pressed: nil, failure: nil}

	pressError := shareKey(t, "", clipboard, presser).Press(t.Context())
	if category := brick.ReasonCategoryOf(pressError); category != "clipboard_failed" || len(presser.pressed) != 0 {
		t.Errorf("clipboard refused: want clipboard_failed and no paste, got %q and %d", category, len(presser.pressed))
	}

	refusing := &presserFake{pressed: nil, failure: platform.ErrKeyNotPermitted}

	pressError = shareKey(t, "", testsupport.NewClipboardTestDouble(), refusing).Press(t.Context())

	want := platform.KeyFailureCategory(platform.ErrKeyNotPermitted)
	if category := brick.ReasonCategoryOf(pressError); category != want {
		t.Errorf("paste refused: want the key press's reason, got %q", category)
	}
}

// TestShareLink_NeedsTheClipboardAndKeyPresses: the key refuses to construct
// without either port, and marks itself as sending key presses, so on an app
// screen its application comes to the front first.
func TestShareLink_NeedsTheClipboardAndKeyPresses(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.ShareKindName)
	parameters := &mural.ShareParameters{Appearance: noAppearance(), Link: boardLink, Message: ""}

	for name, ports := range map[string]brick.Ports{
		"no clipboard": {ShortcutPresser: &presserFake{pressed: nil, failure: nil}},
		"no presser":   {ClipboardWriter: testsupport.NewClipboardTestDouble()},
	} {
		_, constructError := kind.Construct(parameters, ports)
		if !errors.Is(constructError, brick.ErrPortMissing) {
			t.Errorf("%s: want the port missing, got %v", name, constructError)
		}
	}

	key := shareKey(t, "", testsupport.NewClipboardTestDouble(), &presserFake{pressed: nil, failure: nil})
	if _, sends := key.(brick.KeystrokeSender); !sends {
		t.Error("want the key marked as sending key presses")
	}
}
