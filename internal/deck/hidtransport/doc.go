// Package hidtransport is the hidapi adapter behind the DeckDevice and DeckWatcher
// ports, and the only package in the module that uses cgo.
//
// Keeping cgo to this one package means every other package builds and tests with
// nothing but the Go toolchain.
package hidtransport
