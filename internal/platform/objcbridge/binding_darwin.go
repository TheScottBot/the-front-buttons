package objcbridge

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"
)

// ErrBindingFailed reports a binding that could not be made: a library,
// class, selector or C function the system does not have.
var ErrBindingFailed = errors.New("binding failed")

// Bind returns a function that runs bind on its first call only and gives
// every call that same result. A panic inside bind, such as Function's for a
// missing C function, is recovered; it and any error bind returns are given
// as an error wrapping ErrBindingFailed, so a missing system function never
// stops the process.
func Bind[T any](bind func() (T, error)) func() (T, error) {
	return sync.OnceValues(func() (bound T, bindError error) { //nolint:nonamedreturns // the deferred recovery sets them
		defer func() {
			if recovered := recover(); recovered != nil {
				var zero T

				bound, bindError = zero, fmt.Errorf("%w: %v", ErrBindingFailed, recovered)
			}
		}()

		bound, bindError = bind()
		if bindError != nil && !errors.Is(bindError, ErrBindingFailed) {
			bindError = fmt.Errorf("%w: %w", ErrBindingFailed, bindError)
		}

		return bound, bindError
	})
}

// Class returns the named Objective-C class. A class the runtime does not
// have, because its framework is not loaded or the name is wrong, is an error
// wrapping ErrBindingFailed.
func Class(name string) (objc.Class, error) {
	class := objc.GetClass(name)
	if class == 0 {
		return 0, fmt.Errorf("%w: no class %s", ErrBindingFailed, name)
	}

	return class, nil
}

// ClassSelector registers the selector called name and checks that the class
// answers it as a class method, such as generalPasteboard. A class that does
// not is an error wrapping ErrBindingFailed, found at bind time rather than as
// an unrecognised selector exception, which would abort the process.
func ClassSelector(class objc.Class, name string) (objc.SEL, error) {
	selector := objc.RegisterName(name)

	if class == 0 || !objc.Send[bool](objc.ID(class), objc.RegisterName("respondsToSelector:"), selector) {
		return 0, fmt.Errorf("%w: no class method %s", ErrBindingFailed, name)
	}

	return selector, nil
}

// InstanceSelector registers the selector called name and checks that the
// class's instances answer it, such as clearContents. One that does not is an
// error wrapping ErrBindingFailed, found at bind time.
func InstanceSelector(class objc.Class, name string) (objc.SEL, error) {
	selector := objc.RegisterName(name)

	if class == 0 || !objc.Send[bool](objc.ID(class), objc.RegisterName("instancesRespondToSelector:"), selector) {
		return 0, fmt.Errorf("%w: no instance method %s", ErrBindingFailed, name)
	}

	return selector, nil
}
