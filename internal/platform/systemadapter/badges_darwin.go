package systemadapter

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework CoreFoundation
#include <stdlib.h>
#include <string.h>
#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>

// frontButtonsCopyAttribute copies one Accessibility attribute, or NULL.
static CFTypeRef frontButtonsCopyAttribute(AXUIElementRef element, CFStringRef attribute) {
	CFTypeRef value = NULL;
	if (AXUIElementCopyAttributeValue(element, attribute, &value) != kAXErrorSuccess) {
		return NULL;
	}

	return value;
}

// frontButtonsAppendBadge adds a Dock item's identifier and badge to the
// records, if it is an application showing a badge.
static void frontButtonsAppendBadge(NSMutableString *records, AXUIElementRef item) {
	CFTypeRef label = frontButtonsCopyAttribute(item, CFSTR("AXStatusLabel"));
	if (label == NULL) {
		return;
	}

	CFTypeRef address = frontButtonsCopyAttribute(item, kAXURLAttribute);
	if (address != NULL && CFGetTypeID(address) == CFURLGetTypeID() && CFGetTypeID(label) == CFStringGetTypeID() &&
		CFStringGetLength((CFStringRef)label) > 0) {
		NSString *identifier = [[NSBundle bundleWithURL:(NSURL *)address] bundleIdentifier];
		if (identifier != nil) {
			[records appendFormat:@"%@\x1f%@\x1e", identifier, (NSString *)label];
		}
	}

	if (address != NULL) {
		CFRelease(address);
	}

	CFRelease(label);
}

// frontButtonsDockBadges returns a record for each application whose Dock
// icon shows a badge, identifier and label, in memory the caller frees with
// frontButtonsFreeBadges; or NULL if the Dock cannot be read.
static char *frontButtonsDockBadges(void) {
	@autoreleasepool {
		NSRunningApplication *dock =
			[NSRunningApplication runningApplicationsWithBundleIdentifier:@"com.apple.dock"].firstObject;
		if (dock == nil) {
			return NULL;
		}

		AXUIElementRef application = AXUIElementCreateApplication([dock processIdentifier]);
		CFTypeRef lists = frontButtonsCopyAttribute(application, kAXChildrenAttribute);
		CFRelease(application);

		if (lists == NULL || CFGetTypeID(lists) != CFArrayGetTypeID()) {
			if (lists != NULL) {
				CFRelease(lists);
			}

			return NULL;
		}

		NSMutableString *records = [NSMutableString string];

		for (CFIndex listIndex = 0; listIndex < CFArrayGetCount((CFArrayRef)lists); listIndex++) {
			AXUIElementRef list = (AXUIElementRef)CFArrayGetValueAtIndex((CFArrayRef)lists, listIndex);
			CFTypeRef items = frontButtonsCopyAttribute(list, kAXChildrenAttribute);
			if (items == NULL) {
				continue;
			}

			if (CFGetTypeID(items) == CFArrayGetTypeID()) {
				for (CFIndex itemIndex = 0; itemIndex < CFArrayGetCount((CFArrayRef)items); itemIndex++) {
					frontButtonsAppendBadge(records, (AXUIElementRef)CFArrayGetValueAtIndex((CFArrayRef)items, itemIndex));
				}
			}

			CFRelease(items);
		}

		CFRelease(lists);

		return strdup([records UTF8String]);
	}
}

// frontButtonsFreeBadges frees the records.
static void frontButtonsFreeBadges(char *records) {
	free(records);
}
*/
import "C"

import (
	"context"
	"errors"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// errDockUnreadable reports a Dock that would not list its items.
var errDockUnreadable = errors.New("the Dock could not be read")

// ApplicationBadges reads the badges on the Dock's icons on macOS, through
// Accessibility: each application's Dock item has the label it shows. The
// applications themselves are never asked.
type ApplicationBadges struct {
	trusted func() bool
}

// NewApplicationBadges returns this machine's ApplicationBadges.
func NewApplicationBadges() ApplicationBadges {
	return ApplicationBadges{trusted: AccessibilityTrusted}
}

// Badges reads every badge on the Dock at once. Reading takes milliseconds
// and cannot be cancelled part way, so the context is not consulted.
func (badges ApplicationBadges) Badges(context.Context) (map[string]string, error) {
	if !badges.trusted() {
		return nil, platform.ErrBadgesNotPermitted
	}

	records := C.frontButtonsDockBadges()
	if records == nil {
		return nil, errDockUnreadable
	}

	defer C.frontButtonsFreeBadges(records)

	return ParseBadges(C.GoString(records)), nil
}
