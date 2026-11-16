package objcbridge

import (
	"slices"

	"github.com/ebitengine/purego/objc"
)

// pathSystem is the system library holding libproc, which lists processes.
const pathSystem = "/usr/lib/libSystem.B.dylib"

// processIDSlack is room for processes that start between counting them and
// listing them.
const processIDSlack = 64

// runningApplicationMessages are the classes, selectors and function the
// running application lists use, bound once.
type runningApplicationMessages struct {
	applicationClass  objc.ID
	withBundleID      objc.SEL
	withProcessID     objc.SEL
	listAllProcessIDs func(buffer *int32, bufferBytes int32) int32
}

//nolint:gochecknoglobals // bound once, on first use
var runningApplicationBinding = Bind(bindRunningApplications)

func bindRunningApplications() (runningApplicationMessages, error) {
	_, loadError := Framework(PathAppKit)
	if loadError != nil {
		return runningApplicationMessages{}, loadError
	}

	system, systemError := Framework(pathSystem)
	if systemError != nil {
		return runningApplicationMessages{}, systemError
	}

	applicationClass, applicationError := Class("NSRunningApplication")
	withBundleID, bundleError := ClassSelector(applicationClass, "runningApplicationsWithBundleIdentifier:")
	withProcessID, processError := ClassSelector(applicationClass, "runningApplicationWithProcessIdentifier:")

	for _, failure := range []error{applicationError, bundleError, processError} {
		if failure != nil {
			return runningApplicationMessages{}, failure
		}
	}

	messages := runningApplicationMessages{
		applicationClass:  objc.ID(applicationClass),
		withBundleID:      withBundleID,
		withProcessID:     withProcessID,
		listAllProcessIDs: nil,
	}

	Function(&messages.listAllProcessIDs, system, "proc_listallpids")

	return messages, nil
}

// RunningApplications lists every running application as it is now:
// NSRunningApplication objects, not retained, so call it inside
// WithAutoreleasePool and use them before the pool drains, in the order the
// processes started. It may be called from any thread. AppKit or libproc
// missing gives none.
//
// NSWorkspace's runningApplications is not used: it is kept current only by
// the main run loop, which the daemon never runs, so it stays as it was at
// its first call and misses every application launched since. Each process
// is looked up instead, which LaunchServices answers as it is now.
func RunningApplications() []objc.ID {
	messages, bindError := runningApplicationBinding()
	if bindError != nil {
		return nil
	}

	processIDs := make([]int32, messages.listAllProcessIDs(nil, 0)+processIDSlack)

	const processIDBytes = 4

	bufferBytes := int32(len(processIDs) * processIDBytes) //nolint:gosec // a few thousand processes at most
	count := messages.listAllProcessIDs(&processIDs[0], bufferBytes)
	processIDs = processIDs[:max(0, min(int(count), len(processIDs)))]
	slices.Sort(processIDs)

	applications := make([]objc.ID, 0, len(processIDs))

	for _, processID := range processIDs {
		application := messages.applicationClass.Send(messages.withProcessID, processID)
		if application != 0 {
			applications = append(applications, application)
		}
	}

	return applications
}

// RunningApplicationsWithBundleID lists the running applications with the
// bundle identifier, as NSRunningApplication's
// runningApplicationsWithBundleIdentifier: does: not retained, so call it
// inside WithAutoreleasePool and use them before the pool drains. An empty or
// unknown identifier, or AppKit missing, gives none.
func RunningApplicationsWithBundleID(bundleID string) []objc.ID {
	messages, bindError := runningApplicationBinding()
	if bindError != nil || bundleID == "" {
		return nil
	}

	return objectItems(messages.applicationClass.Send(messages.withBundleID, NSString(bundleID)))
}

// objectItems lists an NSArray's objects; an NSArray is a CFArray.
func objectItems(array objc.ID) []objc.ID {
	items := CFArrayItems(CFTypeRef(array))
	objects := make([]objc.ID, 0, len(items))

	for _, item := range items {
		objects = append(objects, objc.ID(item))
	}

	return objects
}
