//go:build cgo

package hidtransport

import "testing"

// TestHIDAPIThread_RaisesAPanicInTheCaller wants a panic on hidapi's thread
// raised again in the caller, where the watcher's recover catches it (finding
// TF5), and the thread still running calls afterwards.
func TestHIDAPIThread_RaisesAPanicInTheCaller(t *testing.T) {
	t.Parallel()

	var recovered any

	func() {
		defer func() { recovered = recover() }()

		onHIDAPIThread(func() { panic("an invalid string") })
	}()

	if recovered != "an invalid string" {
		t.Errorf("want the panic raised in the caller, recovered %v", recovered)
	}

	ran := false

	onHIDAPIThread(func() { ran = true })

	if !ran {
		t.Error("want hidapi's thread still running calls after a panic")
	}
}
