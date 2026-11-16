//nolint:testpackage // the agreement tests reach the unexported adapters through Bridge
package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// agreementAttempts is how many times the window lists are read before a
// difference counts: a window's title can change between two reads.
const agreementAttempts = 3

// accessibilityGranted skips a test when macOS does not let the test binary
// read other applications' windows, since every list would then be empty
// through both bridges and agree without proving anything. It asks the Dock,
// which always runs, for its role.
func accessibilityGranted(t *testing.T) {
	t.Helper()

	dock := objcbridge.AXApplication(runningProcess(t, "com.apple.dock"))
	defer objcbridge.Release(dock)

	value, copyError := objcbridge.AXCopy(dock, "AXRole")
	objcbridge.Release(value)

	var code objcbridge.AXError
	if errors.As(copyError, &code) && code == objcbridge.ErrAXAPIDisabled {
		t.Skip("Accessibility is not granted to the test binary (the Dock's AXRole gave kAXErrorAPIDisabled); " +
			"grant it to the app running go test to compare real windows")
	}
}

// windowListerThrough returns the WindowLister behind the bridge's
// WindowCycler. In a build without cgo both bridges give the purego one.
func windowListerThrough(t *testing.T, bridge Bridge) WindowLister {
	t.Helper()

	ask := func() { t.Error("want no Accessibility prompt from reading windows") }

	switch cycler := NewWindowCycler(ask, bridge).(type) {
	case WindowCycler:
		return cycler.list
	case pureWindowCycler:
		return cycler.list
	default:
		t.Fatalf("want a WindowCycler through %s, got %T", bridge, cycler)

		return nil
	}
}

// readWindows lists the windows of the applications whose identifier starts
// with prefix through one bridge, and releases the snapshot.
func readWindows(t *testing.T, bridge Bridge, prefix string) []AppWindow {
	t.Helper()

	snapshot, listError := windowListerThrough(t, bridge)(context.Background(), prefix)
	if listError != nil {
		t.Fatalf("listing windows through %s: %v", bridge, listError)
	}

	defer snapshot.Release()

	return slices.Clone(snapshot.Windows())
}

// regularApplications reads the bundle identifier of the frontmost
// application, and of every running regular application, through the
// objcbridge helpers the purego adapter does not share with the cgo one.
func regularApplications(t *testing.T) (string, []string) {
	t.Helper()

	var (
		frontmost   string
		identifiers []string
	)

	objcbridge.WithAutoreleasePool(func() {
		workspace := objc.ID(objc.GetClass("NSWorkspace")).Send(objc.RegisterName("sharedWorkspace"))
		front := workspace.Send(objc.RegisterName("frontmostApplication"))
		frontmost = objcbridge.GoStringFromNS(front.Send(objc.RegisterName("bundleIdentifier")))

		for _, running := range objcbridge.RunningApplications() {
			if objc.Send[int](running, objc.RegisterName("activationPolicy")) != 0 {
				continue
			}

			identifier := objcbridge.GoStringFromNS(running.Send(objc.RegisterName("bundleIdentifier")))
			if identifier != "" {
				identifiers = append(identifiers, identifier)
			}
		}
	})

	if len(identifiers) == 0 {
		t.Fatal("want at least one running regular application, got none")
	}

	return frontmost, identifiers
}

// coveringPrefixes are prefixes that between them take in every identifier:
// each identifier's first character, once.
func coveringPrefixes(identifiers []string) []string {
	var prefixes []string

	for _, identifier := range identifiers {
		if first := identifier[:1]; !slices.Contains(prefixes, first) {
			prefixes = append(prefixes, first)
		}
	}

	slices.Sort(prefixes)

	return prefixes
}

