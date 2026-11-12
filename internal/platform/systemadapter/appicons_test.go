package systemadapter_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The application identifiers the icon tests read, named once.
const (
	safariBundleIdentifier = "com.apple.Safari"
	teamsBundleIdentifier  = "com.microsoft.teams2"
	safariName             = "Safari"
	discordBundleName      = "Discord.app"
	teamsAppsFolderTarget  = `shell:AppsFolder\MSTeams_8wekyb3d8bbwe!MSTeams`
)

// errLookupFailed is a lookup's own failure, which must come back wrapped.
var errLookupFailed = errors.New("lookup failed")

// iconLookupRecorder is a fake icon lookup: it records each request and hands
// back fixed bytes or a fixed error.
type iconLookupRecorder struct {
	requests []systemadapter.IconRequest
	content  []byte
	failure  error
}

func (recorder *iconLookupRecorder) lookup(_ context.Context, request systemadapter.IconRequest) ([]byte, error) {
	recorder.requests = append(recorder.requests, request)

	return recorder.content, recorder.failure
}

func smallPNG(t *testing.T) []byte {
	t.Helper()

	var encoded bytes.Buffer

	encodeError := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 16, 16)))
	if encodeError != nil {
		t.Fatal(encodeError)
	}

	return encoded.Bytes()
}

// TestMacOSApplicationIcons_FindsTheApplication_AsTheLauncherDoes: an
// identifier is read the way open_application's launcher reads it, so the icon
// shown is the application a press opens.
func TestMacOSApplicationIcons_FindsTheApplication_AsTheLauncherDoes(t *testing.T) {
	t.Parallel()

	want := map[string]systemadapter.ApplicationReference{
		safariBundleIdentifier:          systemadapter.ByBundleIdentifier,
		teamsBundleIdentifier:           systemadapter.ByBundleIdentifier,
		safariName:                      systemadapter.ByName,
		"Microsoft Teams":               systemadapter.ByName,
		discordBundleName:               systemadapter.ByName,
		"/Applications/Safari.app":      systemadapter.ByPath,
		"/System/Applications/Mail.app": systemadapter.ByPath,
	}

	for identifier, reference := range want {
		recorder := &iconLookupRecorder{requests: nil, content: smallPNG(t), failure: nil}

		icon, iconError := systemadapter.NewMacOSApplicationIcons(recorder.lookup).Icon(t.Context(), identifier)
		if iconError != nil || icon.Bounds().Dx() != 16 {
			t.Fatalf("%q: want a 16 pixel icon, got %v (%v)", identifier, icon, iconError)
		}

		if len(recorder.requests) != 1 || recorder.requests[0].By != reference ||
			recorder.requests[0].Target != identifier {
			t.Errorf("%q: want one lookup by %v, got %+v", identifier, reference, recorder.requests)
		}
	}
}

func TestMacOSApplicationIcons_RefusesABadIdentifier_BeforeLookingAnythingUp(t *testing.T) {
	t.Parallel()

	for _, identifier := range []string{"", "-a", "Safari\n"} {
		recorder := &iconLookupRecorder{requests: nil, content: smallPNG(t), failure: nil}

		_, iconError := systemadapter.NewMacOSApplicationIcons(recorder.lookup).Icon(t.Context(), identifier)
		if !errors.Is(iconError, platform.ErrLaunchRefused) || len(recorder.requests) != 0 {
			t.Errorf("%q: want ErrLaunchRefused with no lookup, got %v after %d", identifier, iconError,
				len(recorder.requests))
		}
	}
}

func TestMacOSApplicationIcons_ReportsALookupsFailure(t *testing.T) {
	t.Parallel()

	recorder := &iconLookupRecorder{requests: nil, content: nil, failure: errLookupFailed}

	icon, iconError := systemadapter.NewMacOSApplicationIcons(recorder.lookup).Icon(t.Context(), safariName)
	if !errors.Is(iconError, errLookupFailed) || icon != nil {
		t.Fatalf("want the lookup's failure and no icon, got %v (%v)", icon, iconError)
	}
}

func TestMacOSApplicationIcons_RefusesBytesThatAreNotAPNG(t *testing.T) {
	t.Parallel()

	recorder := &iconLookupRecorder{requests: nil, content: []byte("not a picture"), failure: nil}

	icon, iconError := systemadapter.NewMacOSApplicationIcons(recorder.lookup).Icon(t.Context(), safariName)
	if !errors.Is(iconError, render.ErrUserIconRejected) || icon != nil {
		t.Fatalf("want ErrUserIconRejected and no icon, got %v (%v)", icon, iconError)
	}
}

// TestWindowsApplicationIcons_SaysAppIconsAreNotAvailableYet: reading an icon
// on Windows needs shell COM, which cannot be run from the Mac this was built
// on, so the adapter says so plainly and the key shows the missing marker.
func TestWindowsApplicationIcons_SaysAppIconsAreNotAvailableYet(t *testing.T) {
	t.Parallel()

	icon, iconError := systemadapter.NewWindowsApplicationIcons().Icon(t.Context(), teamsAppsFolderTarget)
	if !errors.Is(iconError, platform.ErrIconUnavailable) || icon != nil {
		t.Fatalf("want ErrIconUnavailable and no icon, got %v (%v)", icon, iconError)
	}
}
