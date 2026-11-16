package objcbridge_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The CoreFoundation calls the tests make for themselves, registered with
// Function so that Function is exercised too.
type testFunctions struct {
	retainCount  func(objcbridge.CFTypeRef) int
	createNumber func(allocator uintptr, numberType int, value *int64) objcbridge.CFTypeRef
	createDouble func(allocator uintptr, numberType int, value *float64) objcbridge.CFTypeRef
	createArray  func(allocator uintptr, values *objcbridge.CFTypeRef, count int, callBacks uintptr) objcbridge.CFTypeRef
}

// The number types CFNumberCreate takes, from CFNumber.h.
const (
	numberTypeSInt64  = 4
	numberTypeFloat64 = 6
)

func coreFoundation(t *testing.T) uintptr {
	t.Helper()

	library, loadError := objcbridge.Framework(objcbridge.PathCoreFoundation)
	if loadError != nil {
		t.Fatalf("loading CoreFoundation: %v", loadError)
	}

	return library
}

func loadTestFunctions(t *testing.T) testFunctions {
	t.Helper()

	library := coreFoundation(t)

	var functions testFunctions

	objcbridge.Function(&functions.retainCount, library, "CFGetRetainCount")
	objcbridge.Function(&functions.createNumber, library, "CFNumberCreate")
	objcbridge.Function(&functions.createDouble, library, "CFNumberCreate")
	objcbridge.Function(&functions.createArray, library, "CFArrayCreate")

	return functions
}

func newNumber(t *testing.T, value int64) objcbridge.CFTypeRef {
	t.Helper()

	number := loadTestFunctions(t).createNumber(0, numberTypeSInt64, &value)
	t.Cleanup(func() { objcbridge.Release(number) })

	return number
}

func newString(t *testing.T, text string) objcbridge.CFTypeRef {
	t.Helper()

	created := objcbridge.CFString(text)
	if created == 0 {
		t.Fatalf("CFString(%q) gave nil", text)
	}

	t.Cleanup(func() { objcbridge.Release(created) })

	return created
}

func trueBoolean(t *testing.T) objcbridge.CFTypeRef {
	t.Helper()

	value, constantError := objcbridge.Constant(coreFoundation(t), "kCFBooleanTrue")
	if constantError != nil {
		t.Fatalf("reading kCFBooleanTrue: %v", constantError)
	}

	return value
}

// roundTripTexts are the strings every conversion must carry unchanged: empty,
// plain, accented, outside the Basic Multilingual Plane, and one holding a NUL.
func roundTripTexts() []string {
	return []string{
		"", "Mural", "Café crème à la française", "Ελληνικό κείμενο, кириллица", "Spotify 🎵 🎧",
		"before\x00after", strings.Repeat("ünïcödé ", 200),
	}
}

func TestFramework_GivesTheSameHandle_WhenLoadedTwice(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		objcbridge.PathCoreFoundation, objcbridge.PathFoundation, objcbridge.PathAppKit,
		objcbridge.PathApplicationServices, objcbridge.PathIOKit,
	} {
		first, firstError := objcbridge.Framework(path)
		second, secondError := objcbridge.Framework(path)

		if firstError != nil || secondError != nil {
			t.Fatalf("%s: want it loaded, got %v and %v", path, firstError, secondError)
		}

		if first == 0 || first != second {
			t.Errorf("%s: want one non zero handle, got %#x and %#x", path, first, second)
		}
	}
}

func TestFramework_ReturnsError_WhenPathIsMissing(t *testing.T) {
	t.Parallel()

	handle, loadError := objcbridge.Framework("/System/Library/Frameworks/Missing.framework/Missing")
	if loadError == nil || handle != 0 {
		t.Fatalf("want an error and no handle, got %#x and %v", handle, loadError)
	}
}

func TestFunction_CallsTheNamedFunction(t *testing.T) {
	t.Parallel()

	var typeOfString func() uintptr

	objcbridge.Function(&typeOfString, coreFoundation(t), "CFStringGetTypeID")

	if typeOfString() == 0 {
		t.Fatal("want a non zero type ID from CFStringGetTypeID")
	}
}

