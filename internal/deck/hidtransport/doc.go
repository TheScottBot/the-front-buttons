// Package hidtransport is the HID adapter behind the DeckDevice and DeckWatcher
// ports.
//
// Watcher holds the transport's own logic: the process wide lock, the write
// retry, the unlocked read poll, the enumeration panic recovery and the busy
// against disconnected decision. Below it sits a small driver interface, so
// that logic is shared by every driver rather than copied. hidapi_driver.go is
// the driver through go-hid, built only with cgo; each other driver lives in
// its own file and adds itself to builtDrivers. NewWatcher uses the driver
// asked for when the build holds it, and otherwise falls back to another.
//
// Keeping native code to this package and the system adapter means every other
// package builds and tests with nothing but the Go toolchain.
package hidtransport
