package systemadapter

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/windows"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// stopEventName is the named event a running daemon waits on. Local\ puts it
// in the person's own logon session, so only a process in that session, which
// the default security of a named event further limits to the same account,
// can ask the daemon to stop.
const stopEventName = `Local\` + platform.ProductIdentifier + `.stop`

// StopRequest is the stop request on Windows: a named, automatically resetting
// event that the daemon waits on and Request sets.
type StopRequest struct {
	name string
}

// NewStopRequest returns this machine's stop request.
func NewStopRequest() StopRequest { return StopRequest{name: stopEventName} }

// Notify returns a context that ends, with ErrStopRequested as its cause, when
// a stop is requested, and a function that stops listening and must be called
// once the context is no longer needed. If the event cannot be made it returns
// the parent context with the error, so the daemon still runs, only without
// -stop.
func (request StopRequest) Notify(parent context.Context) (context.Context, func(), error) {
	stopEvent, stopError := openStopEvent(request.name)
	if stopError != nil {
		return parent, func() {}, stopError
	}

	// A private event that ends the wait when the context ends first.
	release, releaseError := windows.CreateEvent(nil, 1, 0, nil)
	if releaseError != nil {
		_ = windows.CloseHandle(stopEvent)

		return parent, func() {}, fmt.Errorf("making the stop release event: %w", releaseError)
	}

	stopped, cancel := context.WithCancelCause(parent)
	stopReleasing := context.AfterFunc(stopped, func() { _ = windows.SetEvent(release) })
	waited := make(chan struct{})

	go func() {
		defer close(waited)

		signalled, waitError := windows.WaitForMultipleObjects(
			[]windows.Handle{stopEvent, release}, false, windows.INFINITE)
		if waitError == nil && signalled == windows.WAIT_OBJECT_0 {
			cancel(ErrStopRequested)
		}
	}()

	return stopped, func() {
		stopReleasing()
		cancel(context.Canceled)
		// Set here, not left to the AfterFunc just unregistered, so the wait
		// always ends before the handles it waits on are closed.
		_ = windows.SetEvent(release)

		<-waited

		_ = windows.CloseHandle(release)
		_ = windows.CloseHandle(stopEvent)
	}, nil
}

// openStopEvent makes the named stop event, or opens it if another copy of the
// daemon already has, and clears it: a stop set before this daemon listened was
// meant for one that has gone.
func openStopEvent(name string) (windows.Handle, error) {
	namePointer, nameError := windows.UTF16PtrFromString(name)
	if nameError != nil {
		return 0, fmt.Errorf("naming the stop event: %w", nameError)
	}

	stopEvent, createError := windows.CreateEvent(nil, 0, 0, namePointer)
	if stopEvent == 0 {
		return 0, fmt.Errorf("making the stop event: %w", createError)
	}

	resetError := windows.ResetEvent(stopEvent)
	if resetError != nil {
		_ = windows.CloseHandle(stopEvent)

		return 0, fmt.Errorf("clearing the stop event: %w", resetError)
	}

	return stopEvent, nil
}

// Request asks the running daemon to stop, and returns without waiting for it
// to. It reports ErrNotRunning if no daemon in this session is listening.
func (request StopRequest) Request() error {
	namePointer, nameError := windows.UTF16PtrFromString(request.name)
	if nameError != nil {
		return fmt.Errorf("naming the stop event: %w", nameError)
	}

	stopEvent, openError := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, namePointer)
	if errors.Is(openError, windows.ERROR_FILE_NOT_FOUND) {
		return ErrNotRunning
	}

	if openError != nil {
		return fmt.Errorf("opening the stop event: %w", openError)
	}

	defer func() { _ = windows.CloseHandle(stopEvent) }()

	setError := windows.SetEvent(stopEvent)
	if setError != nil {
		return fmt.Errorf("setting the stop event: %w", setError)
	}

	return nil
}
