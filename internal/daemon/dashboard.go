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
// the rest of the top row is the clock, and its world icon the world clocks
// (clock.go). The second row holds
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
	// dashboardRowClock: the way back in its first key, the local time across
	// the next six, and the world icon in its last.
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
	dashboardDigit
	dashboardZoneLabel
	dashboardWorld
	dashboardDate
)

// dashboardView is which of the dashboard's views is showing: its pages, or
// the world clocks.
type dashboardView int

const (
	dashboardPagesView dashboardView = iota
	dashboardWorldView
)

// localZone is the zone of a clock key showing the local time.
const localZone = -1

// clockFigures is how many keys a time takes, HHMMSS.
const clockFigures = 6

// dashboardTile is one key of the dashboard: what it does; for a page or
// screen, its index in pages; and for a clock key, its zone, localZone for
// the local time, and which of the time's figures it shows.
type dashboardTile struct {
	role      dashboardRole
	pageIndex int
	zone      int
	figure    int
}

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
	view         dashboardView
	stop         chan struct{}
	drawn        map[int]string
}

// tile makes a dashboard key that needs no page or clock.
func tile(role dashboardRole) dashboardTile {
	return dashboardTile{role: role, pageIndex: 0, zone: localZone, figure: 0}
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
	state.closeDashboard()
	state.dashboard = dashboardState{
		open: true, fromPage: pageIndex, markerBrick: brickIndex, screenOffset: 0, view: dashboardPagesView,
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
		open: false, fromPage: 0, markerBrick: 0, screenOffset: 0, view: dashboardPagesView, stop: nil, drawn: nil,
	}
}

// dashboardTiles is the dashboard, by key index: its pages, or the world
// clocks.
func (state *loopState) dashboardTiles() []dashboardTile {
	if state.dashboard.view == dashboardWorldView {
		return state.worldTiles()
	}

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

	return tile(dashboardEmpty)
}

// clockRowTile: the way back in the first key, the local time across the
// next six, and the world icon in the last.
func (state *loopState) clockRowTile(position layout.Position) dashboardTile {
	switch {
	case position.Column == 0:
		return dashboardTile{role: dashboardBack, pageIndex: state.dashboard.fromPage, zone: localZone, figure: 0}
	case position.Column <= clockFigures:
		return dashboardTile{role: dashboardDigit, pageIndex: 0, zone: localZone, figure: position.Column - 1}
	case position.Column == state.grid.Columns()-1:
		return tile(dashboardWorld)
	default:
		return tile(dashboardReserved)
	}
}

// worldTiles is the world clocks: a row each, its label and then its HHMMSS,
// the world icon in the top right and the local date, YY, MM and DD, down the
// right column beneath it.
func (state *loopState) worldTiles() []dashboardTile {
	tiles := make([]dashboardTile, state.grid.KeyCount())
	last := state.grid.Columns() - 1

	for keyIndex := range tiles {
		tiles[keyIndex] = tile(dashboardEmpty)
	}

	for row := range min(state.grid.Rows(), len(state.zones)) {
		for column := range state.grid.Columns() {
			if keyIndex, onGrid := state.grid.KeyIndex(layout.Position{Row: row, Column: column}); onGrid {
				tiles[keyIndex] = worldTileAt(row, column, last)
			}
		}
	}

	return tiles
}

// worldTileAt is what a key of the world clocks holds: its row's label, a
// figure of its time, the world icon, or a part of the date.
func worldTileAt(row int, column int, last int) dashboardTile {
	switch {
	case column == 0:
		return dashboardTile{role: dashboardZoneLabel, pageIndex: 0, zone: row, figure: 0}
	case column <= clockFigures:
		return dashboardTile{role: dashboardDigit, pageIndex: 0, zone: row, figure: column - 1}
	case row == 0 && column == last:
		return tile(dashboardWorld)
	case column == last && row <= len(dateParts()):
		return dashboardTile{role: dashboardDate, pageIndex: 0, zone: localZone, figure: row - 1}
	default:
		return tile(dashboardEmpty)
	}
}

// pagesRowTile: the regular pages, from the first.
func (state *loopState) pagesRowTile(position layout.Position) dashboardTile {
	if position.Column < len(state.regular) {
		return dashboardTile{role: dashboardPage, pageIndex: state.regular[position.Column], zone: localZone, figure: 0}
	}

	return tile(dashboardEmpty)
}

