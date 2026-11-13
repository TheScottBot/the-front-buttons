package daemon_test

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The one application the fake system has an icon for, and the log line a
// missing one is reported with.
const (
	knownApplication  = "com.example.blue"
	appIconNotLoaded  = "application icon not loaded"
	appIconTestLabel  = "Blue"
	appIconTestColumn = 0
)

// blueAppIcon is an application icon as macOS draws one: a blue square inside a
// transparent margin.
var blueAppIcon = func() image.Image { //nolint:gochecknoglobals // a fixed test picture
	picture := image.NewNRGBA(image.Rect(0, 0, 32, 32))

	for x := 4; x < 28; x++ {
		for y := 4; y < 28; y++ {
			picture.SetNRGBA(x, y, color.NRGBA{R: 0x20, G: 0x60, B: 0xe0, A: 0xff})
		}
	}

	return picture
}()

// applicationIconsTestDouble knows one application's icon and no other.
type applicationIconsTestDouble struct{}

func (applicationIconsTestDouble) Icon(_ context.Context, identifier string) (image.Image, error) {
	if identifier == knownApplication {
		return blueAppIcon, nil
	}

	return nil, fmt.Errorf("%w: %s", platform.ErrApplicationNotFound, identifier)
}

func openAppAppearance(background string) brick.Appearance {
	return brick.Appearance{Label: appIconTestLabel, Icon: "", Background: background, IconFile: ""}
}

func appIconKey(column int, application string, background string) config.Key {
	return config.Key{
		Position: layout.Position{Row: 0, Column: column},
		Kind:     openapplication.KindName,
		Parameters: &openapplication.Parameters{
			Appearance: openAppAppearance(background), Application: application, UseAppIcon: true, AppScreen: false,
			ShowNotifications: false,
		},
		Preset: nil,
	}
}

// TestDaemon_DrawsTheApplicationsOwnIcon_OnItsEdgeColour: the author asked for
// an open_application key to show the application's icon, with the rest of the
// key the colour of the icon's edge. An explicit background wins over the
// edge, and an icon that cannot be read shows the missing marker.
func TestDaemon_DrawsTheApplicationsOwnIcon_OnItsEdgeColour(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- config.Configuration{
			SchemaVersion: config.CurrentSchemaVersion,
			Pages: []config.Page{{Name: "Apps", Application: "", Keys: []config.Key{
				appIconKey(appIconTestColumn, knownApplication, ""),
				appIconKey(1, knownApplication, "#800000"),
				appIconKey(2, "com.example.gone", ""),
			}}},
		}

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		want := []render.Appearance{
			{Label: appIconTestLabel, IconImage: blueAppIcon, BackgroundFromIcon: true},
			{
				Label: appIconTestLabel, IconImage: blueAppIcon, BackgroundFromIcon: true,
				Background: color.RGBA{R: 0x80, G: 0, B: 0, A: 0xff},
			},
			{Label: appIconTestLabel, IconMissing: true, BackgroundFromIcon: true},
		}

		// The deck receives JPEG, so each key is checked to be nearest its own
		// appearance, among the others and the icon on the default background.
		candidates := append([]render.Appearance{{Label: appIconTestLabel, IconImage: blueAppIcon}}, want...)

		for keyIndex := range want {
			drawn := drawnImage(t, device, keyIndex)
			nearest := distance(drawn, running.renderer.Render(want[keyIndex]))

			for _, other := range candidates {
				if other != want[keyIndex] && distance(drawn, running.renderer.Render(other)) <= nearest {
					t.Errorf("key %d: drawn nearer %+v than %+v", keyIndex, other, want[keyIndex])
				}
			}
		}

		running.stop(t)
	})
}

// TestDaemon_LogsAMissingApplicationIcon_OncePerConfiguration: the reason an
// icon cannot be read is logged once, not on every repaint, and again only
// when the configuration is reloaded, as icon_file's reason is.
func TestDaemon_LogsAMissingApplicationIcon_OncePerConfiguration(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)

		configuration := config.Configuration{
			SchemaVersion: config.CurrentSchemaVersion,
			Pages: []config.Page{
				{Name: "Apps", Application: "", Keys: []config.Key{appIconKey(appIconTestColumn, "com.example.gone", "")}},
				{Name: "Other", Application: "", Keys: []config.Key{button("other", 0, false)}},
			},
		}
		running.configurations <- configuration

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, forwardKey)
		tap(device, backKey)

		if logged := strings.Count(running.logs.String(), appIconNotLoaded); logged != 1 {
			t.Fatalf("want the missing icon logged once, got %d:\n%s", logged, running.logs.String())
		}

		running.configurations <- configuration

		time.Sleep(settle)
		synctest.Wait()

		if logged := strings.Count(running.logs.String(), appIconNotLoaded); logged != 2 {
			t.Fatalf("want it logged again after a reload, got %d:\n%s", logged, running.logs.String())
		}

		running.stop(t)
	})
}
