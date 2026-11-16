package systemadapter

import "errors"

// A stop request asks a running daemon, from another process, to stop as it
// does on a shutdown signal: blank and close the deck, then exit. Windows needs
// it because it has no way to send a signal to a process with no console:
// Task Scheduler and Stop-Process end the daemon outright, leaving the last
// images on the keys. macOS needs none, because launchd sends a signal.
var (
	// ErrStopRequested is the cause of a context ended by a stop request.
	ErrStopRequested = errors.New("asked to stop by -stop")
	// ErrNotRunning reports a stop request with no daemon to hear it.
	ErrNotRunning = errors.New("not running")
	// ErrStopUnavailable reports a system with no stop request; there the
	// daemon is stopped by a signal from its login agent.
	ErrStopUnavailable = errors.New("stopping from the command line is not available on this system; " +
		"use launchctl, or the uninstall script")
)
