//nolint:testpackage // the clipboard's binding and pasteboard writer are unexported
package systemadapter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// clipboardTexts are what every write must carry as the cgo adapter does:
// empty, plain, accented, outside the Basic Multilingual Plane, long, and one
// holding a NUL, which ends a C string.
func clipboardTexts() map[string]string {
	return map[string]string{
		"":                      "",
		"https://example.com/a": "https://example.com/a",
		"Café crème, кириллица": "Café crème, кириллица",
		"Spotify 🎵 🎧":           "Spotify 🎵 🎧",
		"before\x00after":       "before",
		"line one\nline two\t.": "line one\nline two\t.",
	}
}

// testPasteboard is a pasteboard of the test's own, made with a unique name,
// so writing it never touches the clipboard the person uses.
type testPasteboard struct {
	pasteboard objc.ID
	readString objc.SEL
	stringType objc.ID
}

func newTestPasteboard(t *testing.T) testPasteboard {
	t.Helper()

	messages, bindError := clipboardBinding()
	if bindError != nil {
		t.Fatalf("binding the clipboard: %v", bindError)
	}

	unique, uniqueError := objcbridge.ClassSelector(objc.Class(messages.pasteboardClass), "pasteboardWithUniqueName")
	readString, readError := objcbridge.InstanceSelector(objc.Class(messages.pasteboardClass), "stringForType:")
	release, releaseError := objcbridge.InstanceSelector(objc.Class(messages.pasteboardClass), "releaseGlobally")

	joined := errors.Join(uniqueError, readError, releaseError)
	if joined != nil {
		t.Fatal(joined)
	}

	pasteboard := messages.pasteboardClass.Send(unique)
	if pasteboard == 0 {
		t.Fatal("want a pasteboard with a unique name, got nil")
	}

	t.Cleanup(func() { pasteboard.Send(release) })

	return testPasteboard{pasteboard: pasteboard, readString: readString, stringType: messages.stringType}
}

// text reads the pasteboard's plain text, or "" with false when it has none.
func (board testPasteboard) text() (string, bool) {
	var (
		text  string
		found bool
	)

	objcbridge.WithAutoreleasePool(func() {
		read := board.pasteboard.Send(board.readString, board.stringType)
		text, found = objcbridge.GoStringFromNS(read), read != 0
	})

	return text, found
}

func TestPureClipboard_PutsEachText_OnAPasteboard_AsTheCgoAdapterWould(t *testing.T) {
	t.Parallel()

	board := newTestPasteboard(t)

	messages, _ := clipboardBinding()

	for written, want := range clipboardTexts() {
		var writeError error

		objcbridge.WithAutoreleasePool(func() { writeError = writePasteboard(messages, board.pasteboard, written) })

		got, found := board.text()
		if writeError != nil || !found || got != want {
			t.Errorf("%q: want %q and no error, got %q (found %v) and %v", written, want, got, found, writeError)
		}
	}
}

func TestPureClipboard_RefusesTextThatIsNotUTF8_LeavingThePasteboardCleared(t *testing.T) {
	t.Parallel()

	board := newTestPasteboard(t)
	messages, _ := clipboardBinding()

	var firstError, writeError error

	objcbridge.WithAutoreleasePool(func() {
		firstError = writePasteboard(messages, board.pasteboard, "before")
		writeError = writePasteboard(messages, board.pasteboard, "bad \xff byte")
	})

	if firstError != nil {
		t.Fatal(firstError)
	}

	if got, found := board.text(); !errors.Is(writeError, errPureClipboardRefused) || found {
		t.Errorf("want the write refused and the pasteboard cleared, got %v and %q (found %v)", writeError, got, found)
	}
}

func TestPureClipboard_WritesPlainText(t *testing.T) {
	t.Parallel()

	messages, bindError := clipboardBinding()
	if bindError != nil {
		t.Fatal(bindError)
	}

	if got := objcbridge.GoStringFromNS(messages.stringType); got != "public.utf8-plain-text" {
		t.Errorf("want the plain text type public.utf8-plain-text, got %q", got)
	}
}

