package tablericons_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/render/tablericons"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

func loadIcons(t *testing.T) *tablericons.IconSet {
	t.Helper()

	icons, loadError := tablericons.Load()
	if loadError != nil {
		t.Fatalf("loading the vendored icon font: %v", loadError)
	}

	return icons
}

func TestLoad_ReadsEveryIconNameFromTheVendoredStylesheet(t *testing.T) {
	t.Parallel()

	// The stylesheet for 3.49.0 declares 5,247 icon rules; a few names share a
	// glyph, so at least 5,000 distinct names is a safe floor.
	const minimumIconCount = 5000

	if count := loadIcons(t).Count(); count < minimumIconCount {
		t.Fatalf("want at least %d icons, got %d", minimumIconCount, count)
	}
}

func TestIconMask_DrawsAKnownIconWithinTheRequestedSquare(t *testing.T) {
	t.Parallel()

	icons := loadIcons(t)

	for _, iconName := range []string{"brand-discord", "brand-github", "brand-teams", "circle-check", "dice-5"} {
		mask, found := icons.IconMask(iconName, 44)
		if !found {
			t.Errorf("want %s found", iconName)

			continue
		}

		if mask.Bounds().Dx() != 44 || mask.Bounds().Dy() != 44 {
			t.Errorf("%s: want a 44 pixel square, got %v", iconName, mask.Bounds())
		}

		inked := 0

		for _, alpha := range mask.Pix {
			if alpha > 0 {
				inked++
			}
		}

		if inked < 44*44/20 {
			t.Errorf("%s: want a visible glyph, only %d pixels inked", iconName, inked)
		}
	}
}

func TestIconMask_ReportsAnUnknownName(t *testing.T) {
	t.Parallel()

	if _, found := loadIcons(t).IconMask("no-such-icon", 44); found {
		t.Fatal("want an unknown icon reported missing")
	}
}

func TestIconMask_RefusesAnImplausibleSize(t *testing.T) {
	t.Parallel()

	for _, sizePixels := range []int{0, -1, 10000} {
		if _, found := loadIcons(t).IconMask("circle-check", sizePixels); found {
			t.Errorf("want size %d refused", sizePixels)
		}
	}
}

func TestRenderer_DrawsRealTablerIcons_MatchingGoldenImages(t *testing.T) {
	t.Parallel()

	// The XL's key size as read from source; TE5's model table owns it.
	const keyPixels = 96

	renderer, rendererError := render.NewRenderer(keyPixels, loadIcons(t))
	if rendererError != nil {
		t.Fatalf("building the renderer: %v", rendererError)
	}

	for _, iconName := range []string{"brand-discord", "brand-github", "brand-teams", "dice-5"} {
		appearance := render.Appearance{Label: iconName, Icon: iconName}
		testsupport.CompareWithGolden(t, "icon_"+iconName, renderer.Render(appearance))
	}
}
