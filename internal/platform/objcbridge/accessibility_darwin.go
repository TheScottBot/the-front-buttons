package objcbridge

import (
	"fmt"
	"strconv"
)

// AXError is an Accessibility call's failure code, from AXError.h.
type AXError int32

// The AXError codes callers most often tell apart.
const (
	ErrAXIllegalArgument      AXError = -25201
	ErrAXCannotComplete       AXError = -25204
	ErrAXAttributeUnsupported AXError = -25205
	ErrAXActionUnsupported    AXError = -25206
	ErrAXAPIDisabled          AXError = -25211
	ErrAXNoValue              AXError = -25212
)

func (code AXError) Error() string {
	return "accessibility error " + strconv.Itoa(int(code))
}

// AXApplication makes the Accessibility element of the application with the
// given process ID. The caller owns it and releases it.
func AXApplication(processID int) CFTypeRef {
	return core().axCreateApplication(int32(processID)) //nolint:gosec // process IDs fit pid_t, a 32 bit int
}

// AXSystemWide makes the system wide Accessibility element. The caller owns
// it and releases it.
func AXSystemWide() CFTypeRef {
	return core().axCreateSystemWide()
}

// AXCopy copies an attribute's value from an element. The caller owns the
// value and releases it. A failure, including a missing value, is an AXError.
func AXCopy(element CFTypeRef, attribute string) (CFTypeRef, error) {
	if element == 0 {
		return 0, fmt.Errorf("copying accessibility attribute %s: %w", attribute, ErrAXIllegalArgument)
	}

	name := CFString(attribute)
	defer Release(name)

	var value CFTypeRef

	code := AXError(core().axCopy(element, name, &value))
	if code != 0 {
		return 0, fmt.Errorf("copying accessibility attribute %s: %w", attribute, code)
	}

	if value == 0 {
		return 0, fmt.Errorf("copying accessibility attribute %s: %w", attribute, ErrAXNoValue)
	}

	return value, nil
}

// AXSet sets an attribute of an element. A failure is an AXError.
func AXSet(element CFTypeRef, attribute string, value CFTypeRef) error {
	if element == 0 {
		return fmt.Errorf("setting accessibility attribute %s: %w", attribute, ErrAXIllegalArgument)
	}

	name := CFString(attribute)
	defer Release(name)

	code := AXError(core().axSet(element, name, value))
	if code != 0 {
		return fmt.Errorf("setting accessibility attribute %s: %w", attribute, code)
	}

	return nil
}

// AXPerform performs an action, such as AXRaise, on an element. A failure is
// an AXError.
func AXPerform(element CFTypeRef, action string) error {
	if element == 0 {
		return fmt.Errorf("performing accessibility action %s: %w", action, ErrAXIllegalArgument)
	}

	name := CFString(action)
	defer Release(name)

	code := AXError(core().axPerform(element, name))
	if code != 0 {
		return fmt.Errorf("performing accessibility action %s: %w", action, code)
	}

	return nil
}

// AXProcessID reads the process ID of the application an element belongs to.
// A failure is an AXError.
func AXProcessID(element CFTypeRef) (int, error) {
	if element == 0 {
		return 0, fmt.Errorf("reading an accessibility element's process: %w", ErrAXIllegalArgument)
	}

	var processID int32

	code := AXError(core().axGetPid(element, &processID))
	if code != 0 {
		return 0, fmt.Errorf("reading an accessibility element's process: %w", code)
	}

	return int(processID), nil
}
