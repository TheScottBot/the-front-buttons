package systemadapter

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#import <AppKit/AppKit.h>

// frontButtonsFreeQuitTarget frees a string made by C.CString.
static void frontButtonsFreeQuitTarget(char *target) {
	free(target);
}

// frontButtonsQuit asks every running application with the bundle identifier,
// or with the name when byBundle is 0, to quit as its own Quit command does.
// It returns how many were asked, and how many of those refused.
static int frontButtonsQuit(const char *target, int byBundle, int *refused) {
	@autoreleasepool {
		NSString *wanted = [NSString stringWithUTF8String:target];
		NSArray<NSRunningApplication *> *running = byBundle
			? [NSRunningApplication runningApplicationsWithBundleIdentifier:wanted]
			: [[NSWorkspace sharedWorkspace] runningApplications];
		int asked = 0;

		for (NSRunningApplication *application in running) {
			if (!byBundle && ![application.localizedName isEqualToString:wanted] &&
				![[[application.bundleURL lastPathComponent] stringByDeletingPathExtension] isEqualToString:wanted]) {
				continue;
			}

			asked++;

			if (![application terminate]) {
				(*refused)++;
			}
		}

		return asked;
	}
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
)

// errQuitRefused reports an application that would not be asked to quit.
var errQuitRefused = errors.New("the application refused to quit")

// ApplicationQuitter quits applications on macOS through AppKit, as their own
// Quit command does, which needs no permission. An application with unsaved
// work may still ask about it before it goes.
type ApplicationQuitter struct{}

// NewApplicationQuitter returns this machine's ApplicationQuitter.
func NewApplicationQuitter() ApplicationQuitter { return ApplicationQuitter{} }

// QuitApplication asks the application to quit, if it is running.
func (ApplicationQuitter) QuitApplication(_ context.Context, identifier string) error {
	found := MacOSQuitTarget(identifier)

	target := C.CString(found.Name)
	defer C.frontButtonsFreeQuitTarget(target)

	bundle := C.int(0)
	if found.ByBundle {
		bundle = 1
	}

	var refused C.int

	C.frontButtonsQuit(target, bundle, &refused)

	if refused > 0 {
		return fmt.Errorf("quitting %s: %w", identifier, errQuitRefused)
	}

	return nil
}
