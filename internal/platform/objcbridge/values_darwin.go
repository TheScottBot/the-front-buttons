package objcbridge

import "runtime"

// Kind is a Core Foundation type IsType can recognise.
type Kind int

// The kinds IsType recognises. The zero Kind is none of them.
const (
	KindString Kind = iota + 1
	KindNumber
	KindBoolean
	KindArray
	KindAXElement
)

// numberTypeSInt64 is kCFNumberSInt64Type, from CFNumber.h.
const numberTypeSInt64 = 4

// NSPoint is AppKit's point, laid out as AppKit lays it out, for messages
// that take or return one by value.
type NSPoint struct{ X, Y float64 }

// NSSize is AppKit's size, laid out as AppKit lays it out.
type NSSize struct{ Width, Height float64 }

// NSRect is AppKit's rectangle, laid out as AppKit lays it out, so a message
// taking an NSRect is never given an NSSize by mistake.
type NSRect struct {
	Origin NSPoint
	Size   NSSize
}

// IsType reports whether ref is a non nil object of the given kind.
func IsType(ref CFTypeRef, kind Kind) bool {
	if ref == 0 {
		return false
	}

	functions := core()
	typeID, known := functions.typeIDs[kind]

	return known && functions.getTypeID(ref) == typeID
}

// CFInteger reads a CFNumber as an integer. It reports false for anything
// that is not a number, and for a number with a fraction or out of range.
func CFInteger(ref CFTypeRef) (int64, bool) {
	if !IsType(ref, KindNumber) {
		return 0, false
	}

	var value int64

	exact := core().numberGetValue(ref, numberTypeSInt64, &value)

	return value, exact
}

// CFBool reads a CFBoolean. It reports false for anything that is not one.
func CFBool(ref CFTypeRef) (bool, bool) {
	if !IsType(ref, KindBoolean) {
		return false, false
	}

	return core().booleanGetValue(ref), true
}

// CFArrayItems lists a CFArray's items in order, without retaining them;
// they live as long as the array does. Anything that is not an array has
// none.
func CFArrayItems(ref CFTypeRef) []CFTypeRef {
	if !IsType(ref, KindArray) {
		return nil
	}

	functions := core()
	count := functions.arrayGetCount(ref)
	items := make([]CFTypeRef, 0, count)

	for index := range count {
		items = append(items, functions.arrayGetValueAtIndex(ref, index))
	}

	return items
}

// CFDictionary makes an immutable CFDictionary holding the pairs, retaining
// each key and value. The caller owns it and releases it.
func CFDictionary(pairs map[CFTypeRef]CFTypeRef) CFTypeRef {
	functions := core()
	keys := make([]CFTypeRef, 0, len(pairs))
	values := make([]CFTypeRef, 0, len(pairs))

	for key, value := range pairs {
		keys = append(keys, key)
		values = append(values, value)
	}

	var firstKey, firstValue *CFTypeRef
	if len(keys) > 0 {
		firstKey, firstValue = &keys[0], &values[0]
	}

	return functions.createDictionary(0, firstKey, firstValue, len(keys),
		functions.dictionaryKeyCallBacks, functions.dictionaryValCallBacks)
}

// Release gives up one reference to ref. Nil is ignored.
func Release(ref CFTypeRef) {
	if ref != 0 {
		core().release(ref)
	}
}

// Retain takes one more reference to ref and returns it. Nil is ignored.
func Retain(ref CFTypeRef) CFTypeRef {
	if ref == 0 {
		return 0
	}

	return core().retain(ref)
}

// Equal reports whether two objects are equal by value, as CFEqual does.
// Nil equals only nil.
func Equal(first, second CFTypeRef) bool {
	if first == 0 || second == 0 {
		return first == second
	}

	return core().equal(first, second)
}

// WithAutoreleasePool runs work inside an autorelease pool, drained when work
// returns. Autorelease pools belong to a thread, so the goroutine stays on
// one operating system thread while work runs.
func WithAutoreleasePool(work func()) {
	functions := core()

	runtime.LockOSThread()

	defer runtime.UnlockOSThread()

	pool := functions.poolPush()
	defer functions.poolPop(pool)

	work()
}