func TestFunction_Panics_WhenFunctionIsMissing(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("want a panic for a function the library does not export")
		}
	}()

	var missing func()

	objcbridge.Function(&missing, coreFoundation(t), "CFNoSuchFunctionAnywhere")
}

func TestConstant_ReadsAnExportedCoreFoundationConstant(t *testing.T) {
	t.Parallel()

	value := trueBoolean(t)

	if truth, isBoolean := objcbridge.CFBool(value); !isBoolean || !truth {
		t.Fatalf("want kCFBooleanTrue to read as true, got %v, %v", truth, isBoolean)
	}
}

func TestConstant_ReadsAnExportedAccessibilityConstant(t *testing.T) {
	t.Parallel()

	library, loadError := objcbridge.Framework(objcbridge.PathApplicationServices)
	if loadError != nil {
		t.Fatalf("loading ApplicationServices: %v", loadError)
	}

	value, constantError := objcbridge.Constant(library, "kAXTrustedCheckOptionPrompt")
	if constantError != nil {
		t.Fatalf("reading kAXTrustedCheckOptionPrompt: %v", constantError)
	}

	if text := objcbridge.GoString(value); text != "AXTrustedCheckOptionPrompt" {
		t.Fatalf("want AXTrustedCheckOptionPrompt, got %q", text)
	}
}

func TestConstant_ReturnsErrSymbolMissing_WhenConstantIsMissing(t *testing.T) {
	t.Parallel()

	value, constantError := objcbridge.Constant(coreFoundation(t), "kCFNoSuchConstantAnywhere")
	if !errors.Is(constantError, objcbridge.ErrSymbolMissing) {
		t.Fatalf("want ErrSymbolMissing, got %v", constantError)
	}

	if value != 0 {
		t.Fatalf("want no value, got %#x", value)
	}

	if !strings.Contains(constantError.Error(), "kCFNoSuchConstantAnywhere") {
		t.Fatalf("want the error to name the constant, got %v", constantError)
	}
}

func TestCFString_RoundTripsThroughGoString(t *testing.T) {
	t.Parallel()

	for _, text := range roundTripTexts() {
		if got := objcbridge.GoString(newString(t, text)); got != text {
			t.Errorf("want %q back, got %q", text, got)
		}
	}
}

func TestCFString_ReplacesInvalidUTF8_RatherThanFailing(t *testing.T) {
	t.Parallel()

	if got := objcbridge.GoString(newString(t, "bad \xff byte")); got != "bad \uFFFD byte" {
		t.Fatalf("want the invalid byte replaced, got %q", got)
	}
}

func TestGoString_ReturnsEmpty_WhenNilOrNotAString(t *testing.T) {
	t.Parallel()

	if got := objcbridge.GoString(0); got != "" {
		t.Errorf("want empty for nil, got %q", got)
	}

	if got := objcbridge.GoString(newNumber(t, 7)); got != "" {
		t.Errorf("want empty for a number, got %q", got)
	}
}

func TestNSString_RoundTripsThroughGoStringFromNS(t *testing.T) {
	t.Parallel()

	objcbridge.WithAutoreleasePool(func() {
		for _, text := range roundTripTexts() {
			created := objcbridge.NSString(text)
			if created == 0 {
				t.Errorf("NSString(%q) gave nil", text)

				continue
			}

			if got := objcbridge.GoStringFromNS(created); got != text {
				t.Errorf("want %q back, got %q", text, got)
			}
		}
	})
}

func TestNSString_IsAnNSString(t *testing.T) {
	t.Parallel()

	objcbridge.WithAutoreleasePool(func() {
		created := objcbridge.NSString("Finder")
		stringClass := objc.ID(objc.GetClass("NSString"))

		if !objc.Send[bool](created, objc.RegisterName("isKindOfClass:"), stringClass) {
			t.Fatal("want an NSString")
		}
	})
}

