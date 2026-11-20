package daemon_test

import (
	"image"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// paleAmber is a navigation colour far from the usual black, written as the
// configuration names it.
const paleAmber = "#FFE066"

// requireNavigationIn checks a navigation key is drawn nearer to itself on one
// background than on the other. The deck's pictures are compressed, so an
// exact match is not to be had.
func requireNavigationIn(
	t *testing.T, device *testsupport.DeckDeviceTestDouble, keyIndex int, want image.Image, not image.Image,
) {
	t.Helper()

	drawn := drawnImage(t, device, keyIndex)
	if distance(drawn, want) >= distance(drawn, not) {
		t.Errorf("key %d: want it drawn on the other background", keyIndex)
	}
}

// TestDaemon_RepaintsTheNavigationKeysInTheirColour: the navigation keys
// start usual, and a reload naming a colour for them repaints both in it.
func TestDaemon_RepaintsTheNavigationKeysInTheirColour(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := deckAttached(t, running)

		amber, usual := render.NavigationBackgroundFrom(paleAmber), render.DefaultNavigationBackground()
		onFirst := layout.PageIndication{
			Number: 1, Count: 2, CurrentName: firstPageName, PreviousName: secondPageName, NextName: secondPageName,
		}

		for keyIndex, direction := range map[int]render.Direction{
			backKey: render.DirectionBack, forwardKey: render.DirectionForward,
		} {
			requireNavigationIn(t, device, keyIndex, running.renderer.RenderNavigation(direction, onFirst, true, usual),
				running.renderer.RenderNavigation(direction, onFirst, true, amber))
		}

		coloured := twoPages()
		coloured.NavigationBackground = paleAmber

		running.configurations <- coloured

		time.Sleep(settle)
		synctest.Wait()

		for keyIndex, direction := range map[int]render.Direction{
			backKey: render.DirectionBack, forwardKey: render.DirectionForward,
		} {
			requireNavigationIn(t, device, keyIndex, running.renderer.RenderNavigation(direction, onFirst, true, amber),
				running.renderer.RenderNavigation(direction, onFirst, true, usual))
		}

		running.stop(t)
	})
}

// TestDaemon_DashboardPagingTilesTakeTheNavigationColour: the dashboard's
// paging tiles, in the same corners, are drawn in the navigation keys' colour.
func TestDaemon_DashboardPagingTilesTakeTheNavigationColour(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		configuration := dashboardConfiguration(10)
		configuration.NavigationBackground = paleAmber

		running.configurations <- configuration

		device := deckAttached(t, running)

		holdMarker(device)

		rows := layout.PageIndication{Number: 1, Count: 2, CurrentName: "", PreviousName: "", NextName: ""}
		amber, usual := render.NavigationBackgroundFrom(paleAmber), render.DefaultNavigationBackground()

		requireNavigationIn(t, device, forwardKey,
			running.renderer.RenderNavigation(render.DirectionForward, rows, true, amber),
			running.renderer.RenderNavigation(render.DirectionForward, rows, true, usual))

		running.stop(t)
	})
}
