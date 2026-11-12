package systemadapter_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// TestMacOSKeyCode_MapsF13ToF20_AsTheSDKDefinesThem: the codes are kVK_F13 to
// kVK_F20 in the macOS SDK's HIToolbox Events.h, which defines no F21 to F24.
func TestMacOSKeyCode_MapsF13ToF20_AsTheSDKDefinesThem(t *testing.T) {
	t.Parallel()

	want := map[string]uint16{
		"F13": 0x69, "F14": 0x6B, "F15": 0x71, "F16": 0x6A,
		"F17": 0x40, "F18": 0x4F, "F19": 0x50, "F20": 0x5A,
	}

	for key, code := range want {
		got, mapError := systemadapter.MacOSKeyCode(key)
		if mapError != nil || got != code {
			t.Errorf("%s: want %#x, got %#x (%v)", key, code, got, mapError)
		}
	}

	for _, key := range []string{"F21", "F24", "F12", "A"} {
		_, mapError := systemadapter.MacOSKeyCode(key)
		if !errors.Is(mapError, platform.ErrKeyUnsupported) {
			t.Errorf("%s: want ErrKeyUnsupported, got %v", key, mapError)
		}
	}
}

// TestWindowsVirtualKey_MapsF13ToF24: VK_F13 to VK_F24 as golang.org/x/sys
// v0.49.0 defines them.
func TestWindowsVirtualKey_MapsF13ToF24(t *testing.T) {
	t.Parallel()

	for offset, key := range []string{"F13", "F14", "F15", "F16", "F17", "F18", "F19", "F20", "F21", "F22", "F23", "F24"} {
		got, mapError := systemadapter.WindowsVirtualKey(key)
		if want := uint16(0x7C + offset); mapError != nil || got != want {
			t.Errorf("%s: want %#x, got %#x (%v)", key, want, got, mapError)
		}
	}

	_, mapError := systemadapter.WindowsVirtualKey("F12")
	if !errors.Is(mapError, platform.ErrKeyUnsupported) {
		t.Errorf("F12: want ErrKeyUnsupported, got %v", mapError)
	}
}
