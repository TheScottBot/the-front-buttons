package objcbridge

import "github.com/ebitengine/purego/objc"

// runningApplicationMessages are the classes and selectors the running
// application lists send, bound once.
type runningApplicationMessages struct {
	workspace          objc.ID
	runningOfWorkspace objc.SEL
	applicationClass   objc.ID
	withBundleID       objc.SEL
}

//nolint:gochecknoglobals // bound once, on first use
var runningApplicationBinding = Bind(bindRunningApplications)

func bindRunningApplications() (runningApplicationMessages, error) {
	_, loadError := Framework(PathAppKit)
	if loadError != nil {
		return runningApplicationMessages{}, loadError
	}

	workspaceClass, workspaceError := Class("NSWorkspace")
	applicationClass, applicationError := Class("NSRunningApplication")
	shared, sharedError := ClassSelector(workspaceClass, "sharedWorkspace")
	running, runningError := InstanceSelector(workspaceClass, "runningApplications")
	withBundleID, bundleError := ClassSelector(applicationClass, "runningApplicationsWithBundleIdentifier:")

	for _, failure := range []error{workspaceError, applicationError, sharedError, runningError, bundleError} {
		if failure != nil {
			return runningApplicationMessages{}, failure
		}
	}

	return runningApplicationMessages{
		workspace:          objc.ID(workspaceClass).Send(shared),
		runningOfWorkspace: running,
		applicationClass:   objc.ID(applicationClass),
		withBundleID:       withBundleID,
	}, nil
}

// RunningApplications lists every running application, as NSWorkspace's
// runningApplications does: NSRunningApplication objects, not retained, so
// call it inside WithAutoreleasePool and use them before the pool drains.
// It may be called from any thread. AppKit missing gives none.
func RunningApplications() []objc.ID {
	messages, bindError := runningApplicationBinding()
	if bindError != nil {
		return nil
	}

	return objectItems(messages.workspace.Send(messages.runningOfWorkspace))
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
