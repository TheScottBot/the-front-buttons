package daemon_test

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The keys of the app screen tests: the app key and the return tile are both
// top left, and the button beside each is one column on.
const (
	slowApplication = "com.example.slow"
	appKeyIndex     = 0
	returnKeyIndex  = 0
	besideKeyIndex  = 1
)

// appScreenKey is an open_application key for the application the fake system
// has an icon for, with its app screen ticked or not.
func appScreenKey(appScreen bool) config.Key {
	return appKeyFor(knownApplication, appScreen)
}

func appKeyFor(application string, appScreen bool) config.Key {
	return config.Key{
		Position: layout.Position{Row: 0, Column: 0},
		Kind:     openapplication.KindName,
		Parameters: &openapplication.Parameters{
			Appearance: openAppAppearance(""), Application: application, UseAppIcon: true, AppScreen: appScreen,
			ShowNotifications: false,
		},
		Preset: nil,
	}
}

// keystrokeButton is a fake key that counts as sending key presses.
func keystrokeButton(name string, column int) config.Key {
	key := button(name, column, false)
	key.Kind = testsupport.KeystrokeButtonKindName

	return key
}

// appScreenConfiguration puts the app screen first in the list, so paging
// that counted it would reach it from either regular page.
func appScreenConfiguration() config.Configuration {
	return config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion,
		Pages: []config.Page{
			{Name: "Blue", Application: knownApplication, Keys: []config.Key{button("in app", 1, false)}},
			{Name: firstPageName, Application: "", Keys: []config.Key{appScreenKey(true), button("first", 1, false)}},
			{Name: secondPageName, Application: "", Keys: []config.Key{button("second", 1, false)}},
		},
	}
}

func startAppScreenHarness(t *testing.T) (*harness, *testsupport.DeckDeviceTestDouble) {
	t.Helper()

	running := startHarness(t)
	running.configurations <- appScreenConfiguration()

	device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
	running.watcher.Attach("deck-a", device)
	time.Sleep(settle)
	synctest.Wait()

	return running, device
}

func assertPresses(t *testing.T, running *harness, want ...string) {
	t.Helper()

	if got := running.recorder.Presses(); !slices.Equal(got, want) {
		t.Fatalf("want presses %v, got %v", want, got)
	}
}

// TestDaemon_OpensTheAppScreen_AndReturnsFromItsIcon: an app key with its app
// screen ticked shows the application's screen and still launches it; the
// screen's top left key is the application's icon alone, and pressing it
// returns to the regular page.
func TestDaemon_OpensTheAppScreen_AndReturnsFromItsIcon(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startAppScreenHarness(t)

		tap(device, appKeyIndex)

		if launched := running.launches.Launched(); !slices.Equal(launched, []string{knownApplication}) {
			t.Fatalf("want the application launched as well, got %v", launched)
		}

		tap(device, besideKeyIndex)
		assertPresses(t, running, "in app")

		// The return tile is the icon with no label, unlike the app key.
		iconOnly := running.renderer.Render(render.Appearance{IconImage: blueAppIcon, BackgroundFromIcon: true})
		appKey := running.renderer.Render(render.Appearance{
			Label: appIconTestLabel, IconImage: blueAppIcon, BackgroundFromIcon: true,
		})

		drawn := drawnImage(t, device, returnKeyIndex)
		if distance(drawn, iconOnly) >= distance(drawn, appKey) {
			t.Fatal("want the return tile drawn as the application's icon alone")
		}

		tap(device, returnKeyIndex)
		tap(device, besideKeyIndex)
		assertPresses(t, running, "in app", "first")

		if launched := running.launches.Launched(); len(launched) != 1 {
			t.Fatalf("want the return tile to launch nothing, got %v", launched)
		}

		running.stop(t)
	})
}

// TestDaemon_PagesThroughRegularPagesOnly: back and forward never reach an app
// screen, and on one they leave it for the regular pages either side.
func TestDaemon_PagesThroughRegularPagesOnly(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startAppScreenHarness(t)

		tap(device, besideKeyIndex) // First
		tap(device, forwardKey)
		tap(device, besideKeyIndex) // Second
		tap(device, forwardKey)
		tap(device, besideKeyIndex) // First again, not the app screen
		tap(device, backKey)
		tap(device, besideKeyIndex) // Second, wrapping back

		tap(device, forwardKey)
		tap(device, appKeyIndex)
		tap(device, forwardKey)
		tap(device, besideKeyIndex) // the app screen left forward, for Second

		tap(device, backKey)
		tap(device, appKeyIndex)
		tap(device, backKey)
		tap(device, besideKeyIndex) // the app screen left back, for Second

		assertPresses(t, running, "first", "second", "first", "second", "second", "second")

		running.stop(t)
	})
}

