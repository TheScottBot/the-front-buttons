package systemadapter

import (
	"context"
	"errors"
	"fmt"
	"image"
	"unicode/utf8"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// AppKit's values for the messages below, from NSGraphics.h, NSImageRep.h and
// NSBitmapImageRep.h.
const (
	imageInterpolationHigh   = 3 // NSImageInterpolationHigh
	compositingOperationCopy = 1 // NSCompositingOperationCopy
	bitmapImageFileTypePNG   = 4 // NSBitmapImageFileTypePNG
	iconBitsPerSample        = 8
	iconSamplesPerPixel      = 4
)

// errPureIconNotDrawn reports an application found whose icon AppKit would
// not draw, through purego.
var errPureIconNotDrawn = errors.New("macOS did not draw the application's icon")

// iconCalls are the classes, selectors and constant the purego icon lookup
// sends messages with, bound once.
type iconCalls struct {
	workspaceClass   objc.ID
	fileManagerClass objc.ID
	bitmapClass      objc.ID
	contextClass     objc.ID
	dictionaryClass  objc.ID
	deviceRGB        objc.ID

	sharedWorkspace        objc.SEL
	urlForBundleIdentifier objc.SEL
	fullPathForApplication objc.SEL
	iconForFile            objc.SEL
	path                   objc.SEL
	defaultManager         objc.SEL
	fileExists             objc.SEL
	alloc                  objc.SEL
	initWithPlanes         objc.SEL
	setSize                objc.SEL
	representation         objc.SEL
	release                objc.SEL
	contextWithBitmap      objc.SEL
	saveGraphicsState      objc.SEL
	setCurrentContext      objc.SEL
	restoreGraphicsState   objc.SEL
	setImageInterpolation  objc.SEL
	flushGraphics          objc.SEL
	drawInRect             objc.SEL
	dictionary             objc.SEL
	length                 objc.SEL
	getBytes               objc.SEL
}

//nolint:gochecknoglobals // the classes and selectors are bound once, on first use
var bindIconCalls = objcbridge.Bind(loadIconCalls)

// The classes the lookup sends messages to or checks selectors against.
const (
	classWorkspace       = "NSWorkspace"
	classURL             = "NSURL"
	classFileManager     = "NSFileManager"
	classBitmapImageRep  = "NSBitmapImageRep"
	classGraphicsContext = "NSGraphicsContext"
	classImage           = "NSImage"
	classDictionary      = "NSDictionary"
	classData            = "NSData"
)

// iconSelector is one selector to bind, on a class or on its instances.
type iconSelector struct {
	target   *objc.SEL
	class    string
	name     string
	instance bool
}

// loadIconCalls loads AppKit and binds every class and selector the lookup
// sends, checking each class answers its selector.
func loadIconCalls() (*iconCalls, error) {
	_, foundationError := objcbridge.Framework(objcbridge.PathFoundation)
	if foundationError != nil {
		return nil, fmt.Errorf("loading Foundation: %w", foundationError)
	}

	appKit, appKitError := objcbridge.Framework(objcbridge.PathAppKit)
	if appKitError != nil {
		return nil, fmt.Errorf("loading AppKit: %w", appKitError)
	}

	deviceRGB, constantError := objcbridge.Constant(appKit, "NSDeviceRGBColorSpace")
	if constantError != nil {
		return nil, fmt.Errorf("reading the colour space name: %w", constantError)
	}

	classes := map[string]objc.Class{}

	for _, name := range []string{
		classWorkspace, classURL, classFileManager, classBitmapImageRep, classGraphicsContext, classImage,
		classDictionary, classData,
	} {
		class, classError := objcbridge.Class(name)
		if classError != nil {
			return nil, fmt.Errorf("binding the icon lookup: %w", classError)
		}

		classes[name] = class
	}

	var calls iconCalls

	calls.workspaceClass = objc.ID(classes[classWorkspace])
	calls.fileManagerClass = objc.ID(classes[classFileManager])
	calls.bitmapClass = objc.ID(classes[classBitmapImageRep])
	calls.contextClass = objc.ID(classes[classGraphicsContext])
	calls.dictionaryClass = objc.ID(classes[classDictionary])
	calls.deviceRGB = objc.ID(deviceRGB)

	for _, wanted := range calls.selectors() {
		bind := objcbridge.ClassSelector
		if wanted.instance {
			bind = objcbridge.InstanceSelector
		}

		selector, selectorError := bind(classes[wanted.class], wanted.name)
		if selectorError != nil {
			return nil, fmt.Errorf("binding the icon lookup: %w", selectorError)
		}

		*wanted.target = selector
	}

	return &calls, nil
}

// selectors lists every selector the lookup sends, with the class that
// answers it.
func (calls *iconCalls) selectors() []iconSelector {
	return []iconSelector{
		{&calls.sharedWorkspace, classWorkspace, "sharedWorkspace", false},
		{&calls.urlForBundleIdentifier, classWorkspace, "URLForApplicationWithBundleIdentifier:", true},
		{&calls.fullPathForApplication, classWorkspace, "fullPathForApplication:", true},
		{&calls.iconForFile, classWorkspace, "iconForFile:", true},
		{&calls.path, classURL, "path", true},
		{&calls.defaultManager, classFileManager, "defaultManager", false},
		{&calls.fileExists, classFileManager, "fileExistsAtPath:isDirectory:", true},
		{&calls.alloc, classBitmapImageRep, "alloc", false},
		{&calls.initWithPlanes, classBitmapImageRep, "initWithBitmapDataPlanes:pixelsWide:pixelsHigh:" +
			"bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bytesPerRow:bitsPerPixel:", true},
		{&calls.setSize, classBitmapImageRep, "setSize:", true},
		{&calls.representation, classBitmapImageRep, "representationUsingType:properties:", true},
		{&calls.release, classBitmapImageRep, "release", true},
		{&calls.contextWithBitmap, classGraphicsContext, "graphicsContextWithBitmapImageRep:", false},
		{&calls.saveGraphicsState, classGraphicsContext, "saveGraphicsState", false},
		{&calls.setCurrentContext, classGraphicsContext, "setCurrentContext:", false},
		{&calls.restoreGraphicsState, classGraphicsContext, "restoreGraphicsState", false},
		{&calls.setImageInterpolation, classGraphicsContext, "setImageInterpolation:", true},
		{&calls.flushGraphics, classGraphicsContext, "flushGraphics", true},
		{&calls.drawInRect, classImage, "drawInRect:fromRect:operation:fraction:", true},
		{&calls.dictionary, classDictionary, "dictionary", false},
		{&calls.length, classData, "length", true},
		{&calls.getBytes, classData, "getBytes:length:", true},
	}
}

// pureApplicationIcons reads applications' icons through purego, sending
// AppKit the same messages as the cgo adapter, behind the same decisions.
type pureApplicationIcons struct {
	icons MacOSApplicationIcons
}

// newPureApplicationIcons returns the ApplicationIcons through purego.
func newPureApplicationIcons() pureApplicationIcons {
	return pureApplicationIcons{icons: NewMacOSApplicationIcons(iconLookupThrough(bindIconCalls))}
}

// Icon reads an application's icon, as MacOSApplicationIcons does.
func (pure pureApplicationIcons) Icon(iconContext context.Context, identifier string) (image.Image, error) {
	return pure.icons.Icon(iconContext, identifier)
}

// iconOutcome is how reading an icon ended.
type iconOutcome int

// The ways reading an icon can end.
const (
	iconNotDrawn iconOutcome = iota
	iconRead
	iconNoApplication
)

// iconLookupThrough is the IconLookup through purego, over the calls bind
// gives; a binding that failed answers ErrBridgeUnavailable. AppKit cannot
// be cancelled part way, and drawing one icon takes milliseconds, so the
// context is not consulted. The lookup runs on the calling goroutine's
// thread, inside an autorelease pool.
func iconLookupThrough(bind func() (*iconCalls, error)) IconLookup {
	return func(_ context.Context, request IconRequest) ([]byte, error) {
		calls, bindError := bind()
		if bindError != nil {
			return nil, fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)
		}

		return calls.readApplicationIcon(request)
	}
}

