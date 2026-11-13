package systemadapter

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework CoreFoundation
#include <stdlib.h>
#include <string.h>
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>

// frontButtonsWindow is one standard window, retained until released.
typedef struct {
	AXUIElementRef element;
	pid_t pid;
	char *application;
	char *identifier;
	char *title;
	int focused;
} frontButtonsWindow;

// frontButtonsWindows is a snapshot of windows, in memory the caller releases
// with frontButtonsReleaseWindows.
typedef struct {
	frontButtonsWindow *items;
	int count;
} frontButtonsWindows;

// frontButtonsCopyText copies an NSString into memory the caller frees.
static char *frontButtonsCopyText(NSString *text) {
	const char *utf8 = text == nil ? "" : [text UTF8String];

	return strdup(utf8 == NULL ? "" : utf8);
}

// frontButtonsIsStandardWindow reports a window whose subrole is a standard
// window, leaving out dialogs, palettes and pop ups.
static int frontButtonsIsStandardWindow(AXUIElementRef window) {
	CFTypeRef subrole = NULL;
	if (AXUIElementCopyAttributeValue(window, kAXSubroleAttribute, &subrole) != kAXErrorSuccess || subrole == NULL) {
		return 0;
	}

	int standard = CFGetTypeID(subrole) == CFStringGetTypeID() && CFEqual(subrole, kAXStandardWindowSubrole);
	CFRelease(subrole);

	return standard;
}

// frontButtonsFocusedWindow is the window in front: the focused window of the
// frontmost application, retained, or NULL.
static AXUIElementRef frontButtonsFocusedWindow(void) {
	NSRunningApplication *front = [[NSWorkspace sharedWorkspace] frontmostApplication];
	if (front == nil) {
		return NULL;
	}

	AXUIElementRef application = AXUIElementCreateApplication([front processIdentifier]);
	CFTypeRef window = NULL;
	AXError copied = AXUIElementCopyAttributeValue(application, kAXFocusedWindowAttribute, &window);
	CFRelease(application);

	return copied == kAXErrorSuccess ? (AXUIElementRef)window : NULL;
}

// frontButtonsAppendWindows adds an application's standard windows to the
// snapshot, growing it as needed.
static void frontButtonsAppendWindows(frontButtonsWindows *list, int *capacity, NSRunningApplication *running,
	AXUIElementRef focused) {
	AXUIElementRef application = AXUIElementCreateApplication([running processIdentifier]);
	CFTypeRef windows = NULL;

	if (AXUIElementCopyAttributeValue(application, kAXWindowsAttribute, &windows) == kAXErrorSuccess &&
		windows != NULL && CFGetTypeID(windows) == CFArrayGetTypeID()) {
		CFIndex total = CFArrayGetCount((CFArrayRef)windows);

		for (CFIndex index = 0; index < total; index++) {
			AXUIElementRef window = (AXUIElementRef)CFArrayGetValueAtIndex((CFArrayRef)windows, index);
			if (!frontButtonsIsStandardWindow(window)) {
				continue;
			}

			if (list->count == *capacity) {
				int grown = *capacity == 0 ? 8 : *capacity * 2;
				frontButtonsWindow *items = realloc(list->items, sizeof(frontButtonsWindow) * grown);
				if (items == NULL) {
					break;
				}

				list->items = items;
				*capacity = grown;
			}

			CFTypeRef title = NULL;
			AXUIElementCopyAttributeValue(window, kAXTitleAttribute, &title);
			NSString *titleText = (title != NULL && CFGetTypeID(title) == CFStringGetTypeID()) ? (NSString *)title : nil;

			frontButtonsWindow *item = &list->items[list->count];
			item->element = (AXUIElementRef)CFRetain(window);
			item->pid = [running processIdentifier];
			item->application = frontButtonsCopyText([running localizedName]);
			item->identifier = frontButtonsCopyText([running bundleIdentifier]);
			item->title = frontButtonsCopyText(titleText);
			item->focused = focused != NULL && CFEqual(window, focused);
			list->count++;

			if (title != NULL) {
				CFRelease(title);
			}
		}
	}

	if (windows != NULL) {
		CFRelease(windows);
	}

	CFRelease(application);
}

// frontButtonsListWindows snapshots the standard windows of every running
// regular application whose identifier starts with the prefix.
static frontButtonsWindows frontButtonsListWindows(const char *prefix) {
	frontButtonsWindows list = { NULL, 0 };
	int capacity = 0;

	@autoreleasepool {
		NSString *wanted = [NSString stringWithUTF8String:prefix];
		AXUIElementRef focused = frontButtonsFocusedWindow();

		for (NSRunningApplication *running in [[NSWorkspace sharedWorkspace] runningApplications]) {
			NSString *identifier = [running bundleIdentifier];
			if ([running activationPolicy] != NSApplicationActivationPolicyRegular || identifier == nil ||
				wanted == nil || ![identifier hasPrefix:wanted]) {
				continue;
			}

			frontButtonsAppendWindows(&list, &capacity, running, focused);
		}

		if (focused != NULL) {
			CFRelease(focused);
		}
	}

	return list;
}

