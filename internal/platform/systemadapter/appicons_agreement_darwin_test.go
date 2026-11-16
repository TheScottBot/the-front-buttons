//nolint:testpackage // the agreement tests reach the unexported adapters through Bridge
package systemadapter

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/ebitengine/purego/objc"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/objcbridge"
)

// iconAnswer is what one bridge gave for one application: the icon, or the
// kind of error it reported.
type iconAnswer struct {
	icon image.Image
	kind string
}

// The kinds of answer an icon read can give.
const (
	iconKindRead        = "read"
	iconKindNotFound    = "application not found"
	iconKindUnavailable = "bridge unavailable"
	iconKindOther       = "other error"
)

// installedIconTargets name applications every Mac has, in each of the three
// ways an identifier can name one: by bundle identifier, by name (which goes
// through fullPathForApplication:) and by path.
func installedIconTargets() []string {
	return []string{
		finderBundleID, "com.apple.Safari", "com.apple.systempreferences", "com.apple.TextEdit",
		safariNameForAgreement, "Calculator", "/Applications/Safari.app",
		"/System/Applications/Calculator.app", "/System/Library/CoreServices/Finder.app",
	}
}

// missingIconTargets name nothing installed, in each of the three ways, and
// one name that is not valid UTF-8.
func missingIconTargets() []string {
	return []string{
		"com.example.no-such-application", "No Such Application Anywhere",
		"/Applications/No Such Application.app", "Caf\xe9 Cr\xe8me",
		"/System/Applications/Calculator.app/Contents/Info.plist",
	}
}

// safariNameForAgreement is Safari by name, the way open -a takes it.
const safariNameForAgreement = "Safari"

// finderBundleID names Finder by bundle identifier.
const finderBundleID = "com.apple.finder"

// errIconTestBinding stands for a binding that failed.
var errIconTestBinding = errors.New("no class NSWorkspace")

func readIconThrough(t *testing.T, bridge Bridge, identifier string) iconAnswer {
	t.Helper()

	icon, iconError := NewApplicationIcons(bridge).Icon(t.Context(), identifier)

	switch {
	case iconError == nil:
		return iconAnswer{icon: icon, kind: iconKindRead}
	case errors.Is(iconError, platform.ErrApplicationNotFound):
		return iconAnswer{icon: nil, kind: iconKindNotFound}
	case errors.Is(iconError, ErrBridgeUnavailable):
		return iconAnswer{icon: nil, kind: iconKindUnavailable}
	default:
		t.Logf("%s, %q: %v", bridge, identifier, iconError)

		return iconAnswer{icon: nil, kind: iconKindOther}
	}
}

// skipAgreementWithoutCgo skips a comparison in a build without cgo, where
// both bridges give the purego adapter and agree by construction.
func skipAgreementWithoutCgo(t *testing.T) {
	t.Helper()

	if !CgoAvailable {
		t.Skip("this build has no cgo, so there is no cgo adapter to agree with")
	}
}

// TestApplicationIcons_AgreeBetweenBridges_PixelForPixel reads each installed
// application's icon through cgo and through purego and wants the same
// picture, pixel for pixel, in the same bounds.
func TestApplicationIcons_AgreeBetweenBridges_PixelForPixel(t *testing.T) {
	t.Parallel()
	skipAgreementWithoutCgo(t)

	for _, identifier := range installedIconTargets() {
		t.Run(identifier, func(t *testing.T) {
			t.Parallel()

			cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) iconAnswer {
				t.Helper()

				return readIconThrough(t, bridge, identifier)
			})

			if cgo.kind != iconKindRead || purego.kind != iconKindRead {
				t.Fatalf("want an icon through both bridges, got cgo %s and purego %s", cgo.kind, purego.kind)
			}

			comparePixels(t, cgo.icon, purego.icon)
		})
	}
}

// TestApplicationIcons_AgreeBetweenBridges_OnApplicationsNotInstalled wants
// the same kind of error through both bridges for an application that is not
// there.
func TestApplicationIcons_AgreeBetweenBridges_OnApplicationsNotInstalled(t *testing.T) {
	t.Parallel()
	skipAgreementWithoutCgo(t)

	for _, identifier := range missingIconTargets() {
		cgo, purego := eachBridge(t, func(t *testing.T, bridge Bridge) iconAnswer {
			t.Helper()

			return readIconThrough(t, bridge, identifier)
		})

		if cgo.kind != iconKindNotFound || purego.kind != iconKindNotFound {
			t.Errorf("%q: want %s through both bridges, got cgo %s and purego %s",
				identifier, iconKindNotFound, cgo.kind, purego.kind)
		}
	}
}

// TestPureApplicationIcons_ReadsInstalledApplications_AtTheIconSize runs the
// purego adapter on its own, so it is checked in a build without cgo too.
func TestPureApplicationIcons_ReadsInstalledApplications_AtTheIconSize(t *testing.T) {
	t.Parallel()

	for _, identifier := range installedIconTargets() {
		answer := readIconThrough(t, BridgePurego, identifier)
		if answer.kind != iconKindRead {
			t.Errorf("%q: want an icon, got %s", identifier, answer.kind)

			continue
		}

		bounds := answer.icon.Bounds()
		if bounds.Dx() != ApplicationIconPixels || bounds.Dy() != ApplicationIconPixels {
			t.Errorf("%q: want %d pixels square, got %v", identifier, ApplicationIconPixels, bounds)
		}

		centre := bounds.Min.Add(bounds.Size().Div(2))
		if _, _, _, alpha := answer.icon.At(centre.X, centre.Y).RGBA(); alpha == 0 {
			t.Errorf("%q: want something drawn at the icon's centre", identifier)
		}
	}
}

