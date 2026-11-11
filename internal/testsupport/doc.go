// Package testsupport holds the test doubles and fixtures shared across
// packages: a table of fake brick kinds and presets, so the configuration,
// preset and editor tests all exercise the same fakes rather than each keeping
// its own (engineering datum 0.7).
//
// It is imported only by tests. Nothing here is a real brick.
package testsupport
