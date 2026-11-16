package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The ApplicationIcons on Windows asks the shell for an item's picture with
// SHCreateItemFromParsingName and IShellItemImageFactory::GetImage, which take
// an Apps folder entry and a program's path alike, then reads the bitmap's
// pixels with GetDIBits. appicons.go decides which item to ask for and turns
// the pixels into an image. Not yet run on a Windows machine: the author is
// testing on macOS only for now.

// The Win32 values the icon reader uses, from ShObjIdl_core.h and WinGDI.h:
// GetImage's request for an icon rather than a thumbnail, a bitmap's header
// size, one plane of 32 bits uncompressed, and colours given as RGB values.
const (
	shellImageIconOnly   = 0x4
	bitmapHeaderBytes    = 40
	bitmapPlanes         = 1
	bitmapBitsPerPixel   = 32
	bitmapUncompressed   = 0
	bitmapRGBColours     = 0
	imageFactoryGetImage = 3
	comRelease           = 2
	comFailureBit        = 0x80000000
)

// iidShellItemImageFactory is IShellItemImageFactory's interface ID.
//
//nolint:gochecknoglobals,mnd // a constant GUID, which Go cannot declare as a constant
var iidShellItemImageFactory = windows.GUID{
	Data1: 0xBCC18B79, Data2: 0xBA16, Data3: 0x442F,
	Data4: [8]byte{0x80, 0xC4, 0x8A, 0x59, 0xC3, 0x0C, 0x46, 0x3B},
}

//nolint:gochecknoglobals // the lazily loaded system calls Go's Windows package lacks
var (
	procCreateItemFromParsingName = windows.NewLazySystemDLL("shell32.dll").NewProc("SHCreateItemFromParsingName")
	procGetObject                 = windows.NewLazySystemDLL("gdi32.dll").NewProc("GetObjectW")
	procGetDIBits                 = windows.NewLazySystemDLL("gdi32.dll").NewProc("GetDIBits")
	procDeleteObject              = windows.NewLazySystemDLL("gdi32.dll").NewProc("DeleteObject")
	procGetDC                     = windows.NewLazySystemDLL("user32.dll").NewProc("GetDC")
	procReleaseDC                 = windows.NewLazySystemDLL("user32.dll").NewProc("ReleaseDC")
)

// The ways reading an icon's bitmap fails.
var (
	errBitmapNotDescribed = errors.New("windows did not describe the icon's bitmap")
	errBitmapNotRead      = errors.New("windows did not hand over the icon's pixels")
)

// bitmapDescription is GDI's BITMAP, which GetObjectW fills in.
// Only its size is read; the rest holds the struct's layout.
type bitmapDescription struct {
	_      int32
	width  int32
	height int32
	_      int32
	_      uint16
	_      uint16
	_      uintptr
}

// bitmapHeader is GDI's BITMAPINFOHEADER, the whole of the BITMAPINFO
// GetDIBits needs for 32 bit pixels with no colour table.
type bitmapHeader struct {
	size            uint32
	width           int32
	height          int32
	planes          uint16
	bitsPerPixel    uint16
	compression     uint32
	imageSize       uint32
	xPixelsPerMetre int32
	yPixelsPerMetre int32
	coloursUsed     uint32
	coloursNeeded   uint32
}

// imageFactory is an IShellItemImageFactory as COM hands it over: a pointer
// to its table of methods.
type imageFactory struct {
	methods *[imageFactoryGetImage + 1]uintptr
}

// NewApplicationIcons returns this machine's ApplicationIcons: the shell's
// IShellItemImageFactory. Where any of the functions it needs cannot be found
// it reports that application icons are not available. The bridge is ignored.
func NewApplicationIcons(_ Bridge) WindowsApplicationIcons {
	for _, proc := range []*windows.LazyProc{
		procCreateItemFromParsingName, procGetObject, procGetDIBits, procDeleteObject, procGetDC, procReleaseDC,
	} {
		if proc.Find() != nil {
			return NewWindowsApplicationIcons(nil)
		}
	}

	return NewWindowsApplicationIcons(readShellImage)
}

