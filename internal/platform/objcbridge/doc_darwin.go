// Package objcbridge calls macOS frameworks from Go without cgo, through
// purego. It loads the frameworks once, converts strings, numbers, booleans,
// arrays and dictionaries between Go and Core Foundation, manages retain
// counts and autorelease pools, and wraps the Accessibility calls the
// adapters share.
//
// Adapters send Objective-C messages with purego's objc package directly;
// this package holds what two or more adapters need, so no helper is written
// twice. It builds on macOS only.
package objcbridge
