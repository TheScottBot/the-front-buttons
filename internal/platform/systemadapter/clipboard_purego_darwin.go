package systemadapter

import "context"

// pureClipboardWriter puts text on the clipboard through purego. It is a
// stub until its adapter is written: every call reports the bridge
// unavailable.
type pureClipboardWriter struct{}

// newPureClipboardWriter returns the ClipboardWriter through purego.
func newPureClipboardWriter() pureClipboardWriter { return pureClipboardWriter{} }

// WriteClipboard reports the bridge unavailable.
func (pureClipboardWriter) WriteClipboard(context.Context, string) error { return ErrBridgeUnavailable }