// readApplicationIcon reads one application's icon as PNG bytes.
func (calls *iconCalls) readApplicationIcon(request IconRequest) ([]byte, error) {
	// NSString stringWithUTF8String: gives nil for text that is not UTF-8,
	// which the cgo adapter reports as no application.
	if !utf8.ValidString(request.Target) {
		return nil, fmt.Errorf("%w: %q", platform.ErrApplicationNotFound, request.Target)
	}

	var (
		content []byte
		outcome iconOutcome
	)

	objcbridge.WithAutoreleasePool(func() {
		content, outcome = calls.applicationIcon(request, ApplicationIconPixels)
	})

	switch outcome {
	case iconRead:
		return content, nil
	case iconNoApplication:
		return nil, fmt.Errorf("%w: %q", platform.ErrApplicationNotFound, request.Target)
	case iconNotDrawn:
	}

	return nil, errPureIconNotDrawn
}

// applicationIcon finds an application and draws its icon. It is called
// inside an autorelease pool.
func (calls *iconCalls) applicationIcon(request IconRequest, pixels int) ([]byte, iconOutcome) {
	workspace := calls.workspaceClass.Send(calls.sharedWorkspace)

	path := calls.applicationPath(workspace, objcbridge.NSString(request.Target), request.By)
	if path == 0 {
		return nil, iconNoApplication
	}

	isDirectory := false

	fileManager := calls.fileManagerClass.Send(calls.defaultManager)
	if !objc.Send[bool](fileManager, calls.fileExists, path, &isDirectory) || !isDirectory {
		return nil, iconNoApplication
	}

	icon := workspace.Send(calls.iconForFile, path)
	if icon == 0 {
		return nil, iconNotDrawn
	}

	png := calls.drawIcon(icon, pixels)
	if png == 0 {
		return nil, iconNotDrawn
	}

	length := objc.Send[uint](png, calls.length)
	if length == 0 {
		return nil, iconNotDrawn
	}

	// Anything past the decoder's bound is refused by it, so at most one byte
	// more than the bound is copied.
	length = min(length, uint(render.MaximumUserIconBytes+1))
	content := make([]byte, length)
	png.Send(calls.getBytes, &content[0], length)

	return content, iconRead
}

