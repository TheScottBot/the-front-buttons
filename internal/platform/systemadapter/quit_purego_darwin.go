package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// errPureQuitRefused reports an application that would not be asked to quit,
// as the cgo adapter's errQuitRefused does.
var errPureQuitRefused = errors.New("the application refused to quit")

// quitNames are the classes and selectors the quitter binds, so a test can
// bind a name the system does not have.
type quitNames struct {
	applicationClass      string
	terminate             string
	localizedName         string
	bundleURL             string
	urlClass              string
	lastPathComponent     string
	stringClass           string
	deletingPathExtension string
	isEqualToString       string
}

// systemQuitNames are AppKit's and Foundation's own names.
func systemQuitNames() quitNames {
	return quitNames{
		applicationClass:      "NSRunningApplication",
		terminate:             "terminate",
		localizedName:         "localizedName",
		bundleURL:             "bundleURL",
		urlClass:              "NSURL",
		lastPathComponent:     "lastPathComponent",
		stringClass:           "NSString",
		deletingPathExtension: "stringByDeletingPathExtension",
		isEqualToString:       "isEqualToString:",
	}
}

// quitMessages are what QuitApplication sends, bound once.
type quitMessages struct {
	terminate             objc.SEL
	localizedName         objc.SEL
	bundleURL             objc.SEL
	lastPathComponent     objc.SEL
	deletingPathExtension objc.SEL
	isEqualToString       objc.SEL
}

//nolint:gochecknoglobals // bound once, on first use
var quitBinding = objcbridge.Bind(func() (quitMessages, error) { return bindQuit(systemQuitNames()) })

// bindQuit loads AppKit and finds every class and selector the quitter
// sends.
func bindQuit(names quitNames) (quitMessages, error) {
	_, loadError := objcbridge.Framework(objcbridge.PathAppKit)
	if loadError != nil {
		return quitMessages{}, fmt.Errorf("loading AppKit: %w", loadError)
	}

	applicationClass, applicationError := objcbridge.Class(names.applicationClass)
	urlClass, urlError := objcbridge.Class(names.urlClass)
	stringClass, stringError := objcbridge.Class(names.stringClass)

	var messages quitMessages

	selectorErrors := []error{applicationError, urlError, stringError}

	for _, selector := range []struct {
		target *objc.SEL
		class  objc.Class
		name   string
	}{
		{&messages.terminate, applicationClass, names.terminate},
		{&messages.localizedName, applicationClass, names.localizedName},
		{&messages.bundleURL, applicationClass, names.bundleURL},
		{&messages.lastPathComponent, urlClass, names.lastPathComponent},
		{&messages.deletingPathExtension, stringClass, names.deletingPathExtension},
		{&messages.isEqualToString, stringClass, names.isEqualToString},
	} {
		var selectorError error

		*selector.target, selectorError = objcbridge.InstanceSelector(selector.class, selector.name)
		selectorErrors = append(selectorErrors, selectorError)
	}

	bindError := errors.Join(selectorErrors...)
	if bindError != nil {
		return quitMessages{}, bindError
	}

	return messages, nil
}

// pureApplicationQuitter quits applications through purego, with AppKit, as
// their own Quit command does, which needs no permission. An application
// with unsaved work may still ask about it before it goes.
type pureApplicationQuitter struct{}

// newPureApplicationQuitter returns the ApplicationQuitter through purego.
func newPureApplicationQuitter() pureApplicationQuitter { return pureApplicationQuitter{} }

// QuitApplication asks the application to quit, if it is running.
func (pureApplicationQuitter) QuitApplication(_ context.Context, identifier string) error {
	return quitApplication(quitBinding, identifier)
}

// quitApplication asks every running application the identifier names to
// quit, with the messages binding gives, or reports the bridge unavailable
// when they could not be bound.
func quitApplication(binding func() (quitMessages, error), identifier string) error {
	messages, bindError := binding()
	if bindError != nil {
		return fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
	}

	refused := 0

	objcbridge.WithAutoreleasePool(func() {
		eachQuitCandidate(messages, MacOSQuitTarget(identifier), func(application objc.ID) {
			if !objc.Send[bool](application, messages.terminate) {
				refused++
			}
		})
	})

	if refused > 0 {
		return fmt.Errorf("quitting %s: %w", identifier, errPureQuitRefused)
	}

	return nil
}

// eachQuitCandidate calls ask with every running application the target
// names: by bundle identifier, or else by name or by its bundle's file name
// without the extension. As the cgo adapter's C string does, the name ends
// at its first NUL, and one that is not UTF-8 names nothing. It runs inside
// an autorelease pool, which holds the applications.
func eachQuitCandidate(messages quitMessages, target QuitTarget, ask func(application objc.ID)) {
	wanted, _, _ := strings.Cut(target.Name, "\x00")
	if !utf8.ValidString(wanted) {
		return
	}

	if target.ByBundle {
		for _, application := range objcbridge.RunningApplicationsWithBundleID(wanted) {
			ask(application)
		}

		return
	}

	wantedName := objcbridge.NSString(wanted)

	// A nil name or bundle is never equal: a message to nil answers NO.
	isWanted := func(name objc.ID) bool { return objc.Send[bool](name, messages.isEqualToString, wantedName) }

	for _, application := range objcbridge.RunningApplications() {
		fileName := application.Send(messages.bundleURL).Send(messages.lastPathComponent).
			Send(messages.deletingPathExtension)

		if isWanted(application.Send(messages.localizedName)) || isWanted(fileName) {
			ask(application)
		}
	}
}
