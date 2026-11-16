package objcbridge_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

func TestCFBoolean_GivesCoreFoundationsOwnConstants(t *testing.T) {
	t.Parallel()

	library := coreFoundation(t)

	for _, example := range []struct {
		value    bool
		constant string
	}{
		{value: true, constant: "kCFBooleanTrue"},
		{value: false, constant: "kCFBooleanFalse"},
	} {
		want, constantError := objcbridge.Constant(library, example.constant)
		if constantError != nil {
			t.Fatalf("reading %s: %v", example.constant, constantError)
		}

		got, booleanError := objcbridge.CFBoolean(example.value)
		if booleanError != nil {
			t.Fatalf("CFBoolean(%v): %v", example.value, booleanError)
		}

		if got != want {
			t.Errorf("CFBoolean(%v): want %s itself (%#x), got %#x", example.value, example.constant, want, got)
		}

		if truth, isBoolean := objcbridge.CFBool(got); !isBoolean || truth != example.value {
			t.Errorf("CFBoolean(%v): want it read back as %v, got %v, %v", example.value, example.value, truth, isBoolean)
		}
	}
}

func TestCFBoolean_LeavesTheRetainCountAlone(t *testing.T) {
	t.Parallel()

	functions := loadTestFunctions(t)

	value, booleanError := objcbridge.CFBoolean(true)
	if booleanError != nil {
		t.Fatalf("CFBoolean(true): %v", booleanError)
	}

	before := functions.retainCount(value)

	for range 3 {
		_, againError := objcbridge.CFBoolean(true)
		if againError != nil {
			t.Fatalf("CFBoolean(true): %v", againError)
		}
	}

	if after := functions.retainCount(value); after != before {
		t.Errorf("want the constant's retain count unchanged at %d, got %d", before, after)
	}
}