// agreeOn reads the windows for prefix through both bridges, retrying a
// difference in case a title changed between reads, and fails when they
// still differ. It returns the purego list.
func agreeOn(t *testing.T, prefix string) []AppWindow {
	t.Helper()

	var cgo, purego []AppWindow

	for attempt := 1; attempt <= agreementAttempts; attempt++ {
		cgo, purego = eachBridge(t, func(t *testing.T, bridge Bridge) []AppWindow {
			t.Helper()

			return readWindows(t, bridge, prefix)
		})

		if slices.Equal(cgo, purego) {
			return purego
		}
	}

	t.Errorf("prefix %q: the bridges list different windows after %d reads\ncgo:    %+v\npurego: %+v",
		prefix, agreementAttempts, cgo, purego)

	return purego
}

func TestWindowAgreement_ListsTheSameWindows_ForTheFrontmostApplication(t *testing.T) {
	t.Parallel()

	if !CgoAvailable {
		t.Skip("this build has no cgo bridge to compare with")
	}

	accessibilityGranted(t)

	frontmost, _ := regularApplications(t)
	if frontmost == "" {
		t.Skip("the frontmost application has no bundle identifier")
	}

	windows := agreeOn(t, frontmost)
	t.Logf("%s: %d standard windows, %d focused", frontmost, len(windows),
		len(slices.DeleteFunc(slices.Clone(windows), func(window AppWindow) bool { return !window.Focused })))
}

func TestWindowAgreement_ListsTheSameWindows_ForEveryRegularApplication(t *testing.T) {
	t.Parallel()

	if !CgoAvailable {
		t.Skip("this build has no cgo bridge to compare with")
	}

	accessibilityGranted(t)

	_, identifiers := regularApplications(t)

	total := 0

	for _, prefix := range coveringPrefixes(identifiers) {
		total += len(agreeOn(t, prefix))
	}

	// A desktop with a person at it has windows open, so none through either
	// bridge means both failed to read them. A continuous integration Mac has
	// no one at it and often no standard window at all: the bridges agreeing
	// on none is all it can show.
	if total == 0 && os.Getenv("CI") != "" {
		t.Skipf("no standard window open on this continuous integration Mac (%d regular applications)", len(identifiers))
	}

	if total == 0 {
		t.Error("want at least one standard window among the regular applications, got none through either bridge")
	}

	t.Logf("%d regular applications, %d standard windows", len(identifiers), total)
}

func TestWindowAgreement_ListsNothing_ForAnEmptyOrUnknownPrefix(t *testing.T) {
	t.Parallel()

	accessibilityGranted(t)

	for _, prefix := range []string{"", "com.example.no-such-application-anywhere"} {
		cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) []AppWindow {
			t.Helper()

			return readWindows(t, bridge, prefix)
		})

		if len(cgo) != 0 || len(purego) != 0 {
			t.Errorf("prefix %q: want no windows, got %d through cgo and %d through purego", prefix, len(cgo), len(purego))
		}
	}
}

// focusedIndex is the index of the focused window in a list, or -1.
func focusedIndex(windows []AppWindow) int {
	return slices.IndexFunc(windows, func(window AppWindow) bool { return window.Focused })
}

// TestWindowAgreement_RaisesAWindow_ThatTheOtherBridgeSeesInFront raises a
// window of the frontmost application through each bridge and reads it back
// through the other, then raises the window that was in front again. It
// changes which window is in front, so it runs alone.
//
//nolint:paralleltest // acting tests run one at a time, under the desktop lock
func TestWindowAgreement_RaisesAWindow_ThatTheOtherBridgeSeesInFront(t *testing.T) {
	actingAllowed(t)
	accessibilityGranted(t)

	if !CgoAvailable {
		t.Skip("this build has no cgo bridge to read back through")
	}

	frontmost, _ := regularApplications(t)

	for _, bridges := range [][2]Bridge{{BridgePurego, BridgeCgo}, {BridgeCgo, BridgePurego}} {
		raising, reading := bridges[0], bridges[1]

		snapshot, listError := windowListerThrough(t, raising)(context.Background(), frontmost)
		if listError != nil {
			t.Fatalf("listing windows through %s: %v", raising, listError)
		}

		windows := snapshot.Windows()
		original := focusedIndex(windows)

		if len(windows) < 2 || original < 0 {
			snapshot.Release()
			t.Skipf("%s needs two standard windows, one focused, to raise one; it has %d", frontmost, len(windows))
		}

		target := (original + 1) % len(windows)

		raiseError := snapshot.Raise(windows[target].Index)
		if raiseError != nil {
			t.Errorf("raising %q through %s: %v", windows[target].Title, raising, raiseError)
		}

		seen := readWindows(t, reading, frontmost)
		if index := focusedIndex(seen); index < 0 || seen[index].Title != windows[target].Title {
			t.Errorf("raised %q through %s; want it in front through %s, got %+v",
				windows[target].Title, raising, reading, seen)
		}

		raiseError = snapshot.Raise(windows[original].Index)
		if raiseError != nil {
			t.Errorf("raising %q back through %s: %v", windows[original].Title, raising, raiseError)
		}

		snapshot.Release()
	}
}

