package daemon

import (
	"context"
	"image"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The page dashboard: holding a page marker, as long as an app key is held
// for its screen, puts the page aside for a view of every page on the deck
// itself. Its top left key is the marker held, and goes back to the page;
// the rest of the top row is kept for a clock to come. The second row holds
// the regular pages, each by its number and name, the page held from marked;
// the third the app screens, each by its application's icon and name, a
// row's worth at a time; and the bottom corners page that row along when
// there are more. Pressing a page or screen shows it. The dashboard stays
// until a key is pressed, and a new configuration or a deck let go closes it.
//
// What each row holds is a list, so a row can be given more to show without
// the rest changing; a key a row does not reach is empty.

// dashboardRow is what a row of the dashboard holds.
type dashboardRow int

const (
	dashboardRowEmpty dashboardRow = iota
	// dashboardRowClock: the way back in its first key, the rest kept for
	// the clock.
	dashboardRowClock
	// dashboardRowPages: the regular pages, from the first.
	dashboardRowPages
	// dashboardRowScreens: the app screens, a row's worth from where the
	// paging tiles have moved it.
	dashboardRowScreens
	// dashboardRowPaging: the paging tiles in their corners, when there are
	// more screens than a row holds.
	dashboardRowPaging
)

// dashboardRows is the dashboard's rows, top to bottom; a deck with fewer
// rows shows the first of them.
func dashboardRows() []dashboardRow {
	return []dashboardRow{dashboardRowClock, dashboardRowPages, dashboardRowScreens, dashboardRowPaging}
}

// dashboardRole is what one key of the dashboard does.
type dashboardRole int

const (
	dashboardEmpty dashboardRole = iota
	dashboardBack
	dashboardReserved
	dashboardPage
	dashboardScreen
	dashboardPrevious
	dashboardNext
)

// dashboardTile is one key of the dashboard: what it does, and for a page or
// screen, its index in pages.
type dashboardTile struct {
	role      dashboardRole
	pageIndex int
}

// dashboardState is the dashboard while it is open: the page its marker was
// held on and the marker's place among that page's keys, and how far the
// screens row has been paged along.
type dashboardState struct {
	open         bool
	fromPage     int
	markerBrick  int
	screenOffset int
}

// dashboardMarked is the background of the page the dashboard was opened
// from, the editor's own highlight.
const dashboardMarked = "#4b2050"

// holdsLong reports whether a key waits to see whether it is held: an app
// key that shows its screen when held, or a page marker, which opens the
// dashboard.
func (state *loopState) holdsLong(pageIndex int, brickIndex int) bool {
	_, isMarker := state.pages[pageIndex][brickIndex].instance.(brick.PageMarker)

	return isMarker || state.holdsForScreen(pageIndex, brickIndex)
}

// openDashboard shows the dashboard for a marker held on a page.
func (state *loopState) openDashboard(runContext context.Context, pageIndex int, brickIndex int) {
	state.dashboard = dashboardState{open: true, fromPage: pageIndex, markerBrick: brickIndex, screenOffset: 0}
	state.repaintAll(runContext)
}

// closeDashboard puts the page back.
func (state *loopState) closeDashboard() {
	state.dashboard = dashboardState{open: false, fromPage: 0, markerBrick: 0, screenOffset: 0}
}

// dashboardTiles is the dashboard, by key index.
func (state *loopState) dashboardTiles() []dashboardTile {
	tiles := make([]dashboardTile, state.grid.KeyCount())
	columns := state.grid.Columns()

	for row, holds := range dashboardRows() {
		if row >= state.grid.Rows() {
			break
		}

		for column := range columns {
			position := layout.Position{Row: row, Column: column}
			if keyIndex, onGrid := state.grid.KeyIndex(position); onGrid {
				tiles[keyIndex] = state.dashboardTileAt(holds, position)
			}
		}
	}

	return tiles
}

// dashboardTileAt is what a key holds in a row.
func (state *loopState) dashboardTileAt(holds dashboardRow, position layout.Position) dashboardTile {
	switch holds {
	case dashboardRowClock:
		return state.clockRowTile(position)
	case dashboardRowPages:
		return state.pagesRowTile(position)
	case dashboardRowScreens:
		return state.screensRowTile(position)
	case dashboardRowPaging:
		return state.pagingRowTile(position)
	case dashboardRowEmpty:
	}

	return dashboardTile{role: dashboardEmpty, pageIndex: 0}
}

// clockRowTile: the way back in the first key, the rest kept for the clock.
func (state *loopState) clockRowTile(position layout.Position) dashboardTile {
	if position.Column == 0 {
		return dashboardTile{role: dashboardBack, pageIndex: state.dashboard.fromPage}
	}

	return dashboardTile{role: dashboardReserved, pageIndex: 0}
}

// pagesRowTile: the regular pages, from the first.
func (state *loopState) pagesRowTile(position layout.Position) dashboardTile {
	if position.Column < len(state.regular) {
		return dashboardTile{role: dashboardPage, pageIndex: state.regular[position.Column]}
	}

	return dashboardTile{role: dashboardEmpty, pageIndex: 0}
}

// screensRowTile: the app screens, from where the row has been paged to.
func (state *loopState) screensRowTile(position layout.Position) dashboardTile {
	if at := state.dashboard.screenOffset + position.Column; at < len(state.screenPages) {
		return dashboardTile{role: dashboardScreen, pageIndex: state.screenPages[at]}
	}

	return dashboardTile{role: dashboardEmpty, pageIndex: 0}
}

// pagingRowTile: the paging tiles in their corners, when there are more
// screens than a row holds.
func (state *loopState) pagingRowTile(position layout.Position) dashboardTile {
	if len(state.screenPages) > state.grid.Columns() {
		switch position {
		case state.grid.BackPosition():
			return dashboardTile{role: dashboardPrevious, pageIndex: 0}
		case state.grid.ForwardPosition():
			return dashboardTile{role: dashboardNext, pageIndex: 0}
		}
	}

	return dashboardTile{role: dashboardEmpty, pageIndex: 0}
}

// paintDashboard draws every key of the dashboard.
func (state *loopState) paintDashboard(runContext context.Context) {
	for keyIndex, tile := range state.dashboardTiles() {
		writeError := state.connected.SetKeyImage(keyIndex, state.dashboardImage(runContext, tile))
		if writeError != nil {
			state.disconnect(runContext, "write failed")

			return
		}
	}
}

// dashboardImage draws one key of the dashboard.
func (state *loopState) dashboardImage(runContext context.Context, tile dashboardTile) image.Image {
	renderer := state.options.Renderer

	switch tile.role {
	case dashboardBack:
		return renderer.Render(state.appearanceFor(runContext,
			state.pages[state.dashboard.fromPage][state.dashboard.markerBrick]))
	case dashboardPage:
		number := slices.Index(state.regular, tile.pageIndex) + 1

		return renderer.Render(state.markedFrom(tile.pageIndex, render.Appearance{
			Label: state.allPageNames[tile.pageIndex], Icon: brick.PageMarkerIcon(number),
		}))
	case dashboardScreen:
		application, _ := state.applicationOf(tile.pageIndex)
		icon := state.appIcon(runContext, application)

		return renderer.Render(state.markedFrom(tile.pageIndex, render.Appearance{
			Label: state.allPageNames[tile.pageIndex], IconImage: icon, IconMissing: icon == nil,
			BackgroundFromIcon: true,
		}))
	case dashboardPrevious, dashboardNext:
		direction := render.DirectionForward
		if tile.role == dashboardPrevious {
			direction = render.DirectionBack
		}

		columns := state.grid.Columns()

		return renderer.RenderNavigation(direction, layout.PageIndication{
			Number: state.dashboard.screenOffset/columns + 1, Count: (len(state.screenPages) + columns - 1) / columns,
			CurrentName: "", PreviousName: "", NextName: "",
		}, true)
	case dashboardEmpty, dashboardReserved:
	}

	return renderer.RenderBlank()
}

// markedFrom gives the page the dashboard was opened from its mark.
func (state *loopState) markedFrom(pageIndex int, appearance render.Appearance) render.Appearance {
	if pageIndex == state.dashboard.fromPage {
		appearance.Background, _ = render.ParseColour(dashboardMarked)
	}

	return appearance
}

// pressDashboard does what a dashboard key does: back puts the page back, a
// page or screen is shown, and the paging tiles move the screens along.
func (state *loopState) pressDashboard(runContext context.Context, keyIndex int) {
	tiles := state.dashboardTiles()
	if keyIndex < 0 || keyIndex >= len(tiles) {
		return
	}

	tile := tiles[keyIndex]

	switch tile.role {
	case dashboardBack:
		state.closeDashboard()
	case dashboardPage:
		state.appScreen = ""
		state.paging = state.paging.To(slices.Index(state.regular, tile.pageIndex))
		state.closeDashboard()
		state.rememberPage()
	case dashboardScreen:
		application, _ := state.applicationOf(tile.pageIndex)
		state.appScreen, state.appScreenAt = application, slices.Index(state.appScreens[application], tile.pageIndex)
		state.closeDashboard()
	case dashboardPrevious, dashboardNext:
		step := 1
		if tile.role == dashboardPrevious {
			step = -1
		}

		columns := state.grid.Columns()
		rows := (len(state.screenPages) + columns - 1) / columns
		state.dashboard.screenOffset = ((state.dashboard.screenOffset/columns + step + rows) % rows) * columns
	case dashboardEmpty, dashboardReserved:
		return
	}

	state.repaintAll(runContext)
}
