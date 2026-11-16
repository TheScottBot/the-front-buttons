package systemadapter

import "context"

// StopRequest is the stop request on macOS, where there is none: launchd stops
// the daemon with a signal, which ShutdownSignals already catches.
type StopRequest struct{}

// NewStopRequest returns this machine's stop request.
func NewStopRequest() StopRequest { return StopRequest{} }

// Notify returns the parent context unchanged: no stop request can arrive.
func (StopRequest) Notify(parent context.Context) (context.Context, func(), error) {
	return parent, func() {}, nil
}

// Request reports stopping unavailable.
func (StopRequest) Request() error { return ErrStopUnavailable }