// TestDaemon_KeepsTheAppScreenThroughAReload: a reload keeps the app screen
// showing while the application still has one, and returns to the regular
// pages when it does not.
func TestDaemon_KeepsTheAppScreenThroughAReload(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startAppScreenHarness(t)

		tap(device, appKeyIndex)

		running.configurations <- appScreenConfiguration()

		time.Sleep(settle)
		synctest.Wait()
		tap(device, besideKeyIndex) // still the app screen

		// Paging still counts the two regular pages only.
		tap(device, forwardKey)
		tap(device, besideKeyIndex)
		tap(device, forwardKey)
		tap(device, besideKeyIndex)
		tap(device, appKeyIndex)

		withoutScreen := appScreenConfiguration()
		withoutScreen.Pages = withoutScreen.Pages[1:]
		withoutScreen.Pages[0].Keys[0] = appScreenKey(false)

		running.configurations <- withoutScreen

		time.Sleep(settle)
		synctest.Wait()
		tap(device, besideKeyIndex)
		tap(device, appKeyIndex)

		assertPresses(t, running, "in app", "second", "first", "first")

		// The top left key is the app key again, not a return tile.
		if launched := running.launches.Launched(); len(launched) != 3 {
			t.Fatalf("want the app key pressable again after the reload, got launches %v", launched)
		}

		if strings.Contains(running.logs.String(), "does not fit") {
			t.Fatalf("want every page to fit, got:\n%s", running.logs.String())
		}

		running.stop(t)
	})
}

// keystrokeConfiguration is an app screen for an application, holding a key
// that sends key presses and one that does not, opened from a regular page
// that has a key sending key presses of its own.
func keystrokeConfiguration(application string) config.Configuration {
	return config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion,
		Pages: []config.Page{
			{Name: firstPageName, Application: "", Keys: []config.Key{
				appKeyFor(application, true), keystrokeButton("elsewhere", besideKeyIndex+1),
			}},
			{Name: "Screen", Application: application, Keys: []config.Key{
				keystrokeButton("typed", besideKeyIndex), button("plain", besideKeyIndex+1, false),
			}},
		},
	}
}

// TestDaemon_BringsTheApplicationForward_BeforeAKeystrokeOnItsScreen: a key
// that sends key presses on an app screen brings the application to the front
// first, so they reach it; other keys, and keys on regular pages, do not.
func TestDaemon_BringsTheApplicationForward_BeforeAKeystrokeOnItsScreen(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- keystrokeConfiguration(knownApplication)

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, besideKeyIndex+1)
		tap(device, appKeyIndex)
		tap(device, besideKeyIndex)
		tap(device, besideKeyIndex+1)

		assertPresses(t, running, "elsewhere", "forward "+knownApplication, "typed", "plain")

		running.stop(t)
	})
}

// TestDaemon_SendsNoKeystroke_WhenTheApplicationIsNotOpen: when the
// application cannot be brought forward, the key presses are not sent to
// whatever is in front; the key shows why.
func TestDaemon_SendsNoKeystroke_WhenTheApplicationIsNotOpen(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- keystrokeConfiguration("com.example.closed")

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, appKeyIndex)
		tap(device, besideKeyIndex)

		assertPresses(t, running)

		if !strings.Contains(running.logs.String(), `"reason":"not open"`) {
			t.Fatalf("want the key's error logged as not open, got:\n%s", running.logs.String())
		}

		running.stop(t)
	})
}

// TestDaemon_LogsASlowPress: a press slow enough to be felt is logged with how
// long it took, so a slow key can be traced; a quick one is not.
func TestDaemon_LogsASlowPress(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- keystrokeConfiguration(slowApplication)

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, besideKeyIndex+1)

		if strings.Contains(running.logs.String(), "slow key press") {
			t.Fatalf("want a quick press left out of the log, got:\n%s", running.logs.String())
		}

		tap(device, appKeyIndex)
		tap(device, besideKeyIndex)

		wanted := `"msg":"slow key press","kind":"test_keystroke_button","took":300000000`
		if !strings.Contains(running.logs.String(), wanted) {
			t.Fatalf("want the slow press logged with its time, got:\n%s", running.logs.String())
		}

		running.stop(t)
	})
}