// finderIdentifier is the Finder's bundle identifier. The Finder runs
// whenever someone is logged in, and is a regular application, so the fake
// desktop gives it the windows.
const finderIdentifier = "com.apple.finder"

// fakeProcessBase is above any process ID macOS hands out, so an
// Accessibility element made for one belongs to no application: reading or
// setting it through the fake desktop never reaches a real window.
const fakeProcessBase = 2_000_000_000

// fakeWindow is one window of the fake desktop, known by its fake process ID.
type fakeWindow struct {
	processID int
	subrole   string
	title     string
	minimised bool
}

// fakeDesktop answers the window calls the way macOS does, for windows that
// do not exist: the owner's application element lists the windows, the
// front's gives the focused one, and each window gives its subrole, title
// and minimised flag. Every value it gives is owned by the caller, as macOS's
// are. Sets and actions are recorded, never made.
type fakeDesktop struct {
	t          *testing.T
	owner      int
	front      int
	windows    []fakeWindow
	focused    int
	raiseError error
	sets       []string
	actions    []string
}

// newFakeDesktop gives the Finder three windows, the second focused and the
// third a dialog, with the frontmost application as the one asked for the
// focused window.
func newFakeDesktop(t *testing.T) *fakeDesktop {
	t.Helper()

	owner, front := runningProcess(t, finderIdentifier), frontmostProcess(t)

	return &fakeDesktop{
		t: t, owner: owner, front: front,
		windows: []fakeWindow{
			{processID: fakeProcessBase + 1, subrole: standardWindowSubrole, title: "First", minimised: false},
			{processID: fakeProcessBase + 2, subrole: standardWindowSubrole, title: "Second", minimised: true},
			{processID: fakeProcessBase + 3, subrole: "AXDialog", title: "A dialog", minimised: false},
		},
		focused: fakeProcessBase + 2, raiseError: nil, sets: nil, actions: nil,
	}
}

// runningProcess is the process ID of the running application with the
// bundle identifier, read through objcbridge alone.
func runningProcess(t *testing.T, bundleID string) int {
	t.Helper()

	processID := 0

	objcbridge.WithAutoreleasePool(func() {
		if found := objcbridge.RunningApplicationsWithBundleID(bundleID); len(found) > 0 {
			processID = int(objc.Send[int32](found[0], objc.RegisterName("processIdentifier")))
		}
	})

	if processID == 0 {
		t.Skipf("%s is not running", bundleID)
	}

	return processID
}

// frontmostProcess is the process ID of the frontmost application.
func frontmostProcess(t *testing.T) int {
	t.Helper()

	processID := 0

	objcbridge.WithAutoreleasePool(func() {
		workspace := objc.ID(objc.GetClass("NSWorkspace")).Send(objc.RegisterName("sharedWorkspace"))
		front := workspace.Send(objc.RegisterName("frontmostApplication"))
		processID = int(objc.Send[int32](front, objc.RegisterName("processIdentifier")))
	})

	if processID == 0 {
		t.Skip("no frontmost application")
	}

	return processID
}