// screensRowTile: the app screens, from where the row has been paged to.
func (state *loopState) screensRowTile(position layout.Position) dashboardTile {
	if at := state.dashboard.screenOffset + position.Column; at < len(state.screenPages) {
		return dashboardTile{role: dashboardScreen, pageIndex: state.screenPages[at], zone: localZone, figure: 0}
	}

	return tile(dashboardEmpty)
}

// pagingRowTile: the paging tiles in their corners, when there are more
// screens than a row holds.
func (state *loopState) pagingRowTile(position layout.Position) dashboardTile {
	if len(state.screenPages) > state.grid.Columns() {
		switch position {
		case state.grid.BackPosition():
			return tile(dashboardPrevious)
		case state.grid.ForwardPosition():
			return tile(dashboardNext)
		}
	}

	return tile(dashboardEmpty)
}

// paintDashboard draws every key of the dashboard, noting each clock key's
// figure.
func (state *loopState) paintDashboard(runContext context.Context) {
	state.dashboard.drawn = map[int]string{}

	for keyIndex, shown := range state.dashboardTiles() {
		writeError := state.connected.SetKeyImage(keyIndex, state.dashboardImage(runContext, shown))
		if writeError != nil {
			state.disconnect(runContext, "write failed")

			return
		}

		if ticks(shown) {
			state.dashboard.drawn[keyIndex] = state.figureOf(shown)
		}
	}
}

// dashboardImage draws one key of the dashboard.
func (state *loopState) dashboardImage(runContext context.Context, shown dashboardTile) image.Image {
	renderer := state.options.Renderer

	switch shown.role {
	case dashboardBack:
		return renderer.Render(state.appearanceFor(runContext,
			state.pages[state.dashboard.fromPage][state.dashboard.markerBrick]))
	case dashboardPage:
		number := slices.Index(state.regular, shown.pageIndex) + 1

		return renderer.Render(state.markedFrom(shown.pageIndex, render.Appearance{
			Label: state.allPageNames[shown.pageIndex], Icon: brick.PageMarkerIcon(number),
		}))
	case dashboardScreen:
		application, _ := state.applicationOf(shown.pageIndex)
		icon := state.appIcon(runContext, application)

		return renderer.Render(state.markedFrom(shown.pageIndex, render.Appearance{
			Label: state.allPageNames[shown.pageIndex], IconImage: icon, IconMissing: icon == nil,
			BackgroundFromIcon: true,
		}))
	case dashboardPrevious, dashboardNext:
		direction := render.DirectionForward
		if shown.role == dashboardPrevious {
			direction = render.DirectionBack
		}

		columns := state.grid.Columns()

		return renderer.RenderNavigation(direction, layout.PageIndication{
			Number: state.dashboard.screenOffset/columns + 1, Count: (len(state.screenPages) + columns - 1) / columns,
			CurrentName: "", PreviousName: "", NextName: "",
		}, true)
	case dashboardDigit, dashboardZoneLabel, dashboardWorld, dashboardDate:
		return state.clockKeyImage(shown)
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

	pressed := tiles[keyIndex]

	switch pressed.role {
	case dashboardBack:
		state.closeDashboard()
	case dashboardPage:
		state.appScreen = ""
		state.paging = state.paging.To(slices.Index(state.regular, pressed.pageIndex))
		state.closeDashboard()
		state.rememberPage()
	case dashboardScreen:
		application, _ := state.applicationOf(pressed.pageIndex)
		state.appScreen, state.appScreenAt = application, slices.Index(state.appScreens[application], pressed.pageIndex)
		state.closeDashboard()
	case dashboardPrevious, dashboardNext:
		step := 1
		if pressed.role == dashboardPrevious {
			step = -1
		}

		columns := state.grid.Columns()
		rows := (len(state.screenPages) + columns - 1) / columns
		state.dashboard.screenOffset = ((state.dashboard.screenOffset/columns + step + rows) % rows) * columns
	case dashboardWorld:
		state.dashboard.view = dashboardWorldView - state.dashboard.view
	case dashboardEmpty, dashboardReserved, dashboardDigit, dashboardZoneLabel, dashboardDate:
		return
	}

	state.repaintAll(runContext)
}
