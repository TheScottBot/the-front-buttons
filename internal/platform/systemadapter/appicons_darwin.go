package systemadapter

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <stdlib.h>
#include <string.h>
#import <AppKit/AppKit.h>

// The outcomes of reading an icon, matched by the Go code below.
enum {
	frontButtonsIconRead = 0,
	frontButtonsIconNoApplication = 1,
	frontButtonsIconNotDrawn = 2
};

// The ways the target names an application, matching ApplicationReference.
enum {
	frontButtonsByName = 0,
	frontButtonsByBundleIdentifier = 1,
	frontButtonsByPath = 2
};

// frontButtonsApplicationPath finds an application bundle's path, or nil.
// fullPathForApplication is deprecated, but it is the one AppKit call that finds
// an application by the name open -a takes, so the icon matches what a press
// opens.
static NSString *frontButtonsApplicationPath(NSWorkspace *workspace, NSString *target, int by) {
	switch (by) {
	case frontButtonsByBundleIdentifier:
		return [[workspace URLForApplicationWithBundleIdentifier:target] path];
	case frontButtonsByPath:
		return target;
	default:
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
		return [workspace fullPathForApplication:target];
#pragma clang diagnostic pop
	}
}

// frontButtonsDrawIcon draws an icon into a square bitmap and returns it as PNG
// data, or nil. Drawing into a bitmap of our own, rather than asking the image
// for a representation, gives a picture of the size wanted however the icon is
// stored, including icons in an asset catalogue rather than a .icns file.
static NSData *frontButtonsDrawIcon(NSImage *icon, int pixels) {
	NSBitmapImageRep *bitmap = [[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL
		pixelsWide:pixels pixelsHigh:pixels bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO
		colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
	if (bitmap == nil) {
		return nil;
	}

	[bitmap setSize:NSMakeSize(pixels, pixels)];

	NSGraphicsContext *context = [NSGraphicsContext graphicsContextWithBitmapImageRep:bitmap];
	if (context == nil) {
		[bitmap release];

		return nil;
	}

	[NSGraphicsContext saveGraphicsState];
	[NSGraphicsContext setCurrentContext:context];
	[context setImageInterpolation:NSImageInterpolationHigh];
	[icon drawInRect:NSMakeRect(0, 0, pixels, pixels) fromRect:NSZeroRect
		operation:NSCompositingOperationCopy fraction:1.0];
	[context flushGraphics];
	[NSGraphicsContext restoreGraphicsState];

	NSData *png = [bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
	[bitmap release];

	return png;
}

// frontButtonsIcon is an icon read, or why not: on success its PNG bytes, in
// memory the caller releases with frontButtonsReleaseIcon. It is returned by
// value, so the Go side passes no pointers of its own across.
typedef struct {
	int outcome;
	void *bytes;
	size_t length;
} frontButtonsIcon;

// frontButtonsApplicationIcon finds an application and draws its icon.
static frontButtonsIcon frontButtonsApplicationIcon(const char *target, int by, int pixels) {
	frontButtonsIcon result = { frontButtonsIconNotDrawn, NULL, 0 };

	@autoreleasepool {
		NSString *text = [NSString stringWithUTF8String:target];
		if (text == nil) {
			result.outcome = frontButtonsIconNoApplication;

			return result;
		}

		NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
		NSString *path = frontButtonsApplicationPath(workspace, text, by);
		BOOL isDirectory = NO;
		if (path == nil || ![[NSFileManager defaultManager] fileExistsAtPath:path isDirectory:&isDirectory] ||
			!isDirectory) {
			result.outcome = frontButtonsIconNoApplication;

			return result;
		}

		NSImage *icon = [workspace iconForFile:path];
		NSData *png = icon == nil ? nil : frontButtonsDrawIcon(icon, pixels);
		if (png == nil || [png length] == 0) {
			return result;
		}

		void *copy = malloc([png length]);
		if (copy == NULL) {
			return result;
		}

		memcpy(copy, [png bytes], [png length]);
		result.outcome = frontButtonsIconRead;
		result.bytes = copy;
		result.length = [png length];
	}

	return result;
}

// frontButtonsReleaseIcon frees an icon's bytes.
static void frontButtonsReleaseIcon(frontButtonsIcon icon) {
	free(icon.bytes);
}

// frontButtonsReleaseText frees a string made by C.CString.
static void frontButtonsReleaseText(char *text) {
	free(text);
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// errIconNotDrawn reports an application found whose icon AppKit would not
// draw.
var errIconNotDrawn = errors.New("macOS did not draw the application's icon")

// NewApplicationIcons returns this machine's ApplicationIcons: AppKit's
// NSWorkspace, which finds the application as Launch Services does and draws
// its icon wherever it is stored.
func NewApplicationIcons() MacOSApplicationIcons {
	return NewMacOSApplicationIcons(readApplicationIcon)
}

// readApplicationIcon is the IconLookup on macOS. AppKit cannot be cancelled
// part way, and drawing one icon takes milliseconds, so the context is not
// consulted.
func readApplicationIcon(_ context.Context, request IconRequest) ([]byte, error) {
	target := C.CString(request.Target)
	icon := C.frontButtonsApplicationIcon(target, C.int(request.By), C.int(ApplicationIconPixels))
	C.frontButtonsReleaseText(target)

	switch icon.outcome {
	case C.frontButtonsIconRead:
	case C.frontButtonsIconNoApplication:
		return nil, fmt.Errorf("%w: %q", platform.ErrApplicationNotFound, request.Target)
	default:
		return nil, errIconNotDrawn
	}

	// Anything past the decoder's bound is refused by it, so at most one byte
	// more than the bound is copied, which also keeps the length within an int.
	length := min(uint64(icon.length), uint64(render.MaximumUserIconBytes+1))
	content := C.GoBytes(icon.bytes, C.int(length)) //nolint:nlreturn // cgo expands GoBytes into a closure
	C.frontButtonsReleaseIcon(icon)

	return content, nil
}