func TestGoStringFromNS_ReturnsEmpty_WhenNilOrNotAString(t *testing.T) {
	t.Parallel()

	if got := objcbridge.GoStringFromNS(0); got != "" {
		t.Errorf("want empty for nil, got %q", got)
	}

	objcbridge.WithAutoreleasePool(func() {
		array := objc.ID(objc.GetClass("NSArray")).Send(objc.RegisterName("array"))

		if got := objcbridge.GoStringFromNS(array); got != "" {
			t.Errorf("want empty for an NSArray, got %q", got)
		}
	})
}

func TestCFInteger_ReadsANumber(t *testing.T) {
	t.Parallel()

	for _, want := range []int64{0, 42, -7, 1 << 40} {
		if got, isInteger := objcbridge.CFInteger(newNumber(t, want)); !isInteger || got != want {
			t.Errorf("want %d, true; got %d, %v", want, got, isInteger)
		}
	}
}

func TestCFInteger_RefusesNilAStringAndAFraction(t *testing.T) {
	t.Parallel()

	fraction := 2.5
	double := loadTestFunctions(t).createDouble(0, numberTypeFloat64, &fraction)

	defer objcbridge.Release(double)

	for name, value := range map[string]objcbridge.CFTypeRef{
		"nil": 0, "string": newString(t, "42"), "fraction": double, "boolean": trueBoolean(t),
	} {
		if _, isInteger := objcbridge.CFInteger(value); isInteger {
			t.Errorf("%s: want it refused as an integer", name)
		}
	}
}

func TestCFBool_ReadsBothBooleans(t *testing.T) {
	t.Parallel()

	falseValue, constantError := objcbridge.Constant(coreFoundation(t), "kCFBooleanFalse")
	if constantError != nil {
		t.Fatalf("reading kCFBooleanFalse: %v", constantError)
	}

	if truth, isBoolean := objcbridge.CFBool(falseValue); !isBoolean || truth {
		t.Errorf("want false, true; got %v, %v", truth, isBoolean)
	}

	if truth, isBoolean := objcbridge.CFBool(trueBoolean(t)); !isBoolean || !truth {
		t.Errorf("want true, true; got %v, %v", truth, isBoolean)
	}
}

func TestCFBool_RefusesNilAndANumber(t *testing.T) {
	t.Parallel()

	if _, isBoolean := objcbridge.CFBool(0); isBoolean {
		t.Error("want nil refused")
	}

	if _, isBoolean := objcbridge.CFBool(newNumber(t, 1)); isBoolean {
		t.Error("want a number refused")
	}
}

func TestCFArrayItems_ListsEveryItemInOrder(t *testing.T) {
	t.Parallel()

	items := []objcbridge.CFTypeRef{newString(t, "first"), newString(t, "second"), newNumber(t, 3)}

	callBacks, symbolError := purego.Dlsym(coreFoundation(t), "kCFTypeArrayCallBacks")
	if symbolError != nil {
		t.Fatalf("finding kCFTypeArrayCallBacks: %v", symbolError)
	}

	array := loadTestFunctions(t).createArray(0, &items[0], len(items), callBacks)

	defer objcbridge.Release(array)

	got := objcbridge.CFArrayItems(array)
	if len(got) != len(items) {
		t.Fatalf("want %d items, got %d", len(items), len(got))
	}

	for index, item := range items {
		if !objcbridge.Equal(got[index], item) {
			t.Errorf("item %d: want %#x, got %#x", index, item, got[index])
		}
	}
}

func TestCFArrayItems_ReturnsNothing_WhenNilOrNotAnArray(t *testing.T) {
	t.Parallel()

	if got := objcbridge.CFArrayItems(0); len(got) != 0 {
		t.Errorf("want nothing for nil, got %v", got)
	}

	if got := objcbridge.CFArrayItems(newString(t, "not an array")); len(got) != 0 {
		t.Errorf("want nothing for a string, got %v", got)
	}
}

