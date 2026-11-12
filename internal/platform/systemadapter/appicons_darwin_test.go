package systemadapter_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// TestApplicationIcons_ReadsARealApplicationsIcon_OnMacOS runs AppKit for real
// against applications every Mac has: Finder by bundle identifier, Safari by
// name and by path. Each must give a non empty picture with something visible
// in it.
func TestApplicationIcons_ReadsARealApplicationsIcon_OnMacOS(t *testing.T) {
	t.Parallel()

	icons := systemadapter.NewApplicationIcons()

	for _, identifier := range []string{
		"com.apple.finder", safariName, "/Applications/Safari.app", "com.apple.mail",
		"com.apple.Notes",
	} {
		icon, iconError := icons.Icon(t.Context(), identifier)
		if iconError != nil {
			t.Fatalf("%q: %v", identifier, iconError)
		}

		bounds := icon.Bounds()
		if bounds.Dx() != systemadapter.ApplicationIconPixels || bounds.Dy() != systemadapter.ApplicationIconPixels {
			t.Errorf("%q: want %d pixels square, got %v", identifier, systemadapter.ApplicationIconPixels, bounds)
		}

		centre := bounds.Min.Add(bounds.Size().Div(2))
		if _, _, _, alpha := icon.At(centre.X, centre.Y).RGBA(); alpha == 0 {
			t.Errorf("%q: want something drawn at the icon's centre", identifier)
		}

		edge, found := render.EdgeColour(icon)
		t.Logf("%q: %v, edge colour %v (found %v)", identifier, bounds, edge, found)
	}
}

func TestApplicationIcons_FailsCleanly_ForAnApplicationThatIsNotInstalled(t *testing.T) {
	t.Parallel()

	icons := systemadapter.NewApplicationIcons()

	for _, identifier := range []string{
		"com.example.no-such-application", "No Such Application Anywhere",
		"/Applications/No Such Application.app",
	} {
		icon, iconError := icons.Icon(t.Context(), identifier)
		if !errors.Is(iconError, platform.ErrApplicationNotFound) || icon != nil {
			t.Errorf("%q: want ErrApplicationNotFound and no icon, got %v (%v)", identifier, icon, iconError)
		}
	}
}
