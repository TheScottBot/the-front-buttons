package systemadapter_test

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The Windows identifiers the shell icon tests read, named once.
const (
	systemFolderID      = "{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}"
	signalProgramPath   = `C:\Users\person\AppData\Local\Programs\signal-desktop\Signal.exe`
	bytesPerShellPixel  = 4
	opaqueAlpha         = 255
	shellNotFoundResult = 0x80070002
)

// shellImageRecorder is a fake shell image reader: it records each request
// and hands back a fixed bitmap or a fixed error.
type shellImageRecorder struct {
	requests []systemadapter.WindowsIconRequest
	bitmap   systemadapter.WindowsBitmap
	failure  error
}

func (recorder *shellImageRecorder) read(_ context.Context,
	request systemadapter.WindowsIconRequest,
) (systemadapter.WindowsBitmap, error) {
	recorder.requests = append(recorder.requests, request)

	return recorder.bitmap, recorder.failure
}

// bgraBitmap is a bitmap of the given size whose pixels, top row first, are
// the BGRA quadruples given, as GetDIBits writes them.
func bgraBitmap(width int, height int, pixels ...[bytesPerShellPixel]byte) systemadapter.WindowsBitmap {
	content := make([]byte, 0, len(pixels)*bytesPerShellPixel)
	for _, pixel := range pixels {
		content = append(content, pixel[:]...)
	}

	return systemadapter.WindowsBitmap{Width: width, Height: height, Pixels: content}
}

// opaqueBitmap is a square bitmap of one opaque colour.
func opaqueBitmap(side int) systemadapter.WindowsBitmap {
	pixels := make([][bytesPerShellPixel]byte, side*side)
	for index := range pixels {
		pixels[index] = [bytesPerShellPixel]byte{1, 2, 3, opaqueAlpha}
	}

	return bgraBitmap(side, side, pixels...)
}

// shellIcon reads one icon through a fake reader handing back the bitmap.
func shellIcon(t *testing.T, bitmap systemadapter.WindowsBitmap) (image.Image, error) {
	t.Helper()

	recorder := &shellImageRecorder{requests: nil, bitmap: bitmap, failure: nil}

	icon, iconError := systemadapter.NewWindowsApplicationIcons(recorder.read).Icon(t.Context(), teamsAppID)
	if iconError != nil {
		return nil, fmt.Errorf("reading the icon: %w", iconError)
	}

	return icon, nil
}

// requirePixels checks an icon's pixels, row by row from the top.
func requirePixels(t *testing.T, icon image.Image, want ...color.RGBA) {
	t.Helper()

	bounds := icon.Bounds()
	if bounds.Dx()*bounds.Dy() != len(want) {
		t.Fatalf("want %d pixels, got %v", len(want), bounds)
	}

	for index, wanted := range want {
		x, y := bounds.Min.X+index%bounds.Dx(), bounds.Min.Y+index/bounds.Dx()

		got := color.RGBAModel.Convert(icon.At(x, y))
		if got != wanted {
			t.Errorf("pixel (%d, %d): want %v, got %v", x, y, wanted, got)
		}
	}
}