// windowOf finds the fake window an element was made for.
func (desktop *fakeDesktop) windowOf(element objcbridge.CFTypeRef) (fakeWindow, bool) {
	processID, pidError := objcbridge.AXProcessID(element)
	if pidError != nil {
		return fakeWindow{processID: 0, subrole: "", title: "", minimised: false}, false
	}

	index := slices.IndexFunc(desktop.windows, func(window fakeWindow) bool { return window.processID == processID })
	if index < 0 {
		return fakeWindow{processID: 0, subrole: "", title: "", minimised: false}, false
	}

	return desktop.windows[index], true
}

// windowArray makes an owned array of fresh elements for the fake windows,
// held by the array alone, as AXWindows gives them.
func (desktop *fakeDesktop) windowArray() objcbridge.CFTypeRef {
	array := objc.ID(objc.GetClass("NSMutableArray")).Send(objc.RegisterName("new"))

	for _, window := range desktop.windows {
		element := objcbridge.AXApplication(window.processID)
		array.Send(objc.RegisterName("addObject:"), objc.ID(element))
		objcbridge.Release(element)
	}

	return objcbridge.CFTypeRef(array)
}

func (desktop *fakeDesktop) copy(element objcbridge.CFTypeRef, attribute string) (objcbridge.CFTypeRef, error) {
	processID, _ := objcbridge.AXProcessID(element)

	switch {
	case attribute == "AXWindows" && processID == desktop.owner:
		return desktop.windowArray(), nil
	case attribute == "AXFocusedWindow" && processID == desktop.front:
		return objcbridge.AXApplication(desktop.focused), nil
	}

	window, isWindow := desktop.windowOf(element)
	if !isWindow {
		return 0, fmt.Errorf("fake copy of %s from process %d: %w", attribute, processID,
			objcbridge.ErrAXAttributeUnsupported)
	}

	return desktop.windowAttribute(window, attribute)
}

// windowAttribute gives one fake window's attribute, owned by the caller.
func (desktop *fakeDesktop) windowAttribute(window fakeWindow, attribute string) (objcbridge.CFTypeRef, error) {
	switch attribute {
	case "AXSubrole":
		return objcbridge.CFString(window.subrole), nil
	case "AXTitle":
		return objcbridge.CFString(window.title), nil
	case "AXMinimized":
		value, booleanError := objcbridge.CFBoolean(window.minimised)
		if booleanError != nil {
			desktop.t.Fatalf("reading a Core Foundation boolean: %v", booleanError)
		}

		return objcbridge.Retain(value), nil
	default:
		return 0, fmt.Errorf("fake copy of %s from window %d: %w", attribute, window.processID,
			objcbridge.ErrAXAttributeUnsupported)
	}
}

func (desktop *fakeDesktop) set(element objcbridge.CFTypeRef, attribute string, value objcbridge.CFTypeRef) error {
	processID, _ := objcbridge.AXProcessID(element)
	truth, _ := objcbridge.CFBool(value)
	desktop.sets = append(desktop.sets, fmt.Sprintf("%s=%v on %d", attribute, truth, processID))

	return nil
}

func (desktop *fakeDesktop) perform(element objcbridge.CFTypeRef, action string) error {
	processID, _ := objcbridge.AXProcessID(element)
	desktop.actions = append(desktop.actions, fmt.Sprintf("%s on %d", action, processID))

	return desktop.raiseError
}

// calls are the real reference calls with the fake desktop's copy, set and
// perform.
func (desktop *fakeDesktop) calls() windowCalls {
	calls := realWindowCalls()
	calls.copy, calls.set, calls.perform = desktop.copy, desktop.set, desktop.perform

	return calls
}

// finderName is the Finder's name as macOS gives it in this language.
func finderName(t *testing.T) string {
	t.Helper()

	name := ""

	objcbridge.WithAutoreleasePool(func() {
		if found := objcbridge.RunningApplicationsWithBundleID(finderIdentifier); len(found) > 0 {
			name = objcbridge.GoStringFromNS(found[0].Send(objc.RegisterName("localizedName")))
		}
	})

	return name
}

// ownershipLedger counts, by address, the references the adapter took and
// gave up, to show it releases each one it owns exactly once and never one it
// does not own.
type ownershipLedger struct {
	t      *testing.T
	owned  map[objcbridge.CFTypeRef]int
	taken  int
	overly []objcbridge.CFTypeRef
}

