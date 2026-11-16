package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// ApplicationIconPixels is the size an application's icon is drawn at before
// it is handed to the renderer: well above a key's 96 pixels, so scaling down
// keeps it sharp, and well inside the bound render.DecodeUserIcon applies.
const ApplicationIconPixels = 256

// macOSPathPrefix marks an identifier that is a path to an application
// bundle, which open -a also accepts.
const macOSPathPrefix = "/"

// ApplicationReference says how an identifier names an application on macOS.
type ApplicationReference int

// The ways an identifier can name an application.
const (
	// ByName is an application's name, such as Safari or Discord.app.
	ByName ApplicationReference = iota
	// ByBundleIdentifier is a reverse DNS bundle identifier, such as
	// com.apple.Safari.
	ByBundleIdentifier
	// ByPath is the full path to an application bundle.
	ByPath
)

// IconRequest is one application whose icon is wanted.
type IconRequest struct {
	Target string
	By     ApplicationReference
}

// IconLookup finds an application and returns its icon as PNG bytes,
// ApplicationIconPixels square. On macOS it is AppKit, behind cgo.
type IconLookup func(lookupContext context.Context, request IconRequest) ([]byte, error)

// MacOSApplicationIcons is the ApplicationIcons adapter on macOS. The decisions
// live here, untagged, so they are tested on every machine; the lookup itself
// is supplied by the build tagged file.
type MacOSApplicationIcons struct {
	lookup IconLookup
}

// NewMacOSApplicationIcons returns the macOS ApplicationIcons over a lookup.
func NewMacOSApplicationIcons(lookup IconLookup) MacOSApplicationIcons {
	return MacOSApplicationIcons{lookup: lookup}
}

// Icon reads an application's icon. The identifier passes the same rule a
// launch target does before anything is looked up, and the bytes the lookup
// returns are decoded under the same bounds as a person's own icon file.
func (icons MacOSApplicationIcons) Icon(iconContext context.Context, identifier string) (image.Image, error) {
	checkError := platform.CheckLaunchTarget(identifier)
	if checkError != nil {
		return nil, fmt.Errorf("checking the application: %w", checkError)
	}

	request := IconRequest{Target: identifier, By: ByName}

	switch {
	case strings.HasPrefix(identifier, macOSPathPrefix):
		request.By = ByPath
	case isMacOSBundleIdentifier(identifier):
		request.By = ByBundleIdentifier
	}

	content, lookupError := icons.lookup(iconContext, request)
	if lookupError != nil {
		return nil, fmt.Errorf("reading the icon of %q: %w", identifier, lookupError)
	}

	icon, decodeError := render.DecodeUserIcon(content)
	if decodeError != nil {
		return nil, fmt.Errorf("decoding the icon of %q: %w", identifier, decodeError)
	}

	return icon, nil
}

// The Windows icon reader's limits and results: four bytes a pixel, and the
// HRESULTs, from WinError.h, the shell gives for an item it cannot find (a
// file, a path or a name not found, each a Win32 error as an HRESULT).
const (
	windowsBytesPerPixel       = 4
	windowsFileNotFoundResult  = 0x80070002
	windowsPathNotFoundResult  = 0x80070003
	windowsInvalidNameResult   = 0x8007007B
	windowsWidePointerBytes    = 8
	windowsSizeHeightShift     = 32
	windowsOpaqueAlpha         = 0xFF
	windowsPremultiplyRounding = 0xFF / 2
)

// ErrWindowsBitmapMalformed reports a bitmap from the shell whose size and
// pixels do not agree, or that is larger than any icon is allowed to be.
var ErrWindowsBitmapMalformed = errors.New("the shell's bitmap is malformed")

// WindowsIconRequest is one shell item whose picture is wanted: its parsing
// name, which SHCreateItemFromParsingName takes, put after the path of
// KnownFolder when that is set, and the square size in pixels to draw it at.
type WindowsIconRequest struct {
	ParsingName string
	KnownFolder string
	Pixels      int
}

// WindowsBitmap is a picture the shell drew: Width by Height pixels of blue,
// green, red and alpha, top row first, as GetDIBits writes a 32 bit bitmap
// asked for with a negative height.
type WindowsBitmap struct {
	Width  int
	Height int
	Pixels []byte
}

// WindowsShellImage asks the shell for an item's picture. On Windows it is
// IShellItemImageFactory, reached without cgo.
type WindowsShellImage func(readContext context.Context, request WindowsIconRequest) (WindowsBitmap, error)

// WindowsApplicationIcons is the ApplicationIcons adapter on Windows. The
// decisions live here, untagged, so they are tested on every machine; the
// shell call itself is supplied by the build tagged file.
type WindowsApplicationIcons struct {
	read WindowsShellImage
}

// NewWindowsApplicationIcons returns the Windows ApplicationIcons over a shell
// image reader. With none, as when the shell's functions cannot be found, it
// reports that application icons are not available.
func NewWindowsApplicationIcons(read WindowsShellImage) WindowsApplicationIcons {
	return WindowsApplicationIcons{read: read}
}