// TestWindowsApplicationIcons_ReadsTheItemTheLauncherOpens: an identifier is
// read as the launcher and the quitter read it, so the icon is that of the
// application a press opens: an app ID or Start menu entry through the Apps
// folder, a program by its path, and a path under a known folder by that
// folder.
func TestWindowsApplicationIcons_ReadsTheItemTheLauncherOpens(t *testing.T) {
	t.Parallel()

	want := map[string]systemadapter.WindowsIconRequest{
		teamsAppID:            {ParsingName: teamsAppsFolderTarget, KnownFolder: "", Pixels: 256},
		teamsAppsFolderTarget: {ParsingName: teamsAppsFolderTarget, KnownFolder: "", Pixels: 256},
		`shell:AppsFolder\Chrome`: {
			ParsingName: `shell:AppsFolder\Chrome`, KnownFolder: "", Pixels: 256,
		},
		`shell:AppsFolder\` + systemFolderID + `\notepad.exe`: {
			ParsingName: "notepad.exe", KnownFolder: systemFolderID, Pixels: 256,
		},
		signalProgramPath: {ParsingName: signalProgramPath, KnownFolder: "", Pixels: 256},
		`shell:AppsFolder\` + signalProgramPath: {
			ParsingName: signalProgramPath, KnownFolder: "", Pixels: 256,
		},
	}

	for identifier, request := range want {
		recorder := &shellImageRecorder{requests: nil, bitmap: opaqueBitmap(256), failure: nil}

		icon, iconError := systemadapter.NewWindowsApplicationIcons(recorder.read).Icon(t.Context(), identifier)
		if iconError != nil || icon.Bounds().Dx() != 256 || icon.Bounds().Dy() != 256 {
			t.Fatalf("%q: want a 256 pixel icon, got %v (%v)", identifier, icon, iconError)
		}

		if len(recorder.requests) != 1 || recorder.requests[0] != request {
			t.Errorf("%q: want one read of %+v, got %+v", identifier, request, recorder.requests)
		}
	}
}

func TestWindowsApplicationIcons_RefusesABadIdentifier_BeforeReadingAnything(t *testing.T) {
	t.Parallel()

	for _, identifier := range []string{"", "-a", "Signal\n"} {
		recorder := &shellImageRecorder{requests: nil, bitmap: opaqueBitmap(1), failure: nil}

		_, iconError := systemadapter.NewWindowsApplicationIcons(recorder.read).Icon(t.Context(), identifier)
		if !errors.Is(iconError, platform.ErrLaunchRefused) || len(recorder.requests) != 0 {
			t.Errorf("%q: want ErrLaunchRefused with no read, got %v after %d", identifier, iconError,
				len(recorder.requests))
		}
	}
}

func TestWindowsApplicationIcons_ReportsTheReadersFailure(t *testing.T) {
	t.Parallel()

	recorder := &shellImageRecorder{
		requests: nil, bitmap: systemadapter.WindowsBitmap{Width: 0, Height: 0, Pixels: nil},
		failure: platform.ErrApplicationNotFound,
	}

	icon, iconError := systemadapter.NewWindowsApplicationIcons(recorder.read).Icon(t.Context(), teamsAppID)
	if !errors.Is(iconError, platform.ErrApplicationNotFound) || icon != nil {
		t.Fatalf("want ErrApplicationNotFound and no icon, got %v (%v)", icon, iconError)
	}
}

// TestWindowsApplicationIcons_WithNoReader_SaysIconsAreUnavailable: an adapter
// made without a reader, as when the shell's functions cannot be found, says
// icons are unavailable rather than failing some other way.
func TestWindowsApplicationIcons_WithNoReader_SaysIconsAreUnavailable(t *testing.T) {
	t.Parallel()

	icon, iconError := systemadapter.NewWindowsApplicationIcons(nil).Icon(t.Context(), teamsAppID)
	if !errors.Is(iconError, platform.ErrIconUnavailable) || icon != nil {
		t.Fatalf("want ErrIconUnavailable and no icon, got %v (%v)", icon, iconError)
	}
}

// TestWindowsApplicationIcons_TurnsBGRAIntoRGBA_TopRowFirst: GetDIBits writes
// blue, green, red and alpha, top row first when asked for a negative height;
// the icon holds the same premultiplied colours as red, green, blue and alpha.
func TestWindowsApplicationIcons_TurnsBGRAIntoRGBA_TopRowFirst(t *testing.T) {
	t.Parallel()

	icon, iconError := shellIcon(t, bgraBitmap(2, 2,
		[bytesPerShellPixel]byte{10, 20, 30, 40},
		[bytesPerShellPixel]byte{0, 0, 200, opaqueAlpha},
		[bytesPerShellPixel]byte{0, 0, 0, 0},
		[bytesPerShellPixel]byte{60, 50, 40, 100},
	))
	if iconError != nil {
		t.Fatal(iconError)
	}

	requirePixels(t, icon,
		color.RGBA{R: 30, G: 20, B: 10, A: 40},
		color.RGBA{R: 200, G: 0, B: 0, A: opaqueAlpha},
		color.RGBA{R: 0, G: 0, B: 0, A: 0},
		color.RGBA{R: 40, G: 50, B: 60, A: 100},
	)
}

// TestWindowsApplicationIcons_TakesABitmapWithNoAlphaAsOpaque: a bitmap drawn
// with no alpha channel at all has every alpha byte zero, which would make it
// invisible; it is shown opaque instead.
func TestWindowsApplicationIcons_TakesABitmapWithNoAlphaAsOpaque(t *testing.T) {
	t.Parallel()

	icon, iconError := shellIcon(t, bgraBitmap(2, 1,
		[bytesPerShellPixel]byte{10, 20, 30, 0},
		[bytesPerShellPixel]byte{0, 0, 0, 0},
	))
	if iconError != nil {
		t.Fatal(iconError)
	}

	requirePixels(t, icon,
		color.RGBA{R: 30, G: 20, B: 10, A: opaqueAlpha},
		color.RGBA{R: 0, G: 0, B: 0, A: opaqueAlpha},
	)
}

// TestWindowsApplicationIcons_PremultipliesABitmapWithStraightAlpha: a colour
// above its own alpha cannot be premultiplied, so the bitmap holds straight
// alpha, and every pixel is premultiplied, rounding to the nearest.
func TestWindowsApplicationIcons_PremultipliesABitmapWithStraightAlpha(t *testing.T) {
	t.Parallel()

	icon, iconError := shellIcon(t, bgraBitmap(3, 1,
		[bytesPerShellPixel]byte{51, 0, 255, 51},
		[bytesPerShellPixel]byte{0, 50, 0, 40},
		[bytesPerShellPixel]byte{0, 191, 0, 2},
	))
	if iconError != nil {
		t.Fatal(iconError)
	}

	// 51 by 51 is 2601, 10.2 times 255; 50 by 40 is 2000, 7.84 times; and
	// 191 by 2 is 382, 1.498 times.
	requirePixels(t, icon,
		color.RGBA{R: 51, G: 0, B: 10, A: 51},
		color.RGBA{R: 0, G: 8, B: 0, A: 40},
		color.RGBA{R: 0, G: 1, B: 0, A: 2},
	)
}

func TestWindowsApplicationIcons_RefusesAMalformedBitmap(t *testing.T) {
	t.Parallel()

	tooWide := render.MaximumUserIconPixels + 1

	malformed := map[string]systemadapter.WindowsBitmap{
		"no pixels":      {Width: 0, Height: 0, Pixels: nil},
		"negative width": {Width: -1, Height: 1, Pixels: make([]byte, bytesPerShellPixel)},
		"short":          {Width: 2, Height: 2, Pixels: make([]byte, 3*bytesPerShellPixel)},
		"long":           {Width: 1, Height: 1, Pixels: make([]byte, 2*bytesPerShellPixel)},
		"too wide": {
			Width: tooWide, Height: 1, Pixels: make([]byte, tooWide*bytesPerShellPixel),
		},
	}

	for name, bitmap := range malformed {
		icon, iconError := shellIcon(t, bitmap)
		if !errors.Is(iconError, systemadapter.ErrWindowsBitmapMalformed) || icon != nil {
			t.Errorf("%s: want ErrWindowsBitmapMalformed and no icon, got %v (%v)", name, icon, iconError)
		}
	}
}

// TestWindowsShellFailure_ReadsAMissingItemAsNotFound: the shell answers an
// item it cannot find with a file, path or name not found result, which says
// the application is not found; any other failure is reported as it is.
func TestWindowsShellFailure_ReadsAMissingItemAsNotFound(t *testing.T) {
	t.Parallel()

	for _, result := range []uint32{shellNotFoundResult, 0x80070003, 0x8007007B} {
		failure := systemadapter.WindowsShellFailure("finding the item", result)
		if !errors.Is(failure, platform.ErrApplicationNotFound) {
			t.Errorf("%#x: want ErrApplicationNotFound, got %v", result, failure)
		}
	}

	for _, result := range []uint32{0x80004005, 0x80070005} {
		failure := systemadapter.WindowsShellFailure("drawing the icon", result)
		if failure == nil || errors.Is(failure, platform.ErrApplicationNotFound) {
			t.Errorf("%#x: want a failure other than ErrApplicationNotFound, got %v", result, failure)
		}
	}
}

// TestWindowsSizeArguments_PassesASIZEAsTheCallingConventionDoes: GetImage
// takes a SIZE by value, which a 64 bit call passes in one register, width in
// the low half, and a 32 bit call as two stack words, width first.
func TestWindowsSizeArguments_PassesASIZEAsTheCallingConventionDoes(t *testing.T) {
	t.Parallel()

	wide := systemadapter.WindowsSizeArguments(256, 128, 8)
	if len(wide) != 1 || uint64(wide[0]) != 256|128<<32 {
		t.Errorf("64 bit: want one argument %#x, got %#x", uint64(256|128<<32), wide)
	}

	narrow := systemadapter.WindowsSizeArguments(256, 128, 4)
	if len(narrow) != 2 || narrow[0] != 256 || narrow[1] != 128 {
		t.Errorf("32 bit: want [256 128], got %v", narrow)
	}
}
