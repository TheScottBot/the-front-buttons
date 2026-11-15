package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ClipboardWriter puts text on the clipboard on Windows: the clipboard is
// opened, emptied and given the text as Unicode, in memory Windows takes over
// once it is set. Not yet run on a Windows machine: the author is testing on
// macOS only for now.
type ClipboardWriter struct{}

// The Win32 values the clipboard writer uses, from WinUser.h and WinBase.h:
// Unicode text, and memory the clipboard may move. Another program may hold
// the clipboard for a moment, so opening it is tried a few times.
const (
	clipboardUnicodeText = 13
	globalMoveable       = 0x0002
	clipboardOpenTries   = 5
	clipboardOpenWait    = 20 * time.Millisecond
)

//nolint:gochecknoglobals // the lazily loaded system calls Go's Windows package lacks
var (
	procOpenClipboard    = windows.NewLazySystemDLL("user32.dll").NewProc("OpenClipboard")
	procEmptyClipboard   = windows.NewLazySystemDLL("user32.dll").NewProc("EmptyClipboard")
	procSetClipboardData = windows.NewLazySystemDLL("user32.dll").NewProc("SetClipboardData")
	procCloseClipboard   = windows.NewLazySystemDLL("user32.dll").NewProc("CloseClipboard")
	procGlobalAlloc      = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalAlloc")
	procGlobalLock       = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalLock")
	procGlobalUnlock     = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalUnlock")
	procGlobalFree       = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalFree")
	procMoveMemory       = windows.NewLazySystemDLL("kernel32.dll").NewProc("RtlMoveMemory")
)

// The ways writing the clipboard fails.
var (
	errClipboardBusy    = errors.New("the clipboard is held by another program")
	errClipboardRefused = errors.New("the clipboard refused the text")
)

// NewClipboardWriter returns this machine's ClipboardWriter.
func NewClipboardWriter() ClipboardWriter { return ClipboardWriter{} }

// WriteClipboard puts the text on the clipboard in place of what was there.
func (ClipboardWriter) WriteClipboard(_ context.Context, text string) error {
	encoded, encodeError := windows.UTF16FromString(text)
	if encodeError != nil {
		return fmt.Errorf("encoding the text: %w", encodeError)
	}

	// The clipboard is opened for the thread that opens it.
	runtime.LockOSThread()

	defer runtime.UnlockOSThread()

	openError := openClipboard()
	if openError != nil {
		return openError
	}

	defer closeClipboard()

	if emptied, _, emptyError := procEmptyClipboard.Call(); emptied == 0 {
		return fmt.Errorf("%w: %w", errClipboardRefused, emptyError)
	}

	return setClipboardText(encoded)
}

// closeClipboard closes the clipboard, as each open must be.
func closeClipboard() {
	_, _, _ = procCloseClipboard.Call() //nolint:dogsled // closing cannot usefully fail
}

// openClipboard opens the clipboard, trying again while another program
// holds it.
func openClipboard() error {
	var openError error

	for range clipboardOpenTries {
		opened, _, callError := procOpenClipboard.Call(0)
		if opened != 0 {
			return nil
		}

		openError = callError

		time.Sleep(clipboardOpenWait)
	}

	return fmt.Errorf("%w: %w", errClipboardBusy, openError)
}

// setClipboardText copies the text into memory the clipboard may move and
// hands it over; the clipboard owns it once set, and it is freed only if not.
func setClipboardText(encoded []uint16) error {
	size := uintptr(len(encoded)) * unsafe.Sizeof(encoded[0])

	memory, _, allocError := procGlobalAlloc.Call(globalMoveable, size)
	if memory == 0 {
		return fmt.Errorf("%w: %w", errClipboardRefused, allocError)
	}

	locked, _, lockError := procGlobalLock.Call(memory)
	if locked == 0 {
		_, _, _ = procGlobalFree.Call(memory)

		return fmt.Errorf("%w: %w", errClipboardRefused, lockError)
	}

	// GlobalLock returns the memory's address, so Windows copies the text
	// there itself; unsafe is the only way to pass the text's address.
	text := uintptr(unsafe.Pointer(&encoded[0]))      //nolint:gosec // see above
	_, _, _ = procMoveMemory.Call(locked, text, size) //nolint:dogsled // copying cannot fail
	_, _, _ = procGlobalUnlock.Call(memory)           //nolint:dogsled // unlocking cannot usefully fail

	if set, _, setError := procSetClipboardData.Call(clipboardUnicodeText, memory); set == 0 {
		_, _, _ = procGlobalFree.Call(memory)

		return fmt.Errorf("%w: %w", errClipboardRefused, setError)
	}

	return nil
}
