//go:build darwin && cgo

package systemadapter

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <libproc.h>
#include <stdlib.h>
#import <AppKit/AppKit.h>

// frontButtonsComparePIDs orders process IDs, so applications are listed in
// the order their processes started.
static int frontButtonsComparePIDs(const void *first, const void *second) {
	pid_t left = *(const pid_t *)first, right = *(const pid_t *)second;
	return (left > right) - (left < right);
}

// frontButtonsRunningApplications lists every running application as it is
// now, in the order their processes started. NSWorkspace's
// runningApplications is not used: only the main run loop keeps it current,
// and the daemon never runs one, so it stays as it was at its first call and
// misses every application launched since. Each process is looked up instead,
// which LaunchServices answers as it is now. The window cycler uses it too.
NSArray<NSRunningApplication *> *frontButtonsRunningApplications(void) {
	int counted = proc_listallpids(NULL, 0);
	if (counted <= 0) {
		return @[];
	}

	int capacity = counted + 64;
	pid_t *pids = calloc((size_t)capacity, sizeof(pid_t));
	if (pids == NULL) {
		return @[];
	}

	int listed = proc_listallpids(pids, capacity * (int)sizeof(pid_t));
	if (listed > capacity) {
		listed = capacity;
	}

	NSMutableArray<NSRunningApplication *> *applications = [NSMutableArray array];

	if (listed > 0) {
		qsort(pids, (size_t)listed, sizeof(pid_t), frontButtonsComparePIDs);

		for (int index = 0; index < listed; index++) {
			NSRunningApplication *application =
				[NSRunningApplication runningApplicationWithProcessIdentifier:pids[index]];
			if (application != nil) {
				[applications addObject:application];
			}
		}
	}

	free(pids);

	return applications;
}

// frontButtonsRunningProcessIDs writes the process of each running
// application, as frontButtonsRunningApplications lists them, up to capacity,
// and returns how many there are.
static int frontButtonsRunningProcessIDs(int *processIDs, int capacity) {
	@autoreleasepool {
		NSArray<NSRunningApplication *> *applications = frontButtonsRunningApplications();
		int count = (int)applications.count;

		for (int index = 0; index < count && index < capacity; index++) {
			processIDs[index] = applications[index].processIdentifier;
		}

		return count;
	}
}

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
			: frontButtonsRunningApplications();
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

// cgoApplicationQuitter returns this machine's ApplicationQuitter.
func cgoApplicationQuitter() ApplicationQuitter { return ApplicationQuitter{} }

// runningProcessIDRoom is room for every running application's process.
const runningProcessIDRoom = 4096

// cgoRunningProcessIDs is the process of each running application, as the
// cgo quitter and window cycler list them, in their order.
func cgoRunningProcessIDs() []int32 {
	processIDs := make([]C.int, runningProcessIDRoom)
	count := int(C.frontButtonsRunningProcessIDs(&processIDs[0], C.int(len(processIDs))))

	listed := make([]int32, 0, min(count, len(processIDs)))
	for _, processID := range processIDs[:min(count, len(processIDs))] {
		listed = append(listed, int32(processID))
	}

	return listed
}

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
