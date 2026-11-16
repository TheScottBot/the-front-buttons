//go:build !cgo

package systemadapter

// CgoAvailable reports whether this build has cgo, so the cgo bridge can be
// used. Without it, the macOS constructors give the purego adapters whatever
// bridge they are asked for.
const CgoAvailable = false