func TestCFDictionary_HoldsEveryPair(t *testing.T) {
	t.Parallel()

	prompt, promptValue := newString(t, "AXTrustedCheckOptionPrompt"), trueBoolean(t)
	count, countValue := newString(t, "count"), newNumber(t, 12)

	dictionary := objcbridge.CFDictionary(map[objcbridge.CFTypeRef]objcbridge.CFTypeRef{
		prompt: promptValue, count: countValue,
	})

	defer objcbridge.Release(dictionary)

	var (
		getCount func(objcbridge.CFTypeRef) int
		getValue func(dictionary, key objcbridge.CFTypeRef) objcbridge.CFTypeRef
	)

	objcbridge.Function(&getCount, coreFoundation(t), "CFDictionaryGetCount")
	objcbridge.Function(&getValue, coreFoundation(t), "CFDictionaryGetValue")

	if got := getCount(dictionary); got != 2 {
		t.Fatalf("want 2 pairs, got %d", got)
	}

	if truth, _ := objcbridge.CFBool(getValue(dictionary, newString(t, "AXTrustedCheckOptionPrompt"))); !truth {
		t.Error("want the prompt key, looked up by an equal string, to give true")
	}

	if number, _ := objcbridge.CFInteger(getValue(dictionary, count)); number != 12 {
		t.Errorf("want the count key to give 12, got %d", number)
	}
}

func TestCFDictionary_RetainsItsKeysAndValues(t *testing.T) {
	t.Parallel()

	functions := loadTestFunctions(t)
	key, value := newString(t, strings.Repeat("clé ", 20)), newString(t, strings.Repeat("valeur ", 20))
	keyBefore, valueBefore := functions.retainCount(key), functions.retainCount(value)

	dictionary := objcbridge.CFDictionary(map[objcbridge.CFTypeRef]objcbridge.CFTypeRef{key: value})
	if dictionary == 0 {
		t.Fatal("want a dictionary")
	}

	if functions.retainCount(value) != valueBefore+1 || functions.retainCount(key) <= keyBefore {
		t.Errorf("want the value retained by the dictionary, got key %d to %d and value %d to %d",
			keyBefore, functions.retainCount(key), valueBefore, functions.retainCount(value))
	}

	objcbridge.Release(dictionary)

	if functions.retainCount(value) != valueBefore {
		t.Errorf("want the value released with the dictionary, got %d after %d", functions.retainCount(value), valueBefore)
	}
}

func TestCFDictionary_GivesAnEmptyDictionary_WhenGivenNoPairs(t *testing.T) {
	t.Parallel()

	dictionary := objcbridge.CFDictionary(nil)
	if dictionary == 0 {
		t.Fatal("want an empty dictionary, got nil")
	}

	defer objcbridge.Release(dictionary)

	var getCount func(objcbridge.CFTypeRef) int

	objcbridge.Function(&getCount, coreFoundation(t), "CFDictionaryGetCount")

	if got := getCount(dictionary); got != 0 {
		t.Fatalf("want no pairs, got %d", got)
	}
}

func TestIsType_RecognisesEachKind_AndRefusesTheOthers(t *testing.T) {
	t.Parallel()

	element := objcbridge.AXSystemWide()

	defer objcbridge.Release(element)

	values := map[objcbridge.Kind]objcbridge.CFTypeRef{
		objcbridge.KindString:    newString(t, "text"),
		objcbridge.KindNumber:    newNumber(t, 1),
		objcbridge.KindBoolean:   trueBoolean(t),
		objcbridge.KindArray:     emptyArray(t),
		objcbridge.KindAXElement: element,
	}

	for kind, value := range values {
		for otherKind := range values {
			if got := objcbridge.IsType(value, otherKind); got != (kind == otherKind) {
				t.Errorf("value of kind %v asked as kind %v: want %v, got %v", kind, otherKind, kind == otherKind, got)
			}
		}

		if objcbridge.IsType(0, kind) {
			t.Errorf("want nil refused as kind %v", kind)
		}
	}
}

func emptyArray(t *testing.T) objcbridge.CFTypeRef {
	t.Helper()

	array := loadTestFunctions(t).createArray(0, nil, 0, 0)
	t.Cleanup(func() { objcbridge.Release(array) })

	return array
}

