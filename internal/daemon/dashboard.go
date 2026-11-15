package daemon

import (
	"context"
	"image"
	"slices"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/dashboard"
)

// The page dashboard on the deck: holding a page marker, as long as an app key
// is held for its screen, opens it in place of the page, laid out and drawn by
// the dashboard package. Pressing a page or screen shows it, the top left key
// goes back, the paging tiles move the screens along and the world icon
// switches between the pages and the world clocks. The dashboard stays until a
// key is pressed, and a new configuration or a deck let go closes it.

// dashboardState is the dashboard while it is open: the page its marker was
// held on and the marker's place among that page's keys, how far the screens
// row has been paged along, which view is showing, the clock's ticking, to be
// stopped, and the figure drawn on each clock key, so a tick draws only
// those that change.
type dashboardState struct {
	open         bool
	fromPage     int
	markerBrick  int
	screenOffset int
	view         dashboard.View
	stop         chan struct{}
	drawn        map[int]string
}

// holdsLong reports whether a key waits to see whether it is held: an app
// key that shows its screen when held, or a page marker, which opens the
// dashboard.
func (state *loopState) holdsLong(pageIndex int, brickIndex int) bool {
	_, isMarker := state.pages[pageIndex][brickIndex].instance.(brick.PageMarker)

	return isMarker || state.holdsForScreen(pageIndex, brickIndex)
}

// openDashboard shows the dashboard for a marker held on a page.
func (state *loopState) openDashboard(runContext context.Context, pageIndex int, brickIndex int) {
	state.closeDashboard()
	state.dashboard = dashboardState{
		open: true, fromPage: pageIndex, markerBrick: brickIndex, screenOffset: 0, view: dashboard.PagesView,
		stop: state.startClock(runContext), drawn: map[int]string{},
	}
	state.repaintAll(runContext)
}

// closeDashboard puts the page back and stops the clock.
func (state *loopState) closeDashboard() {
	if state.dashboard.stop != nil {
		close(state.dashboard.stop)
	}

	state.dashboard = closedDashboard()
}

// closedDashboard is the dashboard while it is not open.
func closedDashboard() dashboardState {
	return dashboardState{
		open: false, fromPage: 0, markerBrick: 0, screenOffset: 0, view: dashboard.PagesView, stop: nil, drawn: nil,
	}
}

// scene is the dashboard as it stands on the deck.
func (state *loopState) scene() dashboard.Scene {
	return dashboard.Scene{
		Grid: state.grid, View: state.dashboard.view, FromPage: state.dashboard.fromPage,
		Regular: state.regular, ScreenPages: state.screenPages, ScreenOffset: state.dashboard.screenOffset,
		Zones: state.zones,
	}
}

// look is what the deck draws the dashboard with: the marker held as its page
// draws it, and each app screen's application icon.
func (state *loopState) look(runContext context.Context) dashboard.Look {
	return dashboard.Look{
		Renderer:  state.options.Renderer,
		PageNames: state.allPageNames,
		Marker: func() image.Image {
			return state.options.Renderer.Render(state.appearanceFor(runContext,
				state.pages[state.dashboard.fromPage][state.dashboard.markerBrick]))
		},
		ScreenIcon: func(pageIndex int) image.Image {
			application, _ := state.applicationOf(pageIndex)

			return state.appIcon(runContext, application)
		},
	}
}

// paintDashboard draws every key of the dashboard, noting each clock key's
// figure.
func (state *loopState) paintDashboard(runContext context.Context) {
	state.dashboard.drawn = map[int]string{}
	scene, look, now := state.scene(), state.look(runContext), time.Now()

	for keyIndex, shown := range scene.Tiles() {
		writeError := state.connected.SetKeyImage(keyIndex, scene.Image(look, shown, now))
		if writeError != nil {
			state.disconnect(runContext, "write failed")

			return
		}

		if dashboard.Ticks(shown) {
			state.dashboard.drawn[keyIndex] = scene.Figure(shown, now)
		}
	}
}

// pressDashboard does what a dashboard key does: back puts the page back, a
// page or screen is shown, and the paging tiles move the screens along.
func (state *loopState) pressDashboard(runContext context.Context, keyIndex int) {
	scene := state.scene()

	tiles := scene.Tiles()
	if keyIndex < 0 || keyIndex >= len(tiles) {
		return
	}

	pressed := tiles[keyIndex]

	switch pressed.Role {
	case dashboard.RoleBack:
		state.closeDashboard()
	case dashboard.RolePage:
		state.appScreen = ""
		state.paging = state.paging.To(slices.Index(state.regular, pressed.PageIndex))
		state.closeDashboard()
		state.rememberPage()
	case dashboard.RoleScreen:
		application, _ := state.applicationOf(pressed.PageIndex)
		state.appScreen, state.appScreenAt = application, slices.Index(state.appScreens[application], pressed.PageIndex)
		state.closeDashboard()
	case dashboard.RolePrevious, dashboard.RoleNext:
		step := 1
		if pressed.Role == dashboard.RolePrevious {
			step = -1
		}

		columns, rows := state.grid.Columns(), scene.ScreenRows()
		state.dashboard.screenOffset = ((state.dashboard.screenOffset/columns + step + rows) % rows) * columns
	case dashboard.RoleWorld:
		state.dashboard.view = dashboard.WorldView - state.dashboard.view
	case dashboard.RoleEmpty, dashboard.RoleReserved, dashboard.RoleDigit, dashboard.RoleZoneLabel,
		dashboard.RoleDate:
		return
	}

	state.repaintAll(runContext)
}
