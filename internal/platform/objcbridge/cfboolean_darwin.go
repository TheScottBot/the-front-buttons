package objcbridge

import (
	"fmt"
	"sync"
)

// coreFoundationBooleans are kCFBooleanFalse and kCFBooleanTrue, read once.
type coreFoundationBooleans struct {
	falseValue CFTypeRef
	trueValue  CFTypeRef
}

//nolint:gochecknoglobals // the two constants are read once, on first use
var readBooleans = sync.OnceValues(func() (coreFoundationBooleans, error) {
	library, loadError := Framework(PathCoreFoundation)
	if loadError != nil {
		return coreFoundationBooleans{}, fmt.Errorf("reading the Core Foundation booleans: %w", loadError)
	}

	falseValue, falseError := Constant(library, "kCFBooleanFalse")
	if falseError != nil {
		return coreFoundationBooleans{}, falseError
	}

	trueValue, trueError := Constant(library, "kCFBooleanTrue")
	if trueError != nil {
		return coreFoundationBooleans{}, trueError
	}

	return coreFoundationBooleans{falseValue: falseValue, trueValue: trueValue}, nil
})

// CFBoolean gives kCFBooleanTrue or kCFBooleanFalse, to pass where macOS
// takes a CFBoolean, such as an Accessibility attribute's new value. The
// constants belong to Core Foundation: they are not owned and never
// released. A constant Core Foundation does not export is reported with
// ErrSymbolMissing.
func CFBoolean(value bool) (CFTypeRef, error) {
	booleans, readError := readBooleans()
	if readError != nil {
		return 0, readError
	}

	if value {
		return booleans.trueValue, nil
	}

	return booleans.falseValue, nil
}