func TestReleaseAndRetain_ChangeTheRetainCountByOne(t *testing.T) {
	t.Parallel()

	functions := loadTestFunctions(t)
	created := objcbridge.CFString(strings.Repeat("retenu ", 30))
	before := functions.retainCount(created)

	if retained := objcbridge.Retain(created); retained != created {
		t.Fatalf("want Retain to give back %#x, got %#x", created, retained)
	}

	if got := functions.retainCount(created); got != before+1 {
		t.Fatalf("want %d after Retain, got %d", before+1, got)
	}

	objcbridge.Release(created)

	if got := functions.retainCount(created); got != before {
		t.Fatalf("want %d after Release, got %d", before, got)
	}

	objcbridge.Release(created)
}

func TestReleaseAndRetain_IgnoreNil(t *testing.T) {
	t.Parallel()

	objcbridge.Release(0)

	if retained := objcbridge.Retain(0); retained != 0 {
		t.Fatalf("want nil back, got %#x", retained)
	}
}

func TestEqual_ComparesByValue(t *testing.T) {
	t.Parallel()

	first, same, other := newString(t, "Notes"), newString(t, "Notes"), newString(t, "Mail")

	if !objcbridge.Equal(first, same) {
		t.Error("want two strings holding the same text equal")
	}

	if objcbridge.Equal(first, other) {
		t.Error("want different text unequal")
	}

	if objcbridge.Equal(first, 0) || objcbridge.Equal(0, first) {
		t.Error("want a value unequal to nil")
	}

	if !objcbridge.Equal(0, 0) {
		t.Error("want nil equal to nil")
	}
}

func TestWithAutoreleasePool_DrainsWhatWasAutoreleased(t *testing.T) {
	t.Parallel()

	functions := loadTestFunctions(t)
	created := objcbridge.CFString(strings.Repeat("libéré ", 30))
	before := functions.retainCount(created)
	autorelease := objc.RegisterName("autorelease")

	objcbridge.WithAutoreleasePool(func() {
		objcbridge.Retain(created)
		objc.ID(created).Send(autorelease)

		if got := functions.retainCount(created); got != before+1 {
			t.Errorf("want %d inside the pool, got %d", before+1, got)
		}
	})

	if got := functions.retainCount(created); got != before {
		t.Errorf("want the pool to drain back to %d, got %d", before, got)
	}

	objcbridge.Release(created)
}

func TestWithAutoreleasePool_RunsTheWorkOnce(t *testing.T) {
	t.Parallel()

	runs := 0

	objcbridge.WithAutoreleasePool(func() { runs++ })

	if runs != 1 {
		t.Fatalf("want one run, got %d", runs)
	}
}

func TestAXSystemWide_GivesAnAccessibilityElement(t *testing.T) {
	t.Parallel()

	element := objcbridge.AXSystemWide()
	if element == 0 {
		t.Fatal("want an element")
	}

	defer objcbridge.Release(element)

	if !objcbridge.IsType(element, objcbridge.KindAXElement) {
		t.Fatal("want an AXUIElement")
	}
}

func TestAXApplication_GivesAnElementForTheProcess(t *testing.T) {
	t.Parallel()

	element := objcbridge.AXApplication(os.Getpid())
	if element == 0 {
		t.Fatal("want an element")
	}

	defer objcbridge.Release(element)

	processID, readError := objcbridge.AXProcessID(element)
	if readError != nil {
		t.Fatalf("reading its process ID: %v", readError)
	}

	if processID != os.Getpid() {
		t.Fatalf("want process %d, got %d", os.Getpid(), processID)
	}
}

func TestAXProcessID_ReturnsAnAXError_WhenElementIsNil(t *testing.T) {
	t.Parallel()

	_, readError := objcbridge.AXProcessID(0)

	var axError objcbridge.AXError
	if !errors.As(readError, &axError) || axError != objcbridge.ErrAXIllegalArgument {
		t.Fatalf("want AXErrorIllegalArgument, got %v", readError)
	}
}

