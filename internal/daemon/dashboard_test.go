package daemon_test

import (
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/bricks/pagemarker"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The dashboard's keys on the XL, eight to a row: the way back at the top
// left, a row of pages, a row of app screens, and the paging tiles in the
// bottom corners.
const (
	markerKeyIndex   = 0
	dashboardPageRow = 8
	dashboardAppRow  = 16
	clockKeyIndex    = 3
	pageOneIcon      = "number-1"
	pageTwoIcon      = "number-2"
)

// markerKey is a page marker in the top left corner.
func markerKey() config.Key {
	return config.Key{
		Position: layout.Position{Row: 0, Column: 0}, Kind: pagemarker.KindName,
		Parameters: &pagemarker.Parameters{Appearance: noAppearance(), PageNumber: true}, Preset: nil,
	}
}

// dashboardConfiguration has First, with its marker, and Second, then the
// blue application's two screens, each key beside the corner naming its page.
func dashboardConfiguration(screens int) config.Configuration {
	pages := make([]config.Page, 0, 2+screens)
	pages = append(pages,
		config.Page{Name: firstPageName, Application: "", Keys: []config.Key{markerKey(), button("first", 1, false)}},
		config.Page{Name: secondPageName, Application: "", Keys: []config.Key{button("second", 1, false)}},
	)

	for screen := range screens {
		name := fmt.Sprintf("Blue %d", screen+1)
		pages = append(pages, config.Page{
			Name: name, Application: knownApplication, Keys: []config.Key{button("screen "+name, 1, false)},
		})
	}

	return config.Configuration{SchemaVersion: config.CurrentSchemaVersion, Pages: pages}
}

func startDashboardHarness(t *testing.T, screens int) (*harness, *testsupport.DeckDeviceTestDouble) {
	t.Helper()

	running := startHarness(t)
	running.configurations <- dashboardConfiguration(screens)

	return running, deckAttached(t, running)
}

// holdMarker holds the page marker past the hold that opens the dashboard.
func holdMarker(device *testsupport.DeckDeviceTestDouble) {
	device.PressKeys(markerKeyIndex)
	time.Sleep(time.Second)
	synctest.Wait()
	device.PressKeys()
	time.Sleep(settle)
	synctest.Wait()
}

// TestDaemon_DashboardGoesToThePageOrScreenPressed: holding a page marker
// opens the dashboard; a page pressed in its second row, or an app screen in
// its third, is shown.
func TestDaemon_DashboardGoesToThePageOrScreenPressed(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)

		holdMarker(device)
		tap(device, dashboardPageRow+1) // Second
		tap(device, besideKeyIndex)

		holdMarker(device) // Second has no marker, so nothing opens
		tap(device, besideKeyIndex)

		tap(device, backKey) // First again
		holdMarker(device)
		tap(device, dashboardAppRow+1) // Blue 2
		tap(device, besideKeyIndex)

		assertPresses(t, running, "second", "second", "screen Blue 2")

		running.stop(t)
	})
}

// TestDaemon_DashboardTopLeftGoesBack: the dashboard's top left key goes
// back to the page it was opened from; a clock key or an empty one does
// nothing.
func TestDaemon_DashboardTopLeftGoesBack(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)

		holdMarker(device)
		tap(device, clockKeyIndex)
		tap(device, dashboardPageRow+5)
		tap(device, markerKeyIndex)
		tap(device, besideKeyIndex)

		assertPresses(t, running, "first")

		running.stop(t)
	})
}

// TestDaemon_TappingAPageMarkerDoesNothing: a marker tapped, not held, leaves
// the page as it is.
func TestDaemon_TappingAPageMarkerDoesNothing(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)

		tap(device, markerKeyIndex)
		tap(device, besideKeyIndex)

		assertPresses(t, running, "first")

		running.stop(t)
	})
}

// TestDaemon_DashboardPagesThroughManyAppScreens: with more app screens than
// a row holds, the paging tiles move the row along by a row's worth,
// wrapping round.
func TestDaemon_DashboardPagesThroughManyAppScreens(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 10)

		holdMarker(device)
		tap(device, forwardKey)
		tap(device, dashboardAppRow+1) // the tenth screen
		tap(device, besideKeyIndex)

		tap(device, returnKeyIndex) // back to First
		holdMarker(device)
		tap(device, backKey) // wrapping round to the second row's worth
		tap(device, dashboardAppRow)
		tap(device, besideKeyIndex)

		got := running.recorder.Presses()
		if !slices.Equal(got, []string{"screen Blue 10", "screen Blue 9"}) {
			t.Fatalf("want the tenth screen and then the ninth, got %v", got)
		}

		running.stop(t)
	})
}

// TestDaemon_DashboardDrawsPagesAndScreens: the dashboard draws the marker
// held in the top left, each page by its number and name, the page opened
// from marked, each screen by its application's icon and name, and the keys
// with nothing to show blank.
func TestDaemon_DashboardDrawsPagesAndScreens(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)

		holdMarker(device)

		// Each key is drawn closer to what it should be than to the nearest
		// thing it should not: the page held from with its mark, the next
		// without, and the marker as it shows on its page. The deck's pictures
		// are compressed, so an exact match is not to be had.
		marked, _ := render.ParseColour("#4b2050")
		for _, check := range []struct {
			keyIndex int
			want     render.Appearance
			not      render.Appearance
		}{
			{markerKeyIndex, render.Appearance{Label: "Page 1", Icon: pageOneIcon}, render.Appearance{Label: "First"}},
			{
				dashboardPageRow,
				render.Appearance{Label: firstPageName, Icon: pageOneIcon, Background: marked},
				render.Appearance{Label: firstPageName, Icon: pageOneIcon},
			},
			{
				dashboardPageRow + 1,
				render.Appearance{Label: secondPageName, Icon: pageTwoIcon},
				render.Appearance{Label: secondPageName, Icon: pageTwoIcon, Background: marked},
			},
		} {
			drawn := drawnImage(t, device, check.keyIndex)
			wanted, unwanted := running.renderer.Render(check.want), running.renderer.Render(check.not)

			if distance(drawn, wanted) >= distance(drawn, unwanted) {
				t.Errorf("key %d: want it drawn as %+v", check.keyIndex, check.want)
			}
		}

		blank := running.renderer.RenderBlank()
		for _, keyIndex := range []int{dashboardPageRow + 5, dashboardAppRow + 4} {
			if difference := distance(drawnImage(t, device, keyIndex), blank); difference > blankDifference {
				t.Errorf("key %d: want it blank, differing by %.1f", keyIndex, difference)
			}
		}

		running.stop(t)
	})
}

// TestDaemon_AReloadClosesTheDashboard: a new configuration closes the
// dashboard, back on the page it was opened from.
func TestDaemon_AReloadClosesTheDashboard(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)

		holdMarker(device)

		running.configurations <- dashboardConfiguration(2)

		time.Sleep(settle)
		synctest.Wait()
		tap(device, besideKeyIndex)

		assertPresses(t, running, "first")

		running.stop(t)
	})
}
