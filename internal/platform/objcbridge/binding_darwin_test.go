package objcbridge_test

import (
	"errors"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

var errBindTest = errors.New("the binding test's own failure")

func TestBind_RunsOnce_AndGivesEveryCallTheSameResult(t *testing.T) {
	t.Parallel()

	runs := 0
	bound := objcbridge.Bind(func() (int, error) {
		runs++

		return 42, nil
	})

	for range 3 {
		value, bindError := bound()
		if value != 42 || bindError != nil {
			t.Errorf("want 42 and no error, got %d and %v", value, bindError)
		}
	}

	if runs != 1 {
		t.Errorf("want bind run once, got %d", runs)
	}
}

func TestBind_ReportsAMissingFunction_AsBindingFailed_WithoutPanicking(t *testing.T) {
	t.Parallel()

	runs := 0
	bound := objcbridge.Bind(func() (func(), error) {
		runs++

		var missing func()

		objcbridge.Function(&missing, coreFoundation(t), "CFNoSuchFunctionAnywhere")

		return missing, nil
	})

	for range 2 {
		missing, bindError := bound()
		if !errors.Is(bindError, objcbridge.ErrBindingFailed) || missing != nil {
			t.Errorf("want ErrBindingFailed and no function, got %v and a function %v", bindError, missing != nil)
		}
	}

	if runs != 1 {
		t.Errorf("want bind run once, failure included, got %d", runs)
	}
}

func TestBind_WrapsBindsOwnError_InBindingFailed(t *testing.T) {
	t.Parallel()

	_, bindError := objcbridge.Bind(func() (int, error) { return 7, errBindTest })()

	if !errors.Is(bindError, objcbridge.ErrBindingFailed) || !errors.Is(bindError, errBindTest) {
		t.Errorf("want ErrBindingFailed wrapping the bind's own error, got %v", bindError)
	}
}

func TestClass_FindsAClass_AndReportsAMissingOne(t *testing.T) {
	t.Parallel()

	class, classError := objcbridge.Class("NSObject")
	if class == 0 || classError != nil {
		t.Errorf("NSObject: want the class, got %#x and %v", class, classError)
	}

	missing, missingError := objcbridge.Class("FrontButtonsNoSuchClass")
	if missing != 0 || !errors.Is(missingError, objcbridge.ErrBindingFailed) {
		t.Errorf("a missing class: want ErrBindingFailed, got %#x and %v", missing, missingError)
	}
}

func TestSelectors_AreCheckedAgainstWhatTheClassAnswers(t *testing.T) {
	t.Parallel()

	const (
		classMethod    = "stringWithUTF8String:"
		instanceMethod = "length"
	)

	_, loadError := objcbridge.Framework(objcbridge.PathFoundation)
	if loadError != nil {
		t.Fatal(loadError)
	}

	stringClass, classError := objcbridge.Class("NSString")
	if classError != nil {
		t.Fatal(classError)
	}

	for _, example := range []struct {
		name     string
		lookup   func(objc.Class, string) (objc.SEL, error)
		class    objc.Class
		selector string
		answered bool
	}{
		{"a class method", objcbridge.ClassSelector, stringClass, classMethod, true},
		{"a misspelt class method", objcbridge.ClassSelector, stringClass, "stringWithUTF8Strin:", false},
		{"an instance method asked as a class method", objcbridge.ClassSelector, stringClass, instanceMethod, false},
		{"no class", objcbridge.ClassSelector, 0, classMethod, false},
		{"an instance method", objcbridge.InstanceSelector, stringClass, instanceMethod, true},
		{"a misspelt instance method", objcbridge.InstanceSelector, stringClass, "frontButtonsLength", false},
		{"a class method asked as an instance method", objcbridge.InstanceSelector, stringClass, classMethod, false},
		{"no class for an instance method", objcbridge.InstanceSelector, 0, instanceMethod, false},
	} {
		selector, lookupError := example.lookup(example.class, example.selector)

		if example.answered && (selector != objc.RegisterName(example.selector) || lookupError != nil) {
			t.Errorf("%s: want the selector %s, got %#x and %v", example.name, example.selector, selector, lookupError)
		}

		if !example.answered && (selector != 0 || !errors.Is(lookupError, objcbridge.ErrBindingFailed)) {
			t.Errorf("%s: want ErrBindingFailed, got %#x and %v", example.name, selector, lookupError)
		}
	}
}
