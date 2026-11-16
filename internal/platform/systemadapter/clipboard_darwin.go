//go:build darwin && cgo

package systemadapter

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#import <AppKit/AppKit.h>

// frontButtonsFreeClipboardText frees a string made by C.CString.
static void frontButtonsFreeClipboardText(char *text) {
	free(text);
}

// frontButtonsWriteClipboard puts plain text on the general pasteboard in
// place of what was there. It returns 1 if the pasteboard took it.
static int frontButtonsWriteClipboard(const char *text) {
	@autoreleasepool {
		NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
		[pasteboard clearContents];

		return [pasteboard setString:[NSString stringWithUTF8String:text] forType:NSPasteboardTypeString] ? 1 : 0;
	}
}
*/
import "C"

import (
	"context"
	"errors"
)

// errClipboardRefused reports a pasteboard that would not take the text.
var errClipboardRefused = errors.New("the clipboard refused the text")

// ClipboardWriter puts text on the clipboard on macOS through AppKit's
// general pasteboard, which needs no permission.
type ClipboardWriter struct{}

// cgoClipboardWriter returns this machine's ClipboardWriter.
func cgoClipboardWriter() ClipboardWriter { return ClipboardWriter{} }

// WriteClipboard puts the text on the clipboard in place of what was there.
func (ClipboardWriter) WriteClipboard(_ context.Context, text string) error {
	written := C.CString(text)
	defer C.frontButtonsFreeClipboardText(written)

	if C.frontButtonsWriteClipboard(written) == 0 {
		return errClipboardRefused
	}

	return nil
}
