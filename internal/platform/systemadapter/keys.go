package systemadapter

import (
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// kVK_F13 to kVK_F20 from the macOS SDK's HIToolbox Events.h, read on 12
// November 2026. macOS defines no F21 to F24.
const (
	macOSKeyF13 = 0x69
	macOSKeyF14 = 0x6B
	macOSKeyF15 = 0x71
	macOSKeyF16 = 0x6A
	macOSKeyF17 = 0x40
	macOSKeyF18 = 0x4F
	macOSKeyF19 = 0x50
	macOSKeyF20 = 0x5A
)

// macOSKeyCodes maps each function key macOS has to its virtual key code.
//
//nolint:gochecknoglobals // a fixed table
var macOSKeyCodes = map[string]uint16{
	"F13": macOSKeyF13, "F14": macOSKeyF14, "F15": macOSKeyF15, "F16": macOSKeyF16,
	"F17": macOSKeyF17, "F18": macOSKeyF18, "F19": macOSKeyF19, "F20": macOSKeyF20,
}

// The Windows virtual key codes run VK_F13 to VK_F24 in order, 0x7C to 0x87,
// as golang.org/x/sys v0.49.0 defines them.
const (
	windowsVirtualKeyF13 = 0x7C
	firstExtendedKey     = 13
	lastExtendedKey      = 24
)

// MacOSKeyCode returns a function key's macOS virtual key code.
func MacOSKeyCode(key string) (uint16, error) {
	code, known := macOSKeyCodes[key]
	if !known {
		return 0, fmt.Errorf("%w: %s on macOS", platform.ErrKeyUnsupported, key)
	}

	return code, nil
}

// WindowsVirtualKey returns a function key's Windows virtual key code.
func WindowsVirtualKey(key string) (uint16, error) {
	var number int

	_, scanError := fmt.Sscanf(key, "F%d", &number)
	if scanError != nil || number < firstExtendedKey || number > lastExtendedKey || key != fmt.Sprintf("F%d", number) {
		return 0, fmt.Errorf("%w: %s on Windows", platform.ErrKeyUnsupported, key)
	}

	return uint16(windowsVirtualKeyF13 + number - firstExtendedKey), nil
}
