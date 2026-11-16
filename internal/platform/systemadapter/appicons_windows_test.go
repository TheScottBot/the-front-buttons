package systemadapter_test

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// requireDrawnIcon checks an icon is 256 pixels square with something drawn
// in it, so a blank bitmap does not pass for an icon.
func requireDrawnIcon(t *testing.T, identifier string, icon image.Image) {
	t.Helper()

	bounds := icon.Bounds()
	if bounds.Dx() != systemadapter.ApplicationIconPixels || bounds.Dy() != systemadapter.ApplicationIconPixels {
		t.Fatalf("%q: want %d pixels square, got %v", identifier, systemadapter.ApplicationIconPixels, bounds)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, alpha := icon.At(x, y).RGBA(); alpha != 0 {
				return
			}
		}
	}

	t.Fatalf("%q: want something drawn, got a transparent icon", identifier)
}

// TestWindowsApplicationIcons_OnWindows_DrawsAProgramsIcon: Notepad, by its
// path and by the known folder form the Start menu gives it.
func TestWindowsApplicationIcons_OnWindows_DrawsAProgramsIcon(t *testing.T) {
	t.Parallel()

	notepad := filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe")

	for _, identifier := range []string{notepad, `shell:AppsFolder\` + systemFolderID + `\notepad.exe`} {
		icon, iconError := systemadapter.NewApplicationIcons(systemadapter.BridgePurego).Icon(t.Context(), identifier)
		if iconError != nil {
			t.Fatalf("%q: %v", identifier, iconError)
		}

		requireDrawnIcon(t, identifier, icon)
	}
}

func TestWindowsApplicationIcons_OnWindows_ReportsAMissingProgramAsNotFound(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "not-a-program.exe")

	icon, iconError := systemadapter.NewApplicationIcons(systemadapter.BridgePurego).Icon(t.Context(), missing)
	if !errors.Is(iconError, platform.ErrApplicationNotFound) || icon != nil {
		t.Fatalf("want ErrApplicationNotFound and no icon, got %v (%v)", icon, iconError)
	}
}

// TestWindowsApplicationIcons_OnWindows_DrawsTheStartMenusIcons: every Start
// menu entry the app picker lists, as the picker gives it. An entry the shell
// cannot find is logged, as an installer may leave one behind; one that is
// found must draw, and at least one must.
func TestWindowsApplicationIcons_OnWindows_DrawsTheStartMenusIcons(t *testing.T) {
	t.Parallel()

	applications, listError := systemadapter.NewApplicationCatalogue().Applications(t.Context())
	if listError != nil {
		t.Fatalf("listing the Start menu: %v", listError)
	}

	icons := systemadapter.NewApplicationIcons(systemadapter.BridgePurego)
	drawn := 0

	for _, application := range applications {
		icon, iconError := icons.Icon(t.Context(), application.Identifier)

		switch {
		case errors.Is(iconError, platform.ErrApplicationNotFound):
			t.Logf("%s (%q): not found: %v", application.Name, application.Identifier, iconError)
		case iconError != nil:
			t.Errorf("%s (%q): %v", application.Name, application.Identifier, iconError)
		default:
			requireDrawnIcon(t, application.Identifier, icon)

			drawn++
		}
	}

	t.Logf("%d of %d Start menu entries drawn", drawn, len(applications))

	if drawn == 0 {
		t.Fatal("want at least one Start menu entry drawn, got none")
	}
}