func TestPureApplicationIcons_ReportsNotFound_ForApplicationsNotInstalled(t *testing.T) {
	t.Parallel()

	for _, identifier := range missingIconTargets() {
		if answer := readIconThrough(t, BridgePurego, identifier); answer.kind != iconKindNotFound {
			t.Errorf("%q: want %s, got %s", identifier, iconKindNotFound, answer.kind)
		}
	}
}

// comparePixels wants two pictures of the same bounds with the same colour,
// as 16 bit RGBA, at every pixel. It names the first few that differ.
func comparePixels(t *testing.T, want, got image.Image) {
	t.Helper()

	if want.Bounds() != got.Bounds() {
		t.Fatalf("want bounds %v, got %v", want.Bounds(), got.Bounds())
	}

	const reported = 5

	differing := 0

	bounds := want.Bounds()
	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for column := bounds.Min.X; column < bounds.Max.X; column++ {
			wantRed, wantGreen, wantBlue, wantAlpha := want.At(column, row).RGBA()
			gotRed, gotGreen, gotBlue, gotAlpha := got.At(column, row).RGBA()

			if wantRed == gotRed && wantGreen == gotGreen && wantBlue == gotBlue && wantAlpha == gotAlpha {
				continue
			}

			if differing < reported {
				t.Errorf("pixel (%d, %d): cgo %v, purego %v", column, row,
					[]uint32{wantRed, wantGreen, wantBlue, wantAlpha}, []uint32{gotRed, gotGreen, gotBlue, gotAlpha})
			}

			differing++
		}
	}

	if differing > 0 {
		t.Errorf("%d of %d pixels differ", differing, bounds.Dx()*bounds.Dy())
	}
}

func TestPureIconLookup_AnswersBridgeUnavailable_WhenBindingFails(t *testing.T) {
	t.Parallel()

	for name, bind := range map[string]func() (*iconCalls, error){
		"error": objcbridge.Bind(func() (*iconCalls, error) {
			return nil, errIconTestBinding
		}),
		"panic": objcbridge.Bind(func() (*iconCalls, error) {
			panic("purego: missing function")
		}),
	} {
		request := IconRequest{Target: finderBundleID, By: ByBundleIdentifier}

		content, lookupError := iconLookupThrough(bind)(t.Context(), request)
		if !errors.Is(lookupError, ErrBridgeUnavailable) || !errors.Is(lookupError, objcbridge.ErrBindingFailed) {
			t.Errorf("%s: want ErrBridgeUnavailable wrapping ErrBindingFailed, got %v", name, lookupError)
		}

		if content != nil {
			t.Errorf("%s: want no bytes, got %d", name, len(content))
		}
	}
}

func TestLoadIconCalls_BindsEveryClassAndSelector(t *testing.T) {
	t.Parallel()

	calls, bindError := loadIconCalls()
	if bindError != nil {
		t.Fatalf("want every class and selector bound, got %v", bindError)
	}

	for _, class := range []objc.ID{
		calls.workspaceClass, calls.fileManagerClass, calls.bitmapClass, calls.contextClass,
		calls.dictionaryClass, calls.deviceRGB,
	} {
		if class == 0 {
			t.Errorf("want every class and the colour space name bound, got %+v", calls)
		}
	}

	for _, wanted := range calls.selectors() {
		if *wanted.target == 0 {
			t.Errorf("want %s bound", wanted.name)
		}
	}

	if space := objcbridge.GoStringFromNS(calls.deviceRGB); space != "NSDeviceRGBColorSpace" {
		t.Errorf("want the device RGB colour space name, got %q", space)
	}
}

// TestPureIconLookup_GivesAPNGOfTheIconSize reads the lookup's bytes
// directly: a PNG, ApplicationIconPixels square, with an alpha channel.
func TestPureIconLookup_GivesAPNGOfTheIconSize(t *testing.T) {
	t.Parallel()

	content, lookupError := iconLookupThrough(bindIconCalls)(t.Context(),
		IconRequest{Target: finderBundleID, By: ByBundleIdentifier})
	if lookupError != nil {
		t.Fatalf("want Finder's icon, got %v", lookupError)
	}

	header, decodeError := png.DecodeConfig(bytes.NewReader(content))
	if decodeError != nil {
		t.Fatalf("want a PNG, got %v", decodeError)
	}

	if header.Width != ApplicationIconPixels || header.Height != ApplicationIconPixels {
		t.Errorf("want %d pixels square, got %d by %d", ApplicationIconPixels, header.Width, header.Height)
	}

	if header.ColorModel != color.NRGBAModel && header.ColorModel != color.NRGBA64Model {
		t.Errorf("want non premultiplied RGBA with alpha, got %T", header.ColorModel)
	}
}