// applicationPath finds an application bundle's path, or nil.
// fullPathForApplication: is deprecated, but it is the one AppKit call that
// finds an application by the name open -a takes, so the icon matches what a
// press opens.
func (calls *iconCalls) applicationPath(workspace, target objc.ID, by ApplicationReference) objc.ID {
	switch by {
	case ByBundleIdentifier:
		return workspace.Send(calls.urlForBundleIdentifier, target).Send(calls.path)
	case ByPath:
		return target
	case ByName:
	}

	return workspace.Send(calls.fullPathForApplication, target)
}

// drawIcon draws an icon into a square bitmap and returns it as autoreleased
// PNG data, or nil. Drawing into a bitmap of its own, rather than asking the
// image for a representation, gives a picture of the size wanted however the
// icon is stored.
func (calls *iconCalls) drawIcon(icon objc.ID, pixels int) objc.ID {
	bitmap := calls.bitmapClass.Send(calls.alloc).Send(calls.initWithPlanes,
		uintptr(0), pixels, pixels, iconBitsPerSample, iconSamplesPerPixel, true, false, calls.deviceRGB, 0, 0)
	if bitmap == 0 {
		return 0
	}

	defer bitmap.Send(calls.release)

	side := float64(pixels)
	bitmap.Send(calls.setSize, objcbridge.NSSize{Width: side, Height: side})

	graphicsContext := calls.contextClass.Send(calls.contextWithBitmap, bitmap)
	if graphicsContext == 0 {
		return 0
	}

	whole := objcbridge.NSRect{Origin: objcbridge.NSPoint{X: 0, Y: 0}, Size: objcbridge.NSSize{Width: side, Height: side}}
	zero := objcbridge.NSRect{Origin: objcbridge.NSPoint{X: 0, Y: 0}, Size: objcbridge.NSSize{Width: 0, Height: 0}}

	calls.contextClass.Send(calls.saveGraphicsState)
	calls.contextClass.Send(calls.setCurrentContext, graphicsContext)
	graphicsContext.Send(calls.setImageInterpolation, uint(imageInterpolationHigh))
	icon.Send(calls.drawInRect, whole, zero, uint(compositingOperationCopy), float64(1))
	graphicsContext.Send(calls.flushGraphics)
	calls.contextClass.Send(calls.restoreGraphicsState)

	return bitmap.Send(calls.representation, uint(bitmapImageFileTypePNG), calls.dictionaryClass.Send(calls.dictionary))
}
