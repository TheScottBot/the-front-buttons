//go:build windows

package systemadapter

// NewStopRequestNamed is a stop request on an event of the test's own, so a
// test never stops, or is stopped by, a daemon running in the same session.
func NewStopRequestNamed(name string) StopRequest { return StopRequest{name: name} }
