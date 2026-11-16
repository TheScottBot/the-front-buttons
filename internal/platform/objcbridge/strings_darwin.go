package objcbridge

import (
	"strings"

	"github.com/ebitengine/purego/objc"
)

// encodingUTF8 is kCFStringEncodingUTF8, from CFString.h.
const encodingUTF8 = 0x08000100

// CFString makes a CFString holding text. Invalid UTF-8 is replaced with
// U+FFFD rather than refused. The caller owns it and releases it.
func CFString(text string) CFTypeRef {
	encoded := []byte(strings.ToValidUTF8(text, "\uFFFD"))

	var first *byte
	if len(encoded) > 0 {
		first = &encoded[0]
	}

	return core().createString(0, first, len(encoded), encodingUTF8, false)
}

// GoString reads a CFString as a Go string. It gives "" for nil and for
// anything that is not a string.
func GoString(ref CFTypeRef) string {
	if !IsType(ref, KindString) {
		return ""
	}

	functions := core()
	characters := cfRange{location: 0, length: functions.stringGetLength(ref)}

	needed := 0
	functions.stringGetBytes(ref, characters, encodingUTF8, 0, false, nil, 0, &needed)

	if needed == 0 {
		return ""
	}

	buffer := make([]byte, needed)
	written := 0
	functions.stringGetBytes(ref, characters, encodingUTF8, 0, false, &buffer[0], needed, &written)

	return string(buffer[:written])
}

// NSString makes an autoreleased NSString holding text, to be used inside
// WithAutoreleasePool. A CFString is an NSString, so this is CFString's
// result handed to the pool.
func NSString(text string) objc.ID {
	created := objc.ID(CFString(text))
	created.Send(core().selectorAutorelease)

	return created
}

// GoStringFromNS reads an NSString as a Go string. It gives "" for nil and
// for anything that is not an NSString.
func GoStringFromNS(object objc.ID) string {
	if object == 0 {
		return ""
	}

	functions := core()
	if !objc.Send[bool](object, functions.selectorIsKindOfClass, functions.stringClass) {
		return ""
	}

	return GoString(CFTypeRef(object))
}