// readShellImage is the WindowsShellImage on Windows. The shell cannot be
// cancelled part way, and drawing one icon takes milliseconds, so the context
// is not consulted.
func readShellImage(_ context.Context, request WindowsIconRequest) (WindowsBitmap, error) {
	parsingName := request.ParsingName

	if request.KnownFolder != "" {
		folder, folderError := knownFolderPath(request.KnownFolder)
		if folderError != nil {
			return WindowsBitmap{}, folderError
		}

		parsingName = folder + `\` + parsingName
	}

	name, nameError := windows.UTF16PtrFromString(parsingName)
	if nameError != nil {
		return WindowsBitmap{}, fmt.Errorf("converting the item's name: %w", nameError)
	}

	// COM state belongs to a thread, so the goroutine is pinned to its thread
	// while COM is in use, as shellExecute does.
	runtime.LockOSThread()

	defer runtime.UnlockOSThread()

	comError := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE)
	if comError == nil || errors.Is(comError, windows.Errno(windows.S_FALSE)) {
		defer windows.CoUninitialize()
	}

	bitmap, drawError := drawShellImage(name, uint32(request.Pixels)) //nolint:gosec // ApplicationIconPixels
	if drawError != nil {
		return WindowsBitmap{}, drawError
	}

	defer func() { _, _, _ = procDeleteObject.Call(bitmap) }()

	return readBitmap(bitmap)
}

// drawShellImage finds the item and has the shell draw its icon, returning
// the HBITMAP, which the caller deletes.
func drawShellImage(name *uint16, pixels uint32) (uintptr, error) {
	var factory *imageFactory

	// SHCreateItemFromParsingName takes the name, the interface's ID and where
	// to put the interface; unsafe is the only way to pass their addresses.
	result, _, _ := procCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(name)), 0, //nolint:gosec // see above
		uintptr(unsafe.Pointer(&iidShellItemImageFactory)), uintptr(unsafe.Pointer(&factory))) //nolint:gosec // see above
	if comFailed(result) || factory == nil {
		return 0, WindowsShellFailure("finding the item", hresult(result))
	}

	self := uintptr(unsafe.Pointer(factory)) //nolint:gosec // a COM interface passes itself as its first argument

	defer func() { _, _, _ = syscall.SyscallN(factory.methods[comRelease], self) }()

	var bitmap uintptr

	// The bitmap's address is converted within each call, as Go requires for
	// a pointer passed to a system call; the SIZE is one argument or two.
	getImage := factory.methods[imageFactoryGetImage]

	size := WindowsSizeArguments(pixels, pixels, unsafe.Sizeof(bitmap))
	if len(size) == 1 {
		result, _, _ = syscall.SyscallN(getImage, self, size[0], shellImageIconOnly,
			uintptr(unsafe.Pointer(&bitmap))) //nolint:gosec // see above
	} else {
		result, _, _ = syscall.SyscallN(getImage, self, size[0], size[1], shellImageIconOnly,
			uintptr(unsafe.Pointer(&bitmap))) //nolint:gosec // see above
	}

	if comFailed(result) || bitmap == 0 {
		return 0, WindowsShellFailure("drawing the icon", hresult(result))
	}

	return bitmap, nil
}

// hresult is a COM call's HRESULT, the low 32 bits of what it returns; the
// rest of the register holds nothing.
func hresult(returned uintptr) uint32 {
	return uint32(returned) //nolint:gosec // see above
}

// comFailed reports whether a COM call failed, which its HRESULT's top bit
// says; S_FALSE and other results without it are successes.
func comFailed(returned uintptr) bool {
	return hresult(returned)&comFailureBit != 0
}

// readBitmap reads a bitmap's pixels as 32 bit blue, green, red and alpha,
// top row first, which a negative height asks for.
func readBitmap(bitmap uintptr) (WindowsBitmap, error) {
	var description bitmapDescription

	described, _, _ := procGetObject.Call(bitmap, unsafe.Sizeof(description),
		uintptr(unsafe.Pointer(&description))) //nolint:gosec // GetObjectW fills in the description
	if described == 0 || description.width <= 0 || description.height <= 0 {
		return WindowsBitmap{}, errBitmapNotDescribed
	}

	width, height := int(description.width), int(description.height)
	pixels := make([]byte, width*height*windowsBytesPerPixel)
	header := bitmapHeader{
		size: bitmapHeaderBytes, width: description.width, height: -description.height, planes: bitmapPlanes,
		bitsPerPixel: bitmapBitsPerPixel, compression: bitmapUncompressed, imageSize: 0,
		xPixelsPerMetre: 0, yPixelsPerMetre: 0, coloursUsed: 0, coloursNeeded: 0,
	}

	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return WindowsBitmap{}, errBitmapNotRead
	}

	defer func() { _, _, _ = procReleaseDC.Call(0, screen) }()

	// GetDIBits copies the pixels into the buffer and returns the rows
	// copied; unsafe is the only way to pass the buffer and the header.
	rows, _, _ := procGetDIBits.Call(screen, bitmap, 0, uintptr(height),
		uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), //nolint:gosec // see above
		bitmapRGBColours)
	if int(rows) != height {
		return WindowsBitmap{}, fmt.Errorf("%w: %d of %d rows", errBitmapNotRead, rows, height)
	}

	return WindowsBitmap{Width: width, Height: height, Pixels: pixels}, nil
}