func newOwnershipLedger(t *testing.T) *ownershipLedger {
	t.Helper()

	return &ownershipLedger{t: t, owned: map[objcbridge.CFTypeRef]int{}, taken: 0, overly: nil}
}

func (ledger *ownershipLedger) take(ref objcbridge.CFTypeRef) objcbridge.CFTypeRef {
	if ref != 0 {
		ledger.owned[ref]++
		ledger.taken++
	}

	return ref
}

func (ledger *ownershipLedger) give(ref objcbridge.CFTypeRef) {
	if ref == 0 {
		return
	}

	if ledger.owned[ref] == 0 {
		// Releasing what was never taken would free memory still in use, so
		// it is recorded and not passed on.
		ledger.overly = append(ledger.overly, ref)

		return
	}

	ledger.owned[ref]--

	objcbridge.Release(ref)
}

// counting wraps calls so every reference taken and given up is counted.
func (ledger *ownershipLedger) counting(calls windowCalls) windowCalls {
	return windowCalls{
		application: func(processID int) objcbridge.CFTypeRef { return ledger.take(calls.application(processID)) },
		copy: func(element objcbridge.CFTypeRef, attribute string) (objcbridge.CFTypeRef, error) {
			value, copyError := calls.copy(element, attribute)

			return ledger.take(value), copyError
		},
		retain:  func(ref objcbridge.CFTypeRef) objcbridge.CFTypeRef { return ledger.take(calls.retain(ref)) },
		release: ledger.give,
		set:     calls.set,
		perform: calls.perform,
	}
}

// stillOwned is how many references are taken and not yet given up.
func (ledger *ownershipLedger) stillOwned() int {
	total := 0

	for _, count := range ledger.owned {
		total += count
	}

	return total
}

// balanced fails the test for every reference released without being owned,
// and every one still owned.
func (ledger *ownershipLedger) balanced(when string) {
	ledger.t.Helper()

	for _, ref := range ledger.overly {
		ledger.t.Errorf("%s: released %#x without owning it", when, ref)
	}

	for ref, count := range ledger.owned {
		if count != 0 {
			ledger.t.Errorf("%s: %#x still owned %d times", when, ref, count)
		}
	}
}

func TestPureWindows_ListsTheStandardWindows_AndMarksTheFocusedOneByValue_OnAFakeDesktop(t *testing.T) {
	t.Parallel()

	desktop := newFakeDesktop(t)
	name := finderName(t)

	snapshot, listError := pureWindowLister{calls: desktop.calls()}.list(context.Background(), "com.apple.find")
	if listError != nil {
		t.Fatalf("listing windows: %v", listError)
	}

	defer snapshot.Release()

	// The frontmost application gives its focused window as a fresh
	// element, equal to the second window's but at another address.
	want := []AppWindow{
		{Application: name, Identifier: finderIdentifier, Title: "First", Focused: false, Index: 0},
		{Application: name, Identifier: finderIdentifier, Title: "Second", Focused: true, Index: 1},
	}

	if got := snapshot.Windows(); !slices.Equal(got, want) {
		t.Errorf("want the Finder's two standard windows\nwant %+v\ngot  %+v", want, got)
	}
}

func TestPureWindows_ReleasesEveryReferenceItOwns_AndNoOther_OnAFakeDesktop(t *testing.T) {
	t.Parallel()

	desktop := newFakeDesktop(t)
	ledger := newOwnershipLedger(t)

	snapshot, listError := pureWindowLister{calls: ledger.counting(desktop.calls())}.list(
		context.Background(), finderIdentifier)
	if listError != nil {
		t.Fatalf("listing windows: %v", listError)
	}

	if still, held := ledger.stillOwned(), len(snapshot.Windows()); still != held || held != 2 {
		t.Errorf("want one reference held per standard window, 2, until released; %d windows, %d held", held, still)
	}

	for _, index := range []int{0, 1} {
		raiseError := snapshot.Raise(index)
		if raiseError != nil {
			t.Errorf("raising window %d on the fake desktop: %v", index, raiseError)
		}
	}

	snapshot.Release()
	snapshot.Release()
	ledger.balanced("after listing, raising and releasing twice")

	if ledger.taken == 0 {
		t.Error("want references taken and given up, got none")
	}
}

