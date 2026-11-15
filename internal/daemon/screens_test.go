package daemon_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The second screen's name, for its paging tiles.
const secondScreenName = "Blue 2"

// twoScreenConfiguration gives the blue application two screens, the second
// later in the file than the regular pages, opened from an app key on First.
func twoScreenConfiguration() config.Configuration {
	return config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion,
		WorldClocks:   nil,
		Pages: []config.Page{
			{Name: appIconTestLabel, Application: knownApplication, Keys: []config.Key{button("screen one", 1, false)}},
			{Name: firstPageName, Application: "", Keys: []config.Key{appScreenKey(true), button("first", 1, false)}},
			{Name: secondPageName, Application: "", Keys: []config.Key{button("second", 1, false)}},
			{Name: secondScreenName, Application: knownApplication, Keys: []config.Key{button("screen two", 1, false)}},
		},
	}
}

func startTwoScreenHarness(t *testing.T) (*harness, *testsupport.DeckDeviceTestDouble) {
	t.Helper()

	running := startHarness(t)
	running.configurations <- twoScreenConfiguration()

	device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
	running.watcher.Attach("deck-a", device)
	time.Sleep(settle)
	synctest.Wait()

	return running, device
}

// TestDaemon_PagesBetweenAnApplicationsScreens: with two screens, holding the
// app key opens the first; back and forward page between the screens,
// wrapping round, rather than leaving them; and the return tile leaves for the
// regular page.
func TestDaemon_PagesBetweenAnApplicationsScreens(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startTwoScreenHarness(t)

		holdAppKey(device)
		tap(device, besideKeyIndex) // screen one
		tap(device, forwardKey)
		tap(device, besideKeyIndex) // screen two
		tap(device, forwardKey)
		tap(device, besideKeyIndex) // screen one again, wrapped
		tap(device, backKey)
		tap(device, besideKeyIndex) // screen two, wrapped back
		tap(device, returnKeyIndex)
		tap(device, besideKeyIndex) // First

		holdAppKey(device)
		tap(device, besideKeyIndex) // the first screen again, not the last shown

		assertPresses(t, running, "screen one", "screen two", "screen one", "screen two", "first", "screen one")

		running.stop(t)
	})
}

// TestDaemon_NumbersAnApplicationsScreensOnItsPagingTiles: on one of two
// screens, the paging tiles show which screen it is and the other's name, as
// a regular page's show the pages either side.
func TestDaemon_NumbersAnApplicationsScreensOnItsPagingTiles(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startTwoScreenHarness(t)

		holdAppKey(device)

		onScreen := layout.PageIndication{
			Number: 1, Count: 2, CurrentName: appIconTestLabel, PreviousName: secondScreenName, NextName: secondScreenName,
		}
		screens := running.renderer.RenderNavigation(render.DirectionForward, onScreen, true)
		regular := running.renderer.RenderNavigation(render.DirectionForward, layout.PageIndication{
			Number: 1, Count: 2, CurrentName: firstPageName, PreviousName: secondPageName, NextName: secondPageName,
		}, true)

		drawn := drawnImage(t, device, forwardKey)
		if distance(drawn, screens) >= distance(drawn, regular) {
			t.Fatal("want the forward tile numbering the screens, not the regular pages")
		}

		running.stop(t)
	})
}

// TestDaemon_KeepsTheScreenShowingThroughAReload: a reload keeps the second
// screen showing while it is still there, and shows the first when it is gone.
func TestDaemon_KeepsTheScreenShowingThroughAReload(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startTwoScreenHarness(t)

		holdAppKey(device)
		tap(device, forwardKey)

		running.configurations <- twoScreenConfiguration()

		time.Sleep(settle)
		synctest.Wait()
		tap(device, besideKeyIndex) // still screen two

		withoutSecond := twoScreenConfiguration()
		withoutSecond.Pages = withoutSecond.Pages[:3]

		running.configurations <- withoutSecond

		time.Sleep(settle)
		synctest.Wait()
		tap(device, besideKeyIndex) // screen one, the only screen left

		assertPresses(t, running, "screen two", "screen one")

		running.stop(t)
	})
}
