package objcbridge

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// cfRange is CFRange: a location and a length, in UTF-16 units for strings.
type cfRange struct {
	location int
	length   int
}

// coreFunctions are the C functions and values the helpers call, bound once
// on first use.
type coreFunctions struct {
	release   func(ref CFTypeRef)
	retain    func(ref CFTypeRef) CFTypeRef
	equal     func(first, second CFTypeRef) bool
	getTypeID func(ref CFTypeRef) uintptr

	createString    func(allocator uintptr, bytes *byte, length int, encoding uint32, external bool) CFTypeRef
	stringGetLength func(ref CFTypeRef) int
	stringGetBytes  func(ref CFTypeRef, characters cfRange, encoding uint32, lossByte uint8,
		external bool, buffer *byte, bufferLength int, used *int) int

	numberGetValue       func(ref CFTypeRef, numberType int, value *int64) bool
	booleanGetValue      func(ref CFTypeRef) bool
	arrayGetCount        func(ref CFTypeRef) int
	arrayGetValueAtIndex func(ref CFTypeRef, index int) CFTypeRef
	createDictionary     func(allocator uintptr, keys, values *CFTypeRef, count int,
		keyCallBacks, valueCallBacks uintptr) CFTypeRef

	poolPush func() uintptr
	poolPop  func(pool uintptr)

	axCreateApplication func(processID int32) CFTypeRef
	axCreateSystemWide  func() CFTypeRef
	axCopy              func(element, attribute CFTypeRef, value *CFTypeRef) int32
	axSet               func(element, attribute, value CFTypeRef) int32
	axPerform           func(element, action CFTypeRef) int32
	axGetPid            func(element CFTypeRef, processID *int32) int32

	typeIDs                map[Kind]uintptr
	dictionaryKeyCallBacks uintptr
	dictionaryValCallBacks uintptr
	stringClass            objc.ID
	selectorAutorelease    objc.SEL
	selectorIsKindOfClass  objc.SEL
}

//nolint:gochecknoglobals // the frameworks are bound once, on first use
var core = sync.OnceValue(func() *coreFunctions {
	functions, loadError := loadCore()
	if loadError != nil {
		// Every library loaded here ships with macOS, so failing means this is
		// not a Mac the package can run on.
		panic(fmt.Sprintf("objcbridge: %v", loadError))
	}

	return functions
})

func loadCore() (*coreFunctions, error) {
	libraries := map[string]uintptr{}

	paths := []string{PathCoreFoundation, PathFoundation, PathAppKit, PathApplicationServices, pathObjectiveC}

	for _, path := range paths {
		handle, openError := Framework(path)
		if openError != nil {
			return nil, openError
		}

		libraries[path] = handle
	}

	var functions coreFunctions

	coreFoundation, services := libraries[PathCoreFoundation], libraries[PathApplicationServices]

	bindCoreFoundation(&functions, coreFoundation)

	Function(&functions.poolPush, libraries[pathObjectiveC], "objc_autoreleasePoolPush")
	Function(&functions.poolPop, libraries[pathObjectiveC], "objc_autoreleasePoolPop")

	Function(&functions.axCreateApplication, services, "AXUIElementCreateApplication")
	Function(&functions.axCreateSystemWide, services, "AXUIElementCreateSystemWide")
	Function(&functions.axCopy, services, "AXUIElementCopyAttributeValue")
	Function(&functions.axSet, services, "AXUIElementSetAttributeValue")
	Function(&functions.axPerform, services, "AXUIElementPerformAction")
	Function(&functions.axGetPid, services, "AXUIElementGetPid")

	typeIDs, typeError := readTypeIDs(coreFoundation, services)
	if typeError != nil {
		return nil, typeError
	}

	functions.typeIDs = typeIDs

	for _, callBacks := range []struct {
		target *uintptr
		name   string
	}{
		{&functions.dictionaryKeyCallBacks, "kCFTypeDictionaryKeyCallBacks"},
		{&functions.dictionaryValCallBacks, "kCFTypeDictionaryValueCallBacks"},
	} {
		address, lookupError := purego.Dlsym(coreFoundation, callBacks.name)
		if lookupError != nil {
			return nil, fmt.Errorf("%w: %s", ErrSymbolMissing, callBacks.name)
		}

		*callBacks.target = address
	}

	functions.stringClass = objc.ID(objc.GetClass("NSString"))
	functions.selectorAutorelease = objc.RegisterName("autorelease")
	functions.selectorIsKindOfClass = objc.RegisterName("isKindOfClass:")

	return &functions, nil
}

func bindCoreFoundation(functions *coreFunctions, library uintptr) {
	Function(&functions.release, library, "CFRelease")
	Function(&functions.retain, library, "CFRetain")
	Function(&functions.equal, library, "CFEqual")
	Function(&functions.getTypeID, library, "CFGetTypeID")
	Function(&functions.createString, library, "CFStringCreateWithBytes")
	Function(&functions.stringGetLength, library, "CFStringGetLength")
	Function(&functions.stringGetBytes, library, "CFStringGetBytes")
	Function(&functions.numberGetValue, library, "CFNumberGetValue")
	Function(&functions.booleanGetValue, library, "CFBooleanGetValue")
	Function(&functions.arrayGetCount, library, "CFArrayGetCount")
	Function(&functions.arrayGetValueAtIndex, library, "CFArrayGetValueAtIndex")
	Function(&functions.createDictionary, library, "CFDictionaryCreate")
}

// readTypeIDs asks each library for the type ID of every Kind.
func readTypeIDs(coreFoundation, services uintptr) (map[Kind]uintptr, error) {
	typeIDs := map[Kind]uintptr{}

	for kind, source := range map[Kind]struct {
		library uintptr
		name    string
	}{
		KindString:    {coreFoundation, "CFStringGetTypeID"},
		KindNumber:    {coreFoundation, "CFNumberGetTypeID"},
		KindBoolean:   {coreFoundation, "CFBooleanGetTypeID"},
		KindArray:     {coreFoundation, "CFArrayGetTypeID"},
		KindAXElement: {services, "AXUIElementGetTypeID"},
	} {
		var typeID func() uintptr

		Function(&typeID, source.library, source.name)

		typeIDs[kind] = typeID()
		if typeIDs[kind] == 0 {
			return nil, fmt.Errorf("%w: a type ID from %s", ErrSymbolMissing, source.name)
		}
	}

	return typeIDs, nil
}