func TestPureWindows_HoldsEachWindow_AcrossCalls_OnAFakeDesktop(t *testing.T) {
	t.Parallel()

	desktop := newFakeDesktop(t)

	snapshot, listError := pureWindowLister{calls: desktop.calls()}.list(context.Background(), finderIdentifier)
	if listError != nil {
		t.Fatalf("listing windows: %v", listError)
	}

	defer snapshot.Release()

	pure, isPure := snapshot.(*pureWindows)
	if !isPure || len(pure.held) != 2 {
		t.Fatalf("want a *pureWindows snapshot holding two windows, got %T", snapshot)
	}

	// The list call has returned and the windows array is released: only
	// the snapshot's own references keep each element alive.
	for index, held := range pure.held {
		want := desktop.windows[index].processID

		if !objcbridge.IsType(held.element, objcbridge.KindAXElement) {
			t.Fatalf("window %d: want an Accessibility element still held, got %#x", index, held.element)
		}

		processID, pidError := objcbridge.AXProcessID(held.element)
		if pidError != nil || processID != want {
			t.Errorf("window %d: want the held element's process %d, got %d, %v", index, want, processID, pidError)
		}

		if held.processID != desktop.owner {
			t.Errorf("window %d: want its application's process %d kept, got %d", index, desktop.owner, held.processID)
		}
	}
}

func TestPureWindows_Raise_RestoresRaisesAndBringsForward_OnAFakeDesktop(t *testing.T) {
	t.Parallel()

	desktop := newFakeDesktop(t)

	snapshot, listError := pureWindowLister{calls: desktop.calls()}.list(context.Background(), finderIdentifier)
	if listError != nil {
		t.Fatalf("listing windows: %v", listError)
	}

	defer snapshot.Release()

	first, second, owner := fakeProcessBase+1, fakeProcessBase+2, desktop.owner

	for _, example := range []struct {
		index   int
		sets    []string
		actions []string
	}{
		{
			index:   0,
			sets:    []string{fmt.Sprintf("AXMain=true on %d", first), fmt.Sprintf("AXFrontmost=true on %d", owner)},
			actions: []string{fmt.Sprintf("AXRaise on %d", first)},
		},
		{
			index: 1,
			sets: []string{
				fmt.Sprintf("AXMinimized=false on %d", second), fmt.Sprintf("AXMain=true on %d", second),
				fmt.Sprintf("AXFrontmost=true on %d", owner),
			},
			actions: []string{fmt.Sprintf("AXRaise on %d", second)},
		},
	} {
		desktop.sets, desktop.actions = nil, nil

		raiseError := snapshot.Raise(example.index)
		if raiseError != nil {
			t.Errorf("window %d: want it raised, got %v", example.index, raiseError)
		}

		if !slices.Equal(desktop.sets, example.sets) || !slices.Equal(desktop.actions, example.actions) {
			t.Errorf("window %d:\nwant sets %v and actions %v\ngot  sets %v and actions %v",
				example.index, example.sets, example.actions, desktop.sets, desktop.actions)
		}
	}

	desktop.raiseError = objcbridge.ErrAXCannotComplete

	raiseError := snapshot.Raise(0)
	if !errors.Is(raiseError, errPureWindowNotRaised) || !errors.Is(raiseError, objcbridge.ErrAXCannotComplete) {
		t.Errorf("want a refused raise reported with its AXError, got %v", raiseError)
	}
}

func TestPureWindows_RaiseRefusesAnIndexOutsideTheSnapshot(t *testing.T) {
	t.Parallel()

	snapshot := &pureWindows{calls: realWindowCalls(), held: nil, windows: nil}
	defer snapshot.Release()

	for _, index := range []int{-1, 0, 1} {
		raiseError := snapshot.Raise(index)
		if !errors.Is(raiseError, errPureWindowNotRaised) {
			t.Errorf("Raise(%d) on an empty snapshot: want errPureWindowNotRaised, got %v", index, raiseError)
		}
	}
}

