package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The Dock's bundle identifier, and the Accessibility attributes the badge
// reader copies: kAXChildrenAttribute and kAXURLAttribute as
// AXAttributeConstants.h spells them, and the Dock's own AXStatusLabel, which
// has no constant.
const (
	dockBundleIdentifier = "com.apple.dock"
	childrenAttribute    = "AXChildren"
	urlAttribute         = "AXURL"
	statusLabelAttribute = "AXStatusLabel"
)

// errPureDockUnreadable reports a Dock that would not list its items, in the
// cgo adapter's words.
var errPureDockUnreadable = errors.New("the Dock could not be read")

// badgeMessages are what the badge reader binds once: the messages that find
// the Dock's process and an application bundle's identifier, and the Core
// Foundation calls that tell a CFURL.
type badgeMessages struct {
	processIdentifier objc.SEL
	bundleClass       objc.ID
	bundleWithURL     objc.SEL
	bundleIdentifier  objc.SEL
	typeOf            func(ref objcbridge.CFTypeRef) uintptr
	urlType           uintptr
}

//nolint:gochecknoglobals // bound once, on first use
var badgeBinding = objcbridge.Bind(func() (badgeMessages, error) {
	coreFoundation, coreError := objcbridge.Framework(objcbridge.PathCoreFoundation)
	_, appKitError := objcbridge.Framework(objcbridge.PathAppKit)

	loadError := errors.Join(coreError, appKitError)
	if loadError != nil {
		return badgeMessages{}, fmt.Errorf("loading the badge reader's libraries: %w", loadError)
	}

	var (
		messages badgeMessages
		urlType  func() uintptr
	)

	objcbridge.Function(&messages.typeOf, coreFoundation, "CFGetTypeID")
	objcbridge.Function(&urlType, coreFoundation, "CFURLGetTypeID")
	messages.urlType = urlType()

	running, runningError := objcbridge.Class("NSRunningApplication")
	bundle, bundleError := objcbridge.Class("NSBundle")

	var processError, withURLError, identifierError error

	messages.processIdentifier, processError = objcbridge.InstanceSelector(running, "processIdentifier")
	messages.bundleWithURL, withURLError = objcbridge.ClassSelector(bundle, "bundleWithURL:")
	messages.bundleIdentifier, identifierError = objcbridge.InstanceSelector(bundle, "bundleIdentifier")
	messages.bundleClass = objc.ID(bundle)

	return messages, errors.Join(runningError, bundleError, processError, withURLError, identifierError)
})

// pureApplicationBadges reads the badges on the Dock's icons through purego,
// making the same Accessibility calls as the cgo adapter: each application's
// Dock item has the label it shows. The applications themselves are never
// asked.
type pureApplicationBadges struct {
	trusted       func() bool
	messages      func() (badgeMessages, error)
	accessibility accessibilityCalls
}

// newPureApplicationBadges returns the ApplicationBadges through purego.
func newPureApplicationBadges() pureApplicationBadges {
	return pureApplicationBadges{
		trusted:       pureAccessibilityTrusted,
		messages:      badgeBinding,
		accessibility: systemAccessibility(),
	}
}

// Badges reads every badge on the Dock at once. Reading takes milliseconds
// and cannot be cancelled part way, so the context is not consulted.
func (badges pureApplicationBadges) Badges(context.Context) (map[string]string, error) {
	if !badges.trusted() {
		return nil, platform.ErrBadgesNotPermitted
	}

	messages, bindError := badges.messages()
	if bindError != nil {
		return nil, fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
	}

	var (
		records  strings.Builder
		readable bool
	)

	objcbridge.WithAutoreleasePool(func() { readable = badges.readDock(messages, &records) })

	if !readable {
		return nil, errPureDockUnreadable
	}

	return ParseBadges(records.String()), nil
}

// readDock writes a record for each application whose Dock icon shows a
// badge, or reports false if the Dock cannot be read.
func (badges pureApplicationBadges) readDock(messages badgeMessages, records *strings.Builder) bool {
	dock := firstRunningProcess(dockBundleIdentifier, messages.processIdentifier)
	if dock == 0 {
		return false
	}

	application := badges.accessibility.application(int(dock))
	lists, listsError := badges.accessibility.copyValue(application, childrenAttribute)
	objcbridge.Release(application)

	if listsError != nil || !objcbridge.IsType(lists, objcbridge.KindArray) {
		objcbridge.Release(lists)

		return false
	}

	defer objcbridge.Release(lists)

	for _, list := range objcbridge.CFArrayItems(lists) {
		items, itemsError := badges.accessibility.copyValue(list, childrenAttribute)
		if itemsError != nil {
			continue
		}

		for _, item := range objcbridge.CFArrayItems(items) {
			badges.appendBadge(messages, records, item)
		}

		objcbridge.Release(items)
	}

	return true
}

// appendBadge adds a Dock item's identifier and badge to the records, if it
// is an application showing a badge.
func (badges pureApplicationBadges) appendBadge(
	messages badgeMessages,
	records *strings.Builder,
	item objcbridge.CFTypeRef,
) {
	label, labelError := badges.accessibility.copyValue(item, statusLabelAttribute)
	if labelError != nil {
		return
	}

	defer objcbridge.Release(label)

	address, addressError := badges.accessibility.copyValue(item, urlAttribute)
	if addressError != nil {
		return
	}

	defer objcbridge.Release(address)

	text := objcbridge.GoString(label)
	if messages.typeOf(address) != messages.urlType || text == "" {
		return
	}

	bundle := messages.bundleClass.Send(messages.bundleWithURL, objc.ID(address))

	identifier := objcbridge.GoStringFromNS(bundle.Send(messages.bundleIdentifier))
	if identifier != "" {
		records.WriteString(identifier + badgeFieldSeparator + text + badgeRecordSeparator)
	}
}
