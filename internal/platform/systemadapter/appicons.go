package systemadapter

import (
	"context"
	"fmt"
	"image"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// ApplicationIconPixels is the size an application's icon is drawn at before
// it is handed to the renderer: well above a key's 96 pixels, so scaling down
// keeps it sharp, and well inside the bound render.DecodeUserIcon applies.
const ApplicationIconPixels = 256

// macOSPathPrefix marks an identifier that is a path to an application
// bundle, which open -a also accepts.
const macOSPathPrefix = "/"

// ApplicationReference says how an identifier names an application on macOS.
type ApplicationReference int

// The ways an identifier can name an application.
const (
	// ByName is an application's name, such as Safari or Discord.app.
	ByName ApplicationReference = iota
	// ByBundleIdentifier is a reverse DNS bundle identifier, such as
	// com.apple.Safari.
	ByBundleIdentifier
	// ByPath is the full path to an application bundle.
	ByPath
)

// IconRequest is one application whose icon is wanted.
type IconRequest struct {
	Target string
	By     ApplicationReference
}

// IconLookup finds an application and returns its icon as PNG bytes,
// ApplicationIconPixels square. On macOS it is AppKit, behind cgo.
type IconLookup func(lookupContext context.Context, request IconRequest) ([]byte, error)

// MacOSApplicationIcons is the ApplicationIcons adapter on macOS. The decisions
// live here, untagged, so they are tested on every machine; the lookup itself
// is supplied by the build tagged file.
type MacOSApplicationIcons struct {
	lookup IconLookup
}

// NewMacOSApplicationIcons returns the macOS ApplicationIcons over a lookup.
func NewMacOSApplicationIcons(lookup IconLookup) MacOSApplicationIcons {
	return MacOSApplicationIcons{lookup: lookup}
}

// Icon reads an application's icon. The identifier passes the same rule a
// launch target does before anything is looked up, and the bytes the lookup
// returns are decoded under the same bounds as a person's own icon file.
func (icons MacOSApplicationIcons) Icon(iconContext context.Context, identifier string) (image.Image, error) {
	checkError := platform.CheckLaunchTarget(identifier)
	if checkError != nil {
		return nil, fmt.Errorf("checking the application: %w", checkError)
	}

	request := IconRequest{Target: identifier, By: ByName}

	switch {
	case strings.HasPrefix(identifier, macOSPathPrefix):
		request.By = ByPath
	case isMacOSBundleIdentifier(identifier):
		request.By = ByBundleIdentifier
	}

	content, lookupError := icons.lookup(iconContext, request)
	if lookupError != nil {
		return nil, fmt.Errorf("reading the icon of %q: %w", identifier, lookupError)
	}

	icon, decodeError := render.DecodeUserIcon(content)
	if decodeError != nil {
		return nil, fmt.Errorf("decoding the icon of %q: %w", identifier, decodeError)
	}

	return icon, nil
}

// WindowsApplicationIcons is the ApplicationIcons adapter on Windows. Reading
// an icon there needs shell COM (IShellItemImageFactory) for a
// shell:AppsFolder item, which could not be run or tested from the Mac this
// was written on, so rather than ship untested COM code it says plainly that
// application icons are not available yet, and the key shows the missing
// icon marker.
type WindowsApplicationIcons struct{}

// NewWindowsApplicationIcons returns the Windows ApplicationIcons.
func NewWindowsApplicationIcons() WindowsApplicationIcons { return WindowsApplicationIcons{} }

// Icon always reports that application icons are not available on Windows
// yet.
func (WindowsApplicationIcons) Icon(context.Context, string) (image.Image, error) {
	return nil, fmt.Errorf("%w: reading application icons on Windows is not written yet", platform.ErrIconUnavailable)
}