// TestPureWindows_SpellsEachName_AsTheSDKHeadersDefineIt reads the CFSTR
// macros from the macOS SDK's HIServices headers, where they are defined,
// since the library exports no symbol to check them against.
func TestPureWindows_SpellsEachName_AsTheSDKHeadersDefineIt(t *testing.T) {
	t.Parallel()

	output, xcrunError := exec.CommandContext(context.Background(), "xcrun", "--sdk", "macosx", "--show-sdk-path").Output()
	if xcrunError != nil {
		t.Skipf("no macOS SDK to read the headers from: %v", xcrunError)
	}

	headers := filepath.Join(strings.TrimSpace(string(output)),
		"System/Library/Frameworks/ApplicationServices.framework/Frameworks/HIServices.framework/Headers")

	var definitions strings.Builder

	for _, header := range []string{"AXAttributeConstants.h", "AXActionConstants.h", "AXRoleConstants.h"} {
		text, readError := os.ReadFile(filepath.Join(headers, header)) //nolint:gosec // a path under the SDK
		if readError != nil {
			t.Skipf("reading %s: %v", header, readError)
		}

		definitions.Write(text)
	}

	for macro, spelt := range map[string]string{
		"kAXWindowsAttribute":       attributeWindows,
		"kAXSubroleAttribute":       attributeSubrole,
		"kAXTitleAttribute":         attributeTitle,
		"kAXFocusedWindowAttribute": attributeFocusedWindow,
		"kAXMinimizedAttribute":     attributeMinimised,
		"kAXMainAttribute":          attributeMain,
		"kAXFrontmostAttribute":     attributeFrontmost,
		"kAXRaiseAction":            actionRaise,
		"kAXStandardWindowSubrole":  standardWindowSubrole,
	} {
		definition := regexp.MustCompile(`#define\s+` + macro + `\s+CFSTR\("([^"]*)"\)`).
			FindStringSubmatch(definitions.String())
		if definition == nil {
			t.Errorf("%s: not defined in the SDK headers", macro)

			continue
		}

		if definition[1] != spelt {
			t.Errorf("%s: the headers define %q, the adapter spells %q", macro, definition[1], spelt)
		}
	}
}

func TestPureWindowCycler_UsesThePuregoAccessibilityCheck_AndPassesAskOn(t *testing.T) {
	t.Parallel()

	asked := 0
	cycler := newPureWindowCycler(func() { asked++ })

	if cycler.trusted == nil || cycler.list == nil || cycler.ask == nil {
		t.Fatal("want the purego window cycler built with a trust check, a lister and an ask")
	}

	if got, want := cycler.trusted(), pureAccessibilityTrusted(); got != want {
		t.Errorf("want the purego trust check's answer %v, got %v", want, got)
	}

	cycler.ask()

	if asked != 1 {
		t.Errorf("want ask passed on and called once, got %d", asked)
	}
}

// TestPureWindows_ReleasesEveryReferenceItOwns_OnTheDesktop lists the real
// windows of every regular application through the counting calls; it reads
// and changes nothing on screen.
func TestPureWindows_ReleasesEveryReferenceItOwns_OnTheDesktop(t *testing.T) {
	t.Parallel()

	accessibilityGranted(t)

	frontmost, identifiers := regularApplications(t)
	ledger := newOwnershipLedger(t)
	lister := pureWindowLister{calls: ledger.counting(realWindowCalls())}

	for _, prefix := range append(coveringPrefixes(identifiers), frontmost) {
		snapshot, listError := lister.list(context.Background(), prefix)
		if listError != nil {
			t.Fatalf("listing windows for %q: %v", prefix, listError)
		}

		if still, held := ledger.stillOwned(), len(snapshot.Windows()); still != held {
			t.Errorf("prefix %q: want one reference held per window, %d, until released; holds %d", prefix, held, still)
		}

		snapshot.Release()
		ledger.balanced("prefix " + prefix)
	}
}
