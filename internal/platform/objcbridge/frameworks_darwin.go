package objcbridge

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// CFTypeRef is a Core Foundation object: a CFStringRef, CFNumberRef,
// AXUIElementRef and the like. Zero is nil.
type CFTypeRef uintptr

// ErrSymbolMissing reports a constant a library does not export.
var ErrSymbolMissing = errors.New("symbol missing")

// The binaries of the system libraries the adapters load. IOKit is for the
// HID driver; libobjc holds the autorelease pool calls.
const (
	PathCoreFoundation      = "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"
	PathFoundation          = "/System/Library/Frameworks/Foundation.framework/Foundation"
	PathAppKit              = "/System/Library/Frameworks/AppKit.framework/AppKit"
	PathApplicationServices = "/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices"
	PathIOKit               = "/System/Library/Frameworks/IOKit.framework/IOKit"
	pathObjectiveC          = "/usr/lib/libobjc.A.dylib"
)

//nolint:gochecknoglobals // each library is opened once per process and its handle kept
var (
	loadedLibraries     = map[string]uintptr{}
	loadedLibrariesLock sync.Mutex
)

// Framework opens the library at path, once per path, and returns its
// handle. Later calls with the same path return the same handle without
// opening it again.
func Framework(path string) (uintptr, error) {
	loadedLibrariesLock.Lock()
	defer loadedLibrariesLock.Unlock()

	if handle, loaded := loadedLibraries[path]; loaded {
		return handle, nil
	}

	handle, openError := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if openError != nil {
		return 0, fmt.Errorf("opening %s: %w", path, openError)
	}

	loadedLibraries[path] = handle

	return handle, nil
}

// Function points target, a pointer to a Go function variable, at the
// library's exported C function called name. It panics when the library does
// not export it, as purego.RegisterLibFunc does.
func Function(target any, library uintptr, name string) {
	purego.RegisterLibFunc(target, library, name)
}

// Constant reads the value of an exported Core Foundation constant, such as
// kCFBooleanTrue or kAXTrustedCheckOptionPrompt. A constant the library does
// not export is reported with ErrSymbolMissing, never a crash.
func Constant(library uintptr, name string) (CFTypeRef, error) {
	address, lookupError := purego.Dlsym(library, name)
	if lookupError != nil || address == 0 {
		return 0, fmt.Errorf("%w: %s", ErrSymbolMissing, name)
	}

	// The symbol's address is where the library keeps the constant's value,
	// in memory Go does not manage; reading it needs unsafe.
	return *(*CFTypeRef)(unsafe.Add(unsafe.Pointer(nil), address)), nil //nolint:gosec // see above
}