// frontButtonsRaiseWindow brings one window forward: restored if minimised,
// raised, made its application's main window, and its application brought to
// the front. It reports whether the window was raised.
static int frontButtonsRaiseWindow(frontButtonsWindows list, int index) {
	if (index < 0 || index >= list.count) {
		return 0;
	}

	frontButtonsWindow window = list.items[index];

	CFTypeRef minimised = NULL;
	if (AXUIElementCopyAttributeValue(window.element, kAXMinimizedAttribute, &minimised) == kAXErrorSuccess &&
		minimised != NULL) {
		if (CFEqual(minimised, kCFBooleanTrue)) {
			AXUIElementSetAttributeValue(window.element, kAXMinimizedAttribute, kCFBooleanFalse);
		}

		CFRelease(minimised);
	}

	AXError raised = AXUIElementPerformAction(window.element, kAXRaiseAction);
	AXUIElementSetAttributeValue(window.element, kAXMainAttribute, kCFBooleanTrue);

	AXUIElementRef application = AXUIElementCreateApplication(window.pid);
	AXUIElementSetAttributeValue(application, kAXFrontmostAttribute, kCFBooleanTrue);
	CFRelease(application);

	@autoreleasepool {
		[[NSRunningApplication runningApplicationWithProcessIdentifier:window.pid] activateWithOptions:0];
	}

	return raised == kAXErrorSuccess;
}

// frontButtonsReleaseWindows frees a snapshot.
static void frontButtonsReleaseWindows(frontButtonsWindows list) {
	for (int index = 0; index < list.count; index++) {
		CFRelease(list.items[index].element);
		free(list.items[index].application);
		free(list.items[index].identifier);
		free(list.items[index].title);
	}

	free(list.items);
}

// frontButtonsWindowAt returns one window of a snapshot, so the Go side never
// indexes C memory itself.
static frontButtonsWindow frontButtonsWindowAt(frontButtonsWindows list, int index) {
	return list.items[index];
}

// frontButtonsReleasePrefix frees a string made by C.CString.
static void frontButtonsReleasePrefix(char *text) {
	free(text);
}
*/
import "C"

import (
	"context"
	"errors"
)

// errWindowNotRaised reports a window macOS would not raise.
var errWindowNotRaised = errors.New("macOS did not raise the window")

// NewWindowCycler returns this machine's WindowCycler: the Accessibility API,
// which reads and raises any application's windows once the daemon is trusted
// for Accessibility, as F13 already needs. A refusal calls ask, the key
// presser's RequestAccessibility, so both share one prompt and one restart.
func NewWindowCycler(ask func()) WindowCycler {
	return NewWindowCyclerWith(AccessibilityTrusted, listMacOSWindows, ask)
}

// macOSWindows is a snapshot of windows held in C memory until released.
type macOSWindows struct {
	list    C.frontButtonsWindows
	windows []AppWindow
}

// listMacOSWindows snapshots the standard windows of every running
// application whose identifier starts with the prefix. Reading windows cannot
// be cancelled part way and takes milliseconds, so the context is not
// consulted.
//
//nolint:ireturn // a WindowLister returns the snapshot as its interface
func listMacOSWindows(_ context.Context, prefix string) (WindowSnapshot, error) {
	wanted := C.CString(prefix)
	list := C.frontButtonsListWindows(wanted)
	C.frontButtonsReleasePrefix(wanted)

	snapshot := &macOSWindows{list: list, windows: make([]AppWindow, 0, int(list.count))}

	for index := range int(list.count) {
		window := C.frontButtonsWindowAt(list, C.int(index)) //nolint:nlreturn // cgo expands the call into a closure
		snapshot.windows = append(snapshot.windows, AppWindow{
			Application: C.GoString(window.application),
			Identifier:  C.GoString(window.identifier),
			Title:       C.GoString(window.title),
			Focused:     window.focused != 0,
			Index:       index,
		})
	}

	return snapshot, nil
}

func (snapshot *macOSWindows) Windows() []AppWindow { return snapshot.windows }

func (snapshot *macOSWindows) Raise(index int) error {
	// cgo expands the call into a closure, which nlreturn reads as a return.
	raised := C.frontButtonsRaiseWindow(snapshot.list, C.int(index)) //nolint:nlreturn // see above
	if raised == 0 {
		return errWindowNotRaised
	}

	return nil
}

func (snapshot *macOSWindows) Release() { C.frontButtonsReleaseWindows(snapshot.list) }
