package hidtransport

import (
	"errors"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The macOS HID driver calls IOKit and Core Foundation through purego,
// needing no C compiler, the way hidapi's macOS backend does. Its decisions
// are in iokitcore.go, which every system builds; this file holds only the
// calls for listing, opening, reports out, feature reports and closing, and
// iokit_input_darwin.go those for input reports and removal.

func init() { builtDrivers[DriverPurego] = &iokitDriver{} } //nolint:gochecknoinits // registers the driver

// The IOKit and Core Foundation values the driver passes:
// kIOHIDOptionsTypeNone for the HID manager; kIOHIDOptionsTypeSeizeDevice,
// which opens a device for this process alone, as hidapi's macOS backend
// opens it by default (hid_init turns its open exclusive option on), so both
// drivers hold the deck the same way; kIOMainPortDefault; kCFNumberShortType,
// which hidapi gives the vendor ID in its matching dictionary; and
// kCFAllocatorDefault.
const (
	iokitOptionsNone    = 0
	iokitOptionsSeize   = 1
	deviceOpenOptions   = iokitOptionsSeize
	iokitMainPort       = 0
	numberTypeShort     = 8
	allocatorDefault    = 0
	kernelReturnSuccess = 0
)

// The property keys the driver reads, from IOHIDKeys.h.
const (
	keyVendorID         = "VendorID"
	keyProductID        = "ProductID"
	keyPrimaryUsagePage = "PrimaryUsagePage"
	keyPrimaryUsage     = "PrimaryUsage"
	keyUsagePairs       = "DeviceUsagePairs"
	keyUsagePage        = "DeviceUsagePage"
	keyUsage            = "DeviceUsage"
	keyMaxInputSize     = "MaxInputReportSize"
	keyMaxOutputSize    = "MaxOutputReportSize"
	keyMaxFeatureSize   = "MaxFeatureReportSize"
)

var (
	errManagerNotCreated = errors.New("IOHIDManagerCreate gave no manager")
	errNoService         = errors.New("no IOHIDDevice service has that registry entry ID")
	errDeviceNotCreated  = errors.New("IOHIDDeviceCreate gave no device")
)

// cfRef is a Core Foundation object, as objcbridge passes them.
type cfRef = objcbridge.CFTypeRef

// iokitFunctions are the IOKit and Core Foundation functions the driver
// calls, found once per process (see loadIOKit), and the property keys it
// reads, made once and kept.
type iokitFunctions struct {
	inputFunctions

	managerCreate            func(allocator uintptr, options uint32) cfRef
	managerSetDeviceMatching func(manager, matching cfRef)
	managerCopyDevices       func(manager cfRef) cfRef
	deviceGetProperty        func(device, key cfRef) cfRef
	deviceGetService         func(device cfRef) uint32
	registryEntryID          func(entry uint32, entryID *uint64) int32
	registryEntryIDMatching  func(entryID uint64) cfRef
	serviceGetMatching       func(mainPort uint32, matching cfRef) uint32
	objectRelease            func(object uint32) int32
	deviceCreate             func(allocator uintptr, service uint32) cfRef
	deviceOpen               func(device cfRef, options uint32) int32
	deviceClose              func(device cfRef, options uint32) int32
	deviceSetReport          func(device cfRef, reportType iokitReportType, reportID int, report *byte, length int) int32
	deviceGetReport          func(device cfRef, reportType iokitReportType, reportID int, report *byte, length *int) int32

	numberCreate        func(allocator uintptr, numberType int, value *uint16) cfRef
	setGetCount         func(set cfRef) int
	setGetValues        func(set cfRef, values *cfRef)
	getTypeID           func(ref cfRef) uintptr
	dictionaryGetTypeID func() uintptr
	dictionaryGetValue  func(dictionary, key cfRef) cfRef

	keys map[string]cfRef
}

// binding is one C function the driver calls: where purego points it, the
// library holding it and its name.
type binding struct {
	target  any
	library uintptr
	name    string
}

func (functions *iokitFunctions) listingBindings(iokit, coreFoundation uintptr) []binding {
	return []binding{
		{&functions.managerCreate, iokit, "IOHIDManagerCreate"},
		{&functions.managerSetDeviceMatching, iokit, "IOHIDManagerSetDeviceMatching"},
		{&functions.managerCopyDevices, iokit, "IOHIDManagerCopyDevices"},
		{&functions.deviceGetProperty, iokit, "IOHIDDeviceGetProperty"},
		{&functions.deviceGetService, iokit, "IOHIDDeviceGetService"},
		{&functions.registryEntryID, iokit, "IORegistryEntryGetRegistryEntryID"},
		{&functions.numberCreate, coreFoundation, "CFNumberCreate"},
		{&functions.setGetCount, coreFoundation, "CFSetGetCount"},
		{&functions.setGetValues, coreFoundation, "CFSetGetValues"},
		{&functions.getTypeID, coreFoundation, "CFGetTypeID"},
		{&functions.dictionaryGetTypeID, coreFoundation, "CFDictionaryGetTypeID"},
		{&functions.dictionaryGetValue, coreFoundation, "CFDictionaryGetValue"},
	}
}

func (functions *iokitFunctions) deviceBindings(iokit uintptr) []binding {
	return []binding{
		{&functions.registryEntryIDMatching, iokit, "IORegistryEntryIDMatching"},
		{&functions.serviceGetMatching, iokit, "IOServiceGetMatchingService"},
		{&functions.objectRelease, iokit, "IOObjectRelease"},
		{&functions.deviceCreate, iokit, "IOHIDDeviceCreate"},
		{&functions.deviceOpen, iokit, "IOHIDDeviceOpen"},
		{&functions.deviceClose, iokit, "IOHIDDeviceClose"},
		{&functions.deviceSetReport, iokit, "IOHIDDeviceSetReport"},
		{&functions.deviceGetReport, iokit, "IOHIDDeviceGetReport"},
	}
}

// loadIOKit finds every function the driver calls, once per process, in this
// one place, through objcbridge.Bind: a missing library or function, or a
// panic finding one, makes the driver unavailable (an error wrapping
// objcbridge.ErrBindingFailed) instead of ending the daemon; once found,
// calling a function cannot panic for want of it.
//
//nolint:gochecknoglobals // the system's libraries are process wide
var loadIOKit = objcbridge.Bind(func() (*iokitFunctions, error) {
	iokit, iokitError := objcbridge.Framework(objcbridge.PathIOKit)
	if iokitError != nil {
		return nil, fmt.Errorf("opening IOKit: %w", iokitError)
	}

	coreFoundation, coreError := objcbridge.Framework(objcbridge.PathCoreFoundation)
	if coreError != nil {
		return nil, fmt.Errorf("opening Core Foundation: %w", coreError)
	}

	functions := new(iokitFunctions)
	functions.keys = map[string]cfRef{}

	bindings := append(functions.listingBindings(iokit, coreFoundation), functions.deviceBindings(iokit)...)
	bindings = append(bindings, functions.inputBindings(iokit, coreFoundation)...)

	for _, each := range bindings {
		objcbridge.Function(each.target, each.library, each.name)
	}

	inputError := functions.findInputValues(coreFoundation)
	if inputError != nil {
		return nil, inputError
	}

	// The keys live as long as the process; CFString also proves objcbridge's
	// own bindings load, here where a failure is recovered.
	for _, key := range []string{
		keyVendorID, keyProductID, keyPrimaryUsagePage, keyPrimaryUsage, keyUsagePairs,
		keyUsagePage, keyUsage, keyMaxInputSize, keyMaxOutputSize, keyMaxFeatureSize,
	} {
		functions.keys[key] = objcbridge.CFString(key)
	}

	return functions, nil
})

// integerProperty reads a device's numeric property, 0 when it is missing or
// not a number, as hidapi's get_int_property gives.
func (functions *iokitFunctions) integerProperty(device cfRef, key string) int64 {
	value, isInteger := objcbridge.CFInteger(functions.deviceGetProperty(device, functions.keys[key]))
	if !isInteger {
		return 0
	}

	return value
}

// integerEntry reads a number from a dictionary, reporting whether it was
// there and a number.
func (functions *iokitFunctions) integerEntry(dictionary cfRef, key string) (int64, bool) {
	return objcbridge.CFInteger(functions.dictionaryGetValue(dictionary, functions.keys[key]))
}

// usagePairs reads DeviceUsagePairs, keeping the dictionaries that hold both
// a usage page and a usage, as hidapi does.
func (functions *iokitFunctions) usagePairs(device cfRef) []usagePair {
	var pairs []usagePair

	for _, entry := range objcbridge.CFArrayItems(functions.deviceGetProperty(device, functions.keys[keyUsagePairs])) {
		if entry == 0 || functions.getTypeID(entry) != functions.dictionaryGetTypeID() {
			continue
		}

		page, hasPage := functions.integerEntry(entry, keyUsagePage)
		usage, hasUsage := functions.integerEntry(entry, keyUsage)

		if hasPage && hasUsage {
			pairs = append(pairs, usagePair{page: page, usage: usage})
		}
	}

	return pairs
}

// listing reads what enumeration needs from one device: its vendor, product
// and usages, and its path from the registry entry ID of its service.
func (functions *iokitFunctions) listing(device cfRef) iokitListing {
	path := ""

	if service := functions.deviceGetService(device); service != 0 {
		var entryID uint64

		if functions.registryEntryID(service, &entryID) == kernelReturnSuccess {
			path = iokitPath(entryID)
		}
	}

	return iokitListing{
		vendorID:  lowSixteenBits(functions.integerProperty(device, keyVendorID)),
		productID: lowSixteenBits(functions.integerProperty(device, keyProductID)),
		path:      path,
		primary: usagePair{
			page:  functions.integerProperty(device, keyPrimaryUsagePage),
			usage: functions.integerProperty(device, keyPrimaryUsage),
		},
		pairs: functions.usagePairs(device),
	}
}

// vendorMatching is the matching dictionary for one vendor's devices, owned
// by the caller, or nil for vendor zero, which hidapi reads as every device.
func (functions *iokitFunctions) vendorMatching(vendorID uint16) cfRef {
	if vendorID == 0 {
		return 0
	}

	vendor := functions.numberCreate(allocatorDefault, numberTypeShort, &vendorID)
	defer objcbridge.Release(vendor)

	return objcbridge.CFDictionary(map[cfRef]cfRef{functions.keys[keyVendorID]: vendor})
}

// devices lists the IOHIDDevices in a set IOHIDManagerCopyDevices gave.
func (functions *iokitFunctions) devices(set cfRef) []cfRef {
	count := functions.setGetCount(set)
	if count <= 0 {
		return nil
	}

	devices := make([]cfRef, count)
	functions.setGetValues(set, &devices[0])

	return devices
}

// iokitDriver lists and opens HID devices through IOKit.
type iokitDriver struct{}

// enumerate lists one vendor's devices as hidapi's macOS backend does:
// IOHIDManagerCreate, IOHIDManagerSetDeviceMatching with the vendor ID,
// IOHIDManagerCopyDevices, then each device's properties and the registry
// entry ID of IOHIDDeviceGetService. Listing needs no IOHIDManagerOpen. A
// manager is made for each listing and released after it, so no device set
// outlives the call.
func (*iokitDriver) enumerate(vendorID uint16) ([]deck.Connection, error) {
	functions, loadError := loadIOKit()
	if loadError != nil {
		return nil, loadError
	}

	manager := functions.managerCreate(allocatorDefault, iokitOptionsNone)
	if manager == 0 {
		return nil, errManagerNotCreated
	}
	defer objcbridge.Release(manager)

	matching := functions.vendorMatching(vendorID)
	functions.managerSetDeviceMatching(manager, matching)
	objcbridge.Release(matching)

	set := functions.managerCopyDevices(manager)
	if set == 0 {
		return nil, nil
	}
	defer objcbridge.Release(set)

	var found []deck.Connection

	for _, device := range functions.devices(set) {
		if device != 0 {
			found = append(found, functions.listing(device).connections()...)
		}
	}

	return found, nil
}

// open opens a listed path as hidapi's macOS backend does:
// IORegistryEntryIDMatching, IOServiceGetMatchingService, IOHIDDeviceCreate
// and IOHIDDeviceOpen, seizing the device as hidapi does, then reads its report
// lengths and starts its run loop thread for input reports and removal.
//
//nolint:ireturn // the driver contract returns a handle interface
func (*iokitDriver) open(path string) (handle, error) {
	functions, loadError := loadIOKit()
	if loadError != nil {
		return nil, loadError
	}

	entryID, pathError := parseIOKitPath(path)
	if pathError != nil {
		return nil, pathError
	}

	// IOServiceGetMatchingService takes over the matching dictionary.
	service := functions.serviceGetMatching(iokitMainPort, functions.registryEntryIDMatching(entryID))
	if service == 0 {
		return nil, fmt.Errorf("%w: %s", errNoService, path)
	}
	defer functions.objectRelease(service)

	device := functions.deviceCreate(allocatorDefault, service)
	if device == 0 {
		return nil, fmt.Errorf("%w: %s", errDeviceNotCreated, path)
	}

	openError := checkIOReturn("IOHIDDeviceOpen", functions.deviceOpen(device, deviceOpenOptions))
	if openError != nil {
		objcbridge.Release(device)

		return nil, openError
	}

	lengths := reportLengths{
		input:   int(functions.integerProperty(device, keyMaxInputSize)),
		output:  int(functions.integerProperty(device, keyMaxOutputSize)),
		feature: int(functions.integerProperty(device, keyMaxFeatureSize)),
	}

	return functions.startDevice(device, lengths), nil
}

// iokitDevice is the IOKit calls for one open device.
type iokitDevice struct {
	functions *iokitFunctions
	device    cfRef
	handle    *iokitHandle
	// context names the device to its callbacks (see inputRegistry).
	context uintptr
	// input is the device's run loop thread, nil once stopped.
	input *inputThread
}

// bytesAt is where a report's bytes start for IOKit, nil for none.
func bytesAt(report []byte) *byte {
	if len(report) == 0 {
		return nil
	}

	return &report[0]
}

func (opened *iokitDevice) setReport(reportType iokitReportType, reportID byte, payload []byte) int32 {
	return opened.functions.deviceSetReport(opened.device, reportType, int(reportID), bytesAt(payload), len(payload))
}

func (opened *iokitDevice) getReport(reportType iokitReportType, reportID byte, buffer []byte) (int, int32) {
	filled := len(buffer)
	result := opened.functions.deviceGetReport(opened.device, reportType, int(reportID), bytesAt(buffer), &filled)

	return filled, result
}

// release ends the device's run loop thread, then closes the device with
// IOHIDDeviceClose with the options it was opened with, as hidapi does on
// every macOS since 10.10 even after removal, and releases it.
func (opened *iokitDevice) release() int32 {
	opened.stopInput()

	result := opened.functions.deviceClose(opened.device, deviceOpenOptions)
	objcbridge.Release(opened.device)

	return result
}
