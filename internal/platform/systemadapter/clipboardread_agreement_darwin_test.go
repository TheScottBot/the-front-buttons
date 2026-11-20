//nolint:testpackage // the clipboard's binding and pasteboard reader are unexported
package systemadapter

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// readableTexts are what a read must give back as it was written: the texts
// a write carries unchanged, other than the empty one, which a read gives as
// no text.
func readableTexts() []string {
	var texts []string

	for written, kept := range clipboardTexts() {
		if written != "" && written == kept {
			texts = append(texts, written)
		}
	}

	return texts
}

func TestPureClipboard_ReadsEachText_FromAPasteboard(t *testing.T) {
	t.Parallel()

	board := newTestPasteboard(t)
	messages, _ := clipboardBinding()

	for _, written := range readableTexts() {
		var (
			writeError, readError error
			got                   string
		)

		objcbridge.WithAutoreleasePool(func() {
			writeError = writePasteboard(messages, board.pasteboard, written)
			got, readError = readPasteboard(messages, board.pasteboard)
		})

		if writeError != nil || readError != nil || got != written {
			t.Errorf("%q: want it read back, got %q (%v, %v)", written, got, writeError, readError)
		}
	}
}

func TestPureClipboard_ReadingAPasteboardWithNoText_SaysItIsEmpty(t *testing.T) {
	t.Parallel()

	board := newTestPasteboard(t)
	messages, _ := clipboardBinding()

	var readError error

	objcbridge.WithAutoreleasePool(func() { _, readError = readPasteboard(messages, board.pasteboard) })

	if !errors.Is(readError, platform.ErrClipboardEmpty) {
		t.Errorf("want the clipboard empty, got %v", readError)
	}
}

func TestPureClipboard_ReadBindingAMissingName_ReportsTheBridgeUnavailable(t *testing.T) {
	t.Parallel()

	names := systemClipboardNames()
	names.stringForType = "stringFor:"

	// The binding fails before any pasteboard is reached, so this reads
	// nothing.
	binding := objcbridge.Bind(func() (clipboardMessages, error) { return bindClipboard(names) })

	_, readError := readClipboard(binding)
	if !errors.Is(readError, ErrBridgeUnavailable) || !errors.Is(readError, objcbridge.ErrBindingFailed) {
		t.Errorf("want ErrBridgeUnavailable for a failed binding, got %v", readError)
	}
}

func TestClipboardAgreement_BothBridgesMakeAReader(t *testing.T) {
	t.Parallel()

	cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) bool {
		t.Helper()

		return NewClipboardReader(bridge) != nil
	})

	if !cgo || !purego {
		t.Errorf("want a reader through each bridge, got cgo %v and purego %v", cgo, purego)
	}
}

// TestClipboardAgreement_EachBridgeReadsTheSameText writes the clipboard the
// person uses and reads it through each bridge in turn, then puts back the
// plain text that was there. It acts on the desktop, so it runs alone.
//
//nolint:paralleltest // writes the one clipboard the desktop has
func TestClipboardAgreement_EachBridgeReadsTheSameText(t *testing.T) {
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

	for _, written := range readableTexts() {
		writeError := newPureClipboardWriter().WriteClipboard(t.Context(), written)
		if writeError != nil {
			t.Fatalf("writing %q: %v", written, writeError)
		}

		cgo, purego := eachBridge(t, readThrough)

		if purego != written || (CgoAvailable && cgo != purego) {
			t.Errorf("%q: want it read through each bridge, got cgo %q and purego %q", written, cgo, purego)
		}
	}
}

// readThrough reads the clipboard the person uses through a bridge.
func readThrough(t *testing.T, bridge Bridge) string {
	t.Helper()

	if bridge == BridgeCgo && !CgoAvailable {
		t.Skip("this build has no cgo")
	}

	got, readError := NewClipboardReader(bridge).ReadClipboard(t.Context())
	if readError != nil {
		t.Fatalf("reading through %s: %v", bridge, readError)
	}

	return got
}