// Icon reads an application's icon. The identifier passes the same rule a
// launch target does before anything is read, and is read as the launcher and
// the quitter read it: an app ID or other Start menu entry through the Apps
// folder, a program by its path, and a path under a known folder by that
// folder's path, which the shell finds more surely than an Apps folder entry
// holding a backslash.
func (icons WindowsApplicationIcons) Icon(iconContext context.Context, identifier string) (image.Image, error) {
	checkError := platform.CheckLaunchTarget(identifier)
	if checkError != nil {
		return nil, fmt.Errorf("checking the application: %w", checkError)
	}

	if icons.read == nil {
		return nil, fmt.Errorf("%w: the shell's icon functions were not found", platform.ErrIconUnavailable)
	}

	request := WindowsIconRequest{ParsingName: "", KnownFolder: "", Pixels: ApplicationIconPixels}

	target := NewWindowsQuitTarget(identifier)

	switch target.Kind {
	case WindowsQuitByPackage, WindowsQuitByAppID:
		request.ParsingName = windowsAppsFolderPrefix + target.Value
	case WindowsQuitByPath:
		request.ParsingName = target.Value
		request.KnownFolder = target.KnownFolder
	}

	bitmap, readError := icons.read(iconContext, request)
	if readError != nil {
		return nil, fmt.Errorf("reading the icon of %q: %w", identifier, readError)
	}

	icon, convertError := windowsBitmapImage(bitmap)
	if convertError != nil {
		return nil, fmt.Errorf("converting the icon of %q: %w", identifier, convertError)
	}

	return icon, nil
}

// windowsBitmapImage turns the shell's bitmap into an image. The shell draws
// icons with premultiplied alpha, as image.RGBA holds them, so the channels
// are only reordered. A bitmap with every alpha byte zero was drawn with no
// alpha channel and is shown opaque; one with any colour above its own alpha
// cannot be premultiplied, so it holds straight alpha and is premultiplied
// here.
func windowsBitmapImage(bitmap WindowsBitmap) (*image.RGBA, error) {
	if bitmap.Width <= 0 || bitmap.Height <= 0 ||
		bitmap.Width > render.MaximumUserIconPixels || bitmap.Height > render.MaximumUserIconPixels {
		return nil, fmt.Errorf("%w: %d by %d pixels; the limit is %d", ErrWindowsBitmapMalformed,
			bitmap.Width, bitmap.Height, render.MaximumUserIconPixels)
	}

	if len(bitmap.Pixels) != bitmap.Width*bitmap.Height*windowsBytesPerPixel {
		return nil, fmt.Errorf("%w: %d bytes for %d by %d pixels", ErrWindowsBitmapMalformed,
			len(bitmap.Pixels), bitmap.Width, bitmap.Height)
	}

	icon := image.NewRGBA(image.Rect(0, 0, bitmap.Width, bitmap.Height))
	hasAlpha, straight := windowsAlphaKind(bitmap.Pixels)

	for offset := 0; offset < len(bitmap.Pixels); offset += windowsBytesPerPixel {
		pixel := bitmap.Pixels[offset : offset+windowsBytesPerPixel]
		blue, green, red, alpha := pixel[0], pixel[1], pixel[2], pixel[3]

		switch {
		case !hasAlpha:
			alpha = windowsOpaqueAlpha
		case straight:
			red, green, blue = premultiply(red, alpha), premultiply(green, alpha), premultiply(blue, alpha)
		}

		// image.RGBA keeps its pixels in the same order, four bytes each, so
		// the offset into one is the offset into the other.
		icon.Pix[offset], icon.Pix[offset+1], icon.Pix[offset+2], icon.Pix[offset+3] = red, green, blue, alpha
	}

	return icon, nil
}

// windowsAlphaKind reports whether any pixel has alpha, and whether any
// colour is above its own alpha, which premultiplied alpha never is.
func windowsAlphaKind(pixels []byte) (bool, bool) {
	hasAlpha, straight := false, false

	for offset := 0; offset < len(pixels); offset += windowsBytesPerPixel {
		alpha := pixels[offset+3]
		hasAlpha = hasAlpha || alpha != 0
		straight = straight || max(pixels[offset], pixels[offset+1], pixels[offset+2]) > alpha
	}

	return hasAlpha, straight
}

// premultiply scales a colour by its alpha, rounding to the nearest.
func premultiply(colour byte, alpha byte) byte {
	product := uint(colour)*uint(alpha) + windowsPremultiplyRounding

	return byte(product / windowsOpaqueAlpha) //nolint:gosec // at most 255 by 255 over 255, which fits a byte
}

// WindowsShellFailure turns a failed shell call's HRESULT into an error: a
// file, path or name not found says the application is not found, and any
// other failure is reported with its result.
func WindowsShellFailure(operation string, result uint32) error {
	switch result {
	case windowsFileNotFoundResult, windowsPathNotFoundResult, windowsInvalidNameResult:
		return fmt.Errorf("%w: %s: HRESULT %#x", platform.ErrApplicationNotFound, operation, result)
	default:
		return fmt.Errorf("%w: %s: HRESULT %#x", errWindowsShellFailed, operation, result)
	}
}

// errWindowsShellFailed reports a shell call that failed for a reason other
// than the item not being found.
var errWindowsShellFailed = errors.New("the shell call failed")

// WindowsSizeArguments is how a SIZE passed by value goes to a system call:
// in one register on a 64 bit system, width in the low half, and as two
// stack words, width first, on a 32 bit one.
func WindowsSizeArguments(width uint32, height uint32, pointerBytes uintptr) []uintptr {
	if pointerBytes == windowsWidePointerBytes {
		return []uintptr{uintptr(uint64(width) | uint64(height)<<windowsSizeHeightShift)}
	}

	return []uintptr{uintptr(width), uintptr(height)}
}