func TestPureClipboard_BindingAMissingName_ReportsTheBridgeUnavailable(t *testing.T) {
	t.Parallel()

	for field, rename := range map[string]func(*clipboardNames){
		"class":          func(names *clipboardNames) { names.pasteboardClass = "NSPasteboardNoSuchClass" },
		"class method":   func(names *clipboardNames) { names.generalPasteboard = "generalPasteBoard" },
		"clear":          func(names *clipboardNames) { names.clearContents = "clearContent" },
		"set the string": func(names *clipboardNames) { names.setStringForType = "setString:" },
		"constant":       func(names *clipboardNames) { names.stringType = "NSPasteboardTypeNoSuchType" },
	} {
		names := systemClipboardNames()
		rename(&names)

		// The binding fails before any pasteboard is reached, so this writes
		// nothing.
		binding := objcbridge.Bind(func() (clipboardMessages, error) { return bindClipboard(names) })

		writeError := writeClipboard(binding, "never written")
		if !errors.Is(writeError, ErrBridgeUnavailable) || !errors.Is(writeError, objcbridge.ErrBindingFailed) {
			t.Errorf("%s: want ErrBridgeUnavailable for a failed binding, got %v", field, writeError)
		}
	}
}

func TestClipboardAgreement_BothBridgesMakeAWriter(t *testing.T) {
	t.Parallel()

	cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) bool {
		t.Helper()

		return NewClipboardWriter(bridge) != nil
	})

	if !cgo || !purego {
		t.Errorf("want a writer through each bridge, got cgo %v and purego %v", cgo, purego)
	}
}

// generalPasteboardText reads the clipboard the person uses, through purego,
// whichever bridge wrote it.
func generalPasteboardText(t *testing.T) (string, bool) {
	t.Helper()

	messages, bindError := clipboardBinding()
	if bindError != nil {
		t.Fatal(bindError)
	}

	readString, readError := objcbridge.InstanceSelector(objc.Class(messages.pasteboardClass), "stringForType:")
	if readError != nil {
		t.Fatal(readError)
	}

	var board testPasteboard

	objcbridge.WithAutoreleasePool(func() {
		board = testPasteboard{
			pasteboard: messages.pasteboardClass.Send(messages.generalPasteboard),
			readString: readString, stringType: messages.stringType,
		}
	})

	return board.text()
}

// TestClipboardAgreement_EachBridgeWritesTheSameText writes the clipboard the
// person uses, through each bridge in turn, and reads it back, then puts back
// the plain text that was there. It acts on the desktop, so it runs alone.
//
//nolint:paralleltest // writes the one clipboard the desktop has
func TestClipboardAgreement_EachBridgeWritesTheSameText(t *testing.T) {
	actingAllowed(t)

	saved, hadText := generalPasteboardText(t)

	t.Cleanup(func() {
		if !hadText {
			return
		}

		restoreError := newPureClipboardWriter().WriteClipboard(t.Context(), saved)
		if restoreError != nil {
			t.Errorf("putting back the clipboard's text: %v", restoreError)
		}
	})

	for written, want := range clipboardTexts() {
		cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) string {
			t.Helper()

			writeError := writeThrough(t, bridge, written)
			if writeError != nil {
				t.Fatalf("writing %q: %v", written, writeError)
			}

			got, _ := generalPasteboardText(t)

			return got
		})

		if purego != want || (CgoAvailable && cgo != purego) {
			t.Errorf("%q: want %q through each bridge, got cgo %q and purego %q", written, want, cgo, purego)
		}
	}
}

// TestClipboardAgreement_EachBridgeRefusesTextThatIsNotUTF8 clears the
// clipboard the person uses through each bridge. It acts on the desktop, so
// it runs alone.
//
//nolint:paralleltest // writes the one clipboard the desktop has
func TestClipboardAgreement_EachBridgeRefusesTextThatIsNotUTF8(t *testing.T) {
	actingAllowed(t)

	cgoRefused, puregoRefused := eachBridge(t, func(t *testing.T, bridge Bridge) bool {
		t.Helper()

		return writeThrough(t, bridge, "bad \xff byte") != nil
	})

	if !puregoRefused || (CgoAvailable && !cgoRefused) {
		t.Errorf("text that is not UTF-8: want it refused through each bridge, got cgo %v and purego %v",
			cgoRefused, puregoRefused)
	}
}

// writeThrough writes the clipboard through the bridge, skipping the cgo half
// in a build without cgo.
func writeThrough(t *testing.T, bridge Bridge, text string) error {
	t.Helper()

	if bridge == BridgeCgo && !CgoAvailable {
		t.Skip("this build has no cgo")
	}

	writeError := NewClipboardWriter(bridge).WriteClipboard(t.Context(), text)
	if writeError != nil {
		return fmt.Errorf("through %s: %w", bridge, writeError)
	}

	return nil
}
