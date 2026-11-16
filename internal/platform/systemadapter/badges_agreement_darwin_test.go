//nolint:testpackage // the agreement tests reach the unexported adapters through Bridge
package systemadapter

import (
	"errors"
	"maps"
	"testing"

	"github.com/ebitengine/purego"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// The names the Dock tree answers to, written out here rather than taken from
// the adapter, so a misspelt name in the adapter finds nothing.
const (
	dockChildren    = "AXChildren"
	dockURL         = "AXURL"
	dockStatusLabel = "AXStatusLabel"
)

// badgeReadAttempts is how often the badges are read through both bridges
// before a difference counts, as a badge can change between two reads.
const badgeReadAttempts = 3

// badgeAnswer is what reading the badges gave: the badges, or the error's
// words, which are the same through both bridges.
type badgeAnswer struct {
	badges map[string]string
	failed string
}

func (answer badgeAnswer) equal(other badgeAnswer) bool {
	return answer.failed == other.failed && maps.Equal(answer.badges, other.badges)
}

func readBadgesFrom(t *testing.T, reader platform.ApplicationBadges) badgeAnswer {
	t.Helper()

	badges, readError := reader.Badges(t.Context())
	if readError != nil {
		return badgeAnswer{badges: nil, failed: readError.Error()}
	}

	return badgeAnswer{badges: badges, failed: ""}
}

// processTrusted asks macOS, apart from either adapter, whether this test
// process is trusted for Accessibility.
func processTrusted(t *testing.T) bool {
	t.Helper()

	library, loadError := objcbridge.Framework(objcbridge.PathApplicationServices)
	if loadError != nil {
		t.Fatalf("loading ApplicationServices: %v", loadError)
	}

	var trusted func() bool

	objcbridge.Function(&trusted, library, "AXIsProcessTrusted")

	return trusted()
}

func TestBadges_AgreeBetweenBridges(t *testing.T) {
	t.Parallel()

	var cgo, purego badgeAnswer

	for range badgeReadAttempts {
		cgo, purego = eachBridge(t, func(t *testing.T, bridge Bridge) badgeAnswer {
			t.Helper()

			return readBadgesFrom(t, NewApplicationBadges(bridge))
		})

		if cgo.equal(purego) {
			return
		}
	}

	t.Errorf("want the same badges through both bridges, got cgo %+v and purego %+v", cgo, purego)
}

// TestPureBadges_ReadsTheDock_AsMacOSAllows reads the Dock through purego
// with the trust check passed, so the reader itself runs: it gives the same
// answer as cgo when this process is trusted, and the Dock unreadable when it
// is not, as Accessibility then refuses every copy.
func TestPureBadges_ReadsTheDock_AsMacOSAllows(t *testing.T) {
	t.Parallel()

	reader := newPureApplicationBadges()
	reader.trusted = func() bool { return true }

	if !processTrusted(t) {
		if got := readBadgesFrom(t, reader); got.failed != errPureDockUnreadable.Error() {
			t.Fatalf("untrusted: want %q, got %+v", errPureDockUnreadable, got)
		}

		return
	}

	var cgo, purego badgeAnswer

	for range badgeReadAttempts {
		cgo, purego = readBadgesFrom(t, NewApplicationBadges(BridgeCgo)), readBadgesFrom(t, reader)
		if cgo.equal(purego) {
			return
		}
	}

	t.Errorf("want the Dock read as cgo reads it, got cgo %+v and purego %+v", cgo, purego)
}

func TestPureBadges_BindsEveryMessage(t *testing.T) {
	t.Parallel()

	messages, bindError := badgeBinding()
	if bindError != nil {
		t.Fatalf("want every class, selector and function bound, got %v", bindError)
	}

	text := objcbridge.CFString("not a url ü")
	defer objcbridge.Release(text)

	if messages.urlType == 0 || messages.urlType == messages.typeOf(text) {
		t.Errorf("want CFURL's own type ID, got %d", messages.urlType)
	}
}

func TestPureBadges_PassesApplesAttributeNames(t *testing.T) {
	t.Parallel()

	checkAttributeNames(t, map[string]string{
		"kAXChildrenAttribute": childrenAttribute,
		"kAXURLAttribute":      urlAttribute,
	})
}

func TestPureBadges_AsksNothing_WhenNotTrusted_OrNotBound(t *testing.T) {
	t.Parallel()

	reader := newPureApplicationBadges()
	reader.trusted = func() bool { return false }
	reader.accessibility = refusingAccessibility(t)

	_, refused := reader.Badges(t.Context())
	if !errors.Is(refused, platform.ErrBadgesNotPermitted) {
		t.Errorf("untrusted: want ErrBadgesNotPermitted, got %v", refused)
	}

	reader.trusted = func() bool { return true }
	reader.messages = func() (badgeMessages, error) { return badgeMessages{}, objcbridge.ErrBindingFailed }

	_, unbound := reader.Badges(t.Context())
	if !errors.Is(unbound, ErrBridgeUnavailable) {
		t.Errorf("not bound: want ErrBridgeUnavailable, got %v", unbound)
	}
}

// dockTree stands in for the Dock's Accessibility elements with real Core
// Foundation objects the test holds: what each element's attributes copy to.
// Every copy hands out a new reference, as AXUIElementCopyAttributeValue
// does, so counting references afterwards shows what the reader gave back.
type dockTree struct {
	t        *testing.T
	dock     objcbridge.CFTypeRef
	values   map[objcbridge.CFTypeRef]map[string]objcbridge.CFTypeRef
	held     []objcbridge.CFTypeRef
	baseline map[objcbridge.CFTypeRef]int
}

// dockLibrary holds the Core Foundation calls the tree makes its objects
// with.
type dockLibrary struct {
	createArray func(allocator uintptr, values *objcbridge.CFTypeRef, count int, callBacks uintptr) objcbridge.CFTypeRef
	createURL   func(allocator uintptr, path objcbridge.CFTypeRef, style int, directory bool) objcbridge.CFTypeRef
	callBacks   uintptr
}

func loadDockLibrary(t *testing.T) dockLibrary {
	t.Helper()

	library, loadError := objcbridge.Framework(objcbridge.PathCoreFoundation)
	if loadError != nil {
		t.Fatalf("loading CoreFoundation: %v", loadError)
	}

	var made dockLibrary

	objcbridge.Function(&made.createArray, library, "CFArrayCreate")
	objcbridge.Function(&made.createURL, library, "CFURLCreateWithFileSystemPath")

	callBacks, symbolError := purego.Dlsym(library, "kCFTypeArrayCallBacks")
	if symbolError != nil {
		t.Fatalf("finding kCFTypeArrayCallBacks: %v", symbolError)
	}

	made.callBacks = callBacks

	return made
}

func newDockTree(t *testing.T) *dockTree {
	t.Helper()

	tree := &dockTree{
		t:        t,
		dock:     objcbridge.AXApplication(runningProcessOf(t, activationDock)),
		values:   map[objcbridge.CFTypeRef]map[string]objcbridge.CFTypeRef{},
		held:     nil,
		baseline: nil,
	}

	tree.held = append(tree.held, tree.dock)

	t.Cleanup(func() {
		for _, held := range tree.held {
			objcbridge.Release(held)
		}
	})

	return tree
}

// hold keeps a reference the tree owns, released when the test ends.
func (tree *dockTree) hold(ref objcbridge.CFTypeRef) objcbridge.CFTypeRef {
	if ref == 0 {
		tree.t.Fatal("want an object to hold, got nil")
	}

	tree.held = append(tree.held, ref)

	return ref
}

// text is a string the reader cannot mistake for a tagged pointer, whose
// references would not count.
func (tree *dockTree) text(value string) objcbridge.CFTypeRef {
	return tree.hold(objcbridge.CFString(value))
}

func (tree *dockTree) array(library dockLibrary, items ...objcbridge.CFTypeRef) objcbridge.CFTypeRef {
	var first *objcbridge.CFTypeRef
	if len(items) > 0 {
		first = &items[0]
	}

	return tree.hold(library.createArray(0, first, len(items), library.callBacks))
}

func (tree *dockTree) url(library dockLibrary, path string) objcbridge.CFTypeRef {
	return tree.hold(library.createURL(0, tree.text(path), 0, true))
}

func (tree *dockTree) set(element objcbridge.CFTypeRef, attribute string, value objcbridge.CFTypeRef) {
	if tree.values[element] == nil {
		tree.values[element] = map[string]objcbridge.CFTypeRef{}
	}

	tree.values[element][attribute] = value
}

func (tree *dockTree) calls() accessibilityCalls {
	calls := refusingAccessibility(tree.t)

	calls.application = func(processID int) objcbridge.CFTypeRef {
		if owner, _ := objcbridge.AXProcessID(tree.dock); owner != processID {
			tree.t.Errorf("want the Dock's element, process %d, asked for %d", owner, processID)
		}

		return objcbridge.Retain(tree.dock)
	}
	calls.copyValue = func(element objcbridge.CFTypeRef, attribute string) (objcbridge.CFTypeRef, error) {
		value, found := tree.values[element][attribute]
		if !found {
			return 0, objcbridge.ErrAXNoValue
		}

		return objcbridge.Retain(value), nil
	}

	return calls
}

// count notes every held object's references before the read.
func (tree *dockTree) count() {
	tree.baseline = map[objcbridge.CFTypeRef]int{}

	for _, held := range tree.held {
		tree.baseline[held] = activationRetainCount()(held)
	}
}

// balanced wants every held object back to its references before the read.
func (tree *dockTree) balanced() {
	tree.t.Helper()

	for _, held := range tree.held {
		if got := activationRetainCount()(held); got != tree.baseline[held] {
			tree.t.Errorf("object %#x: want %d references after the read, got %d", held, tree.baseline[held], got)
		}
	}
}

func (tree *dockTree) reader() pureApplicationBadges {
	reader := newPureApplicationBadges()
	reader.trusted = func() bool { return true }
	reader.accessibility = tree.calls()

	return reader
}

func TestPureBadges_ReadsEachBadgedApplication_AndReleasesWhatItCopied(t *testing.T) {
	t.Parallel()

	library, tree := loadDockLibrary(t), newDockTree(t)

	finder, mail, unbadged := tree.text("Finder ü"), tree.text("Mail ü"), tree.text("Notes ü")
	notApplication, numbered, empty := tree.text("Trash ü"), tree.text("Calendar ü"), tree.text("Music ü")
	broken, missing := tree.text("Books ü"), tree.text("Maps ü")

	tree.set(finder, dockStatusLabel, tree.text("ünï 3"))
	tree.set(finder, dockURL, tree.url(library, "/System/Library/CoreServices/Finder.app"))
	tree.set(mail, dockStatusLabel, tree.text("12 ✉︎"))
	tree.set(mail, dockURL, tree.url(library, "/System/Applications/Mail.app"))
	tree.set(unbadged, dockURL, tree.url(library, "/System/Applications/Notes.app"))
	tree.set(notApplication, dockStatusLabel, tree.text("full ü"))
	tree.set(notApplication, dockURL, tree.text("/System/Applications/Notes.app"))
	tree.set(numbered, dockStatusLabel, tree.array(library))
	tree.set(numbered, dockURL, tree.url(library, "/System/Applications/Calendar.app"))
	tree.set(empty, dockStatusLabel, tree.text(""))
	tree.set(empty, dockURL, tree.url(library, "/System/Applications/Music.app"))
	tree.set(missing, dockStatusLabel, tree.text("lost ü"))

	applications, others := tree.text("applications ü"), tree.text("others ü")
	tree.set(applications, dockChildren, tree.array(library, finder, unbadged, notApplication, numbered, empty))
	tree.set(others, dockChildren, tree.array(library, mail, missing))
	tree.set(broken, dockChildren, tree.text("not a list ü"))
	tree.set(tree.dock, dockChildren, tree.array(library, applications, broken, tree.text("no children ü"), others))

	tree.count()

	badges, readError := tree.reader().Badges(t.Context())
	if readError != nil {
		t.Fatalf("want the Dock read, got %v", readError)
	}

	want := map[string]string{"com.apple.finder": "ünï 3", "com.apple.mail": "12 ✉︎"}
	if !maps.Equal(badges, want) {
		t.Errorf("want %v, got %v", want, badges)
	}

	tree.balanced()
}

func TestPureBadges_ReportsTheDockUnreadable_WhenItListsNothing(t *testing.T) {
	t.Parallel()

	library, tree := loadDockLibrary(t), newDockTree(t)
	notAList := tree.text("not a list either ü")
	tree.count()

	_, missingError := tree.reader().Badges(t.Context())
	if !errors.Is(missingError, errPureDockUnreadable) {
		t.Errorf("no children: want the Dock unreadable, got %v", missingError)
	}

	tree.set(tree.dock, dockChildren, notAList)

	_, notListError := tree.reader().Badges(t.Context())
	if !errors.Is(notListError, errPureDockUnreadable) {
		t.Errorf("children not an array: want the Dock unreadable, got %v", notListError)
	}

	tree.balanced()

	tree.set(tree.dock, dockChildren, tree.array(library))
	tree.count()

	badges, readError := tree.reader().Badges(t.Context())
	if readError != nil || len(badges) != 0 {
		t.Errorf("an empty Dock: want no badges, got %v and %v", badges, readError)
	}

	tree.balanced()
}
