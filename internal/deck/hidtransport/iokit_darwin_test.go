package hidtransport //nolint:testpackage // drives the IOKit driver itself, which is unexported

import (
	"cmp"
	"errors"
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// The IOKit driver's own calls, on the Mac. These only list devices and
// refuse paths no device has, so they need no lock and never open the deck.
// The driver contract (driver_contract_test.go) opens and drives the deck
// through this driver under THE_FRONT_BUTTONS_EXPERIMENT.

const (
	iokitAbsentVendor = 0xFFFE
	anyVendor         = 0
)

func TestIOKit_IsBuiltInAsThePuregoDriver(t *testing.T) {
	t.Parallel()

	if _, isIOKit := builtDrivers[DriverPurego].(*iokitDriver); !isIOKit {
		t.Errorf("want the purego driver to be the IOKit driver, got %T", builtDrivers[DriverPurego])
	}
}

func TestIOKit_FindsEveryFunctionItCalls(t *testing.T) {
	t.Parallel()

	functions, loadError := loadIOKit()
	if loadError != nil || functions == nil {
		t.Fatalf("want IOKit and Core Foundation to hold every function, got %v", loadError)
	}
}

func listWith(t *testing.T, listing driver, vendorID uint16) []deck.Connection {
	t.Helper()

	var listed []deck.Connection

	listError := withDriverLock(func() error {
		var enumerateError error

		listed, enumerateError = listing.enumerate(vendorID)

		return enumerateError
	})
	if listError != nil {
		t.Fatalf("listing vendor %#04x: %v", vendorID, listError)
	}

	return listed
}

func sortedConnections(connections []deck.Connection) []deck.Connection {
	return slices.SortedFunc(slices.Values(connections), func(first, second deck.Connection) int {
		return cmp.Or(
			cmp.Compare(first.Path, second.Path),
			cmp.Compare(first.VendorID, second.VendorID),
			cmp.Compare(first.ProductID, second.ProductID),
		)
	})
}

func TestIOKit_ListsNothingForAVendorWithNothingAttached(t *testing.T) {
	t.Parallel()

	if listed := listWith(t, &iokitDriver{}, iokitAbsentVendor); len(listed) != 0 {
		t.Errorf("want nothing, got %v", listed)
	}
}

// TestIOKit_ListsPathsInHIDAPIsForm wants every device listed, of any vendor,
// to have a DevSrvsID path that reads back, or none, as hidapi gives.
func TestIOKit_ListsPathsInHIDAPIsForm(t *testing.T) {
	t.Parallel()

	listed := listWith(t, &iokitDriver{}, anyVendor)
	if len(listed) == 0 {
		t.Skip("no HID device is attached to list")
	}

	for _, connection := range listed {
		if connection.Path == "" {
			continue
		}

		entryID, parseError := parseIOKitPath(connection.Path)
		if parseError != nil || iokitPath(entryID) != connection.Path {
			t.Errorf("%+v: want a DevSrvsID path, got %v", connection, parseError)
		}
	}
}

// TestIOKit_ListsTheSameDevicesAsHIDAPI is the parity test: the same
// connections and paths through both drivers, for the deck's vendor, for
// every vendor and for a vendor with nothing attached. It only lists.
func TestIOKit_ListsTheSameDevicesAsHIDAPI(t *testing.T) {
	t.Parallel()

	hidapi, built := builtDrivers[DriverHIDAPI]
	if !built {
		t.Skip("the hidapi driver is not built in without cgo")
	}

	for _, vendorID := range []uint16{elgatoVendorID, anyVendor, iokitAbsentVendor} {
		fromHIDAPI := sortedConnections(listWith(t, hidapi, vendorID))
		fromIOKit := sortedConnections(listWith(t, &iokitDriver{}, vendorID))

		if !slices.Equal(fromHIDAPI, fromIOKit) {
			t.Errorf("vendor %#04x:\nhidapi lists %v\nIOKit lists  %v", vendorID, fromHIDAPI, fromIOKit)
		}

		devices := map[string]bool{}
		for _, connection := range fromIOKit {
			devices[connection.Path] = true
		}

		t.Logf("vendor %#04x: %d listings of %d devices through each driver", vendorID, len(fromIOKit), len(devices))
	}
}

func TestIOKit_RefusesToOpenAPathNoDeviceHas(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]error{
		"no-such-device-path":                  errNotIOKitPath,
		"IOService:/AppleACPIPlatformExpert":   errNotIOKitPath,
		"DevSrvsID:18446744073709551615":       errNoService,
		"DevSrvsID:" + "123456789012345678901": errNotIOKitPath,
	} {
		var opened handle

		openError := withDriverLock(func() error {
			var pathError error

			opened, pathError = (&iokitDriver{}).open(path)

			return pathError
		})
		if opened != nil {
			_ = withDriverLock(opened.close)
		}

		if !errors.Is(openError, want) {
			t.Errorf("%q: want %v, got %v", path, want, openError)
		}
	}
}

// TestIOKit_SeizesTheDeviceAsHIDAPIDoes wants devices opened with
// kIOHIDOptionsTypeSeizeDevice (1, from IOHIDKeys.h), the option go-hid's
// hidapi opens with, so the two drivers are compared like with like.
func TestIOKit_SeizesTheDeviceAsHIDAPIDoes(t *testing.T) {
	t.Parallel()

	const seizeDevice = 1

	if deviceOpenOptions != seizeDevice {
		t.Errorf("want kIOHIDOptionsTypeSeizeDevice (1), got %d", deviceOpenOptions)
	}
}
