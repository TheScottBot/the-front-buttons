package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// errPureClipboardRefused reports a pasteboard that would not take the text,
// as the cgo adapter's errClipboardRefused does.
var errPureClipboardRefused = errors.New("the clipboard refused the text")

// clipboardNames are the class, selectors and constant the clipboard binds,
// so a test can bind a name the system does not have.
type clipboardNames struct {
	pasteboardClass   string
	generalPasteboard string
	clearContents     string
	setStringForType  string
	stringForType     string
	stringType        string
}

// systemClipboardNames are AppKit's own names.
func systemClipboardNames() clipboardNames {
	return clipboardNames{
		pasteboardClass:   "NSPasteboard",
		generalPasteboard: "generalPasteboard",
		clearContents:     "clearContents",
		setStringForType:  "setString:forType:",
		stringForType:     "stringForType:",
		stringType:        "NSPasteboardTypeString",
	}
}

// clipboardMessages are what WriteClipboard sends, bound once.
type clipboardMessages struct {
	pasteboardClass   objc.ID
	generalPasteboard objc.SEL
	clearContents     objc.SEL
	setStringForType  objc.SEL
	stringForType     objc.SEL
	stringType        objc.ID
}

//nolint:gochecknoglobals // bound once, on first use
var clipboardBinding = objcbridge.Bind(func() (clipboardMessages, error) {
	return bindClipboard(systemClipboardNames())
})

// bindClipboard loads AppKit and finds every class, selector and constant
// the clipboard needs.
func bindClipboard(names clipboardNames) (clipboardMessages, error) {
	appKit, loadError := objcbridge.Framework(objcbridge.PathAppKit)
	if loadError != nil {
		return clipboardMessages{}, fmt.Errorf("loading AppKit: %w", loadError)
	}

	class, classError := objcbridge.Class(names.pasteboardClass)
	general, generalError := objcbridge.ClassSelector(class, names.generalPasteboard)
	clearContents, clearError := objcbridge.InstanceSelector(class, names.clearContents)
	setString, setError := objcbridge.InstanceSelector(class, names.setStringForType)
	readString, readError := objcbridge.InstanceSelector(class, names.stringForType)
	stringType, typeError := objcbridge.Constant(appKit, names.stringType)

	bindError := errors.Join(classError, generalError, clearError, setError, readError, typeError)
	if bindError != nil {
		return clipboardMessages{}, bindError
	}

	return clipboardMessages{
		pasteboardClass:   objc.ID(class),
		generalPasteboard: general,
		clearContents:     clearContents,
		setStringForType:  setString,
		stringForType:     readString,
		stringType:        objc.ID(stringType),
	}, nil
}

// pureClipboardWriter puts text on the clipboard through purego, with
// AppKit's general pasteboard, which needs no permission.
type pureClipboardWriter struct{}

// newPureClipboardWriter returns the ClipboardWriter through purego.
func newPureClipboardWriter() pureClipboardWriter { return pureClipboardWriter{} }

// WriteClipboard puts the text on the clipboard in place of what was there.
func (pureClipboardWriter) WriteClipboard(_ context.Context, text string) error {
	return writeClipboard(clipboardBinding, text)
}

// writeClipboard puts the text on the general pasteboard with the messages
// binding gives, or reports the bridge unavailable when they could not be
// bound.
func writeClipboard(binding func() (clipboardMessages, error), text string) error {
	messages, bindError := binding()
	if bindError != nil {
		return fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
	}

	var writeError error

	objcbridge.WithAutoreleasePool(func() {
		writeError = writePasteboard(messages, messages.pasteboardClass.Send(messages.generalPasteboard), text)
	})

	return writeError
}

// writePasteboard clears the pasteboard and puts text on it as plain text.
// As the cgo adapter's C string does, the text ends at its first NUL, and
// text that is not UTF-8 is refused once the pasteboard is cleared.
func writePasteboard(messages clipboardMessages, pasteboard objc.ID, text string) error {
	text, _, _ = strings.Cut(text, "\x00")

	pasteboard.Send(messages.clearContents)

	if !utf8.ValidString(text) {
		return errPureClipboardRefused
	}

	if !objc.Send[bool](pasteboard, messages.setStringForType, objcbridge.NSString(text), messages.stringType) {
		return errPureClipboardRefused
	}

	return nil
}

// ReadClipboard reads the plain text on the clipboard.
func (pureClipboardWriter) ReadClipboard(_ context.Context) (string, error) {
	return readClipboard(clipboardBinding)
}

// readClipboard reads the general pasteboard's plain text with the messages
// binding gives, or reports the bridge unavailable when they could not be
// bound.
func readClipboard(binding func() (clipboardMessages, error)) (string, error) {
	messages, bindError := binding()
	if bindError != nil {
		return "", fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
	}

	var (
		text      string
		readError error
	)

	objcbridge.WithAutoreleasePool(func() {
		text, readError = readPasteboard(messages, messages.pasteboardClass.Send(messages.generalPasteboard))
	})

	return text, readError
}

// readPasteboard reads a pasteboard's plain text, or says it holds none.
func readPasteboard(messages clipboardMessages, pasteboard objc.ID) (string, error) {
	read := pasteboard.Send(messages.stringForType, messages.stringType)
	if read == 0 {
		return "", platform.ErrClipboardEmpty
	}

	return objcbridge.GoStringFromNS(read), nil
}