func TestAXCopy_ReturnsAnAXError_WhenAttributeIsNonsense(t *testing.T) {
	t.Parallel()

	element := objcbridge.AXSystemWide()

	defer objcbridge.Release(element)

	value, copyError := objcbridge.AXCopy(element, "TFBNoSuchAttribute")
	if copyError == nil {
		objcbridge.Release(value)
		t.Fatal("want an error for an attribute nothing has")
	}

	var axError objcbridge.AXError
	if !errors.As(copyError, &axError) || axError == 0 {
		t.Fatalf("want a non zero AXError, got %v", copyError)
	}

	if !strings.Contains(copyError.Error(), "TFBNoSuchAttribute") {
		t.Fatalf("want the error to name the attribute, got %v", copyError)
	}
}

func TestAXCopy_ReturnsAnAXError_WhenElementIsNil(t *testing.T) {
	t.Parallel()

	_, copyError := objcbridge.AXCopy(0, "AXRole")

	var axError objcbridge.AXError
	if !errors.As(copyError, &axError) || axError != objcbridge.ErrAXIllegalArgument {
		t.Fatalf("want AXErrorIllegalArgument, got %v", copyError)
	}
}

// The setting and action tests aim at this test process's own element, which
// has no windows or attributes to change, so nothing on the desktop moves.
func TestAXSet_ReturnsAnAXError_WhenAttributeIsNonsense(t *testing.T) {
	t.Parallel()

	element := objcbridge.AXApplication(os.Getpid())

	defer objcbridge.Release(element)

	setError := objcbridge.AXSet(element, "TFBNoSuchAttribute", trueBoolean(t))

	var axError objcbridge.AXError
	if !errors.As(setError, &axError) || axError == 0 {
		t.Fatalf("want a non zero AXError, got %v", setError)
	}

	nilError := objcbridge.AXSet(0, "AXFrontmost", trueBoolean(t))
	if !errors.As(nilError, &axError) || axError != objcbridge.ErrAXIllegalArgument {
		t.Fatalf("want AXErrorIllegalArgument for a nil element, got %v", nilError)
	}
}

func TestAXPerform_ReturnsAnAXError_WhenActionIsNonsense(t *testing.T) {
	t.Parallel()

	element := objcbridge.AXApplication(os.Getpid())

	defer objcbridge.Release(element)

	performError := objcbridge.AXPerform(element, "TFBNoSuchAction")

	var axError objcbridge.AXError
	if !errors.As(performError, &axError) || axError == 0 {
		t.Fatalf("want a non zero AXError, got %v", performError)
	}

	nilError := objcbridge.AXPerform(0, "AXRaise")
	if !errors.As(nilError, &axError) || axError != objcbridge.ErrAXIllegalArgument {
		t.Fatalf("want AXErrorIllegalArgument for a nil element, got %v", nilError)
	}
}

func TestAXError_NamesItsCode(t *testing.T) {
	t.Parallel()

	if text := objcbridge.ErrAXAPIDisabled.Error(); !strings.Contains(text, "-25211") {
		t.Fatalf("want the code in the message, got %q", text)
	}
}

func TestNSGeometry_MatchesAppKitsLayout(t *testing.T) {
	t.Parallel()

	_, loadError := objcbridge.Framework(objcbridge.PathFoundation)
	if loadError != nil {
		t.Fatalf("loading Foundation: %v", loadError)
	}

	rect := objcbridge.NSRect{
		Origin: objcbridge.NSPoint{X: 1, Y: 2},
		Size:   objcbridge.NSSize{Width: 3, Height: 4},
	}

	objcbridge.WithAutoreleasePool(func() {
		value := objc.ID(objc.GetClass("NSValue")).Send(objc.RegisterName("valueWithRect:"), rect)
		back := objc.Send[objcbridge.NSRect](value, objc.RegisterName("rectValue"))

		if back != rect {
			t.Fatalf("want %+v back through NSValue, got %+v", rect, back)
		}
	})
}
