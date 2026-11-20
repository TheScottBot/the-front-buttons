// Package dashboard lays out and draws the page dashboard, the view of every
// page a held page marker opens on the deck, and its world clocks, from what
// it is given, so the deck and the editor's preview draw it alike.
//
// Holding a page marker, as long as an app key is held for its screen, puts
// the page aside for the dashboard. Its top left key is the marker held, and
// goes back to the page; the rest of the top row is the local time, HHMMSS, a
// figure to a key, and its last key the world icon, which shows the world
// clocks. The second row holds the regular pages, each by its number and
// name, the page held from marked; the third the app screens, each by its
// application's icon and name, a row's worth at a time; and the bottom
// corners page that row along when there are more.
//
// The world clocks are a row each from the top, its label and then its
// HHMMSS, the rows past them blank, with the world icon kept in the top right
// to go back and the local date, YY, MM and DD, down the right column beneath
// it.
//
// What each row holds is a list, so a row can be given more to show without
// the rest changing; a key a row does not reach is empty.
package dashboard

import (
	"image"
	"image/color"
	"log/slog"
	"slices"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// Role is what one key of the dashboard does.
type Role int

// The roles a key of the dashboard may have.
const (
	RoleEmpty Role = iota
	RoleBack
	RoleReserved
	RolePage
	RoleScreen
	RolePrevious
	RoleNext
	RoleDigit
	RoleZoneLabel
	RoleWorld
	RoleDate
)

// View is which of the dashboard's views is showing: its pages, or the world
// clocks.
type View int

// The dashboard's views.
const (
	PagesView View = iota
	WorldView
)

// LocalZone is the zone of a clock key showing the local time.
const LocalZone = -1

// ClockFigures is how many keys a time takes, HHMMSS.
const ClockFigures = 6

// MarkedBackground is the background of the page the dashboard was opened
// from, the editor's own highlight.
const MarkedBackground = "#4b2050"

// worldIcon is the bundled icon the world clocks are opened and left by.
const worldIcon = "world"

// timeFigures is how a time is written across its keys, HHMMSS.
const timeFigures = "150405"

// figuresPerPair is how many figures each of the hours, minutes and seconds
// take.
const figuresPerPair = 2

// Tile is one key of the dashboard: what it does; for a page or screen, its
// index in the pages; and for a clock key, its zone, LocalZone for the local
// time, and which of the time's figures, or the date's parts, it shows.
type Tile struct {
	Role      Role
	PageIndex int
	Zone      int
	Figure    int
}

// tile makes a key that needs no page or clock.
func tile(role Role) Tile {
	return Tile{Role: role, PageIndex: 0, Zone: LocalZone, Figure: 0}
}

// Zone is one world clock: its label and where it is.
type Zone struct {
	Label    string
	Location *time.Location
}

// LoadZones loads the world clocks, leaving out, and logging, any zone that
// will not load, so a wrong time is never shown under a right label. The
// configuration's zones are checked as the file loads, and the zone database
// is embedded with it, so this is only a guard.
func LoadZones(clocks []config.WorldClock, logger *slog.Logger) []Zone {
	zones := make([]Zone, 0, len(clocks))

	for _, clock := range clocks {
		location, loadError := time.LoadLocation(clock.Zone)
		if loadError != nil {
			logger.Warn("world clock not loaded", slog.String("zone", clock.Label))

			continue
		}

		zones = append(zones, Zone{Label: clock.Label, Location: location})
	}

	return zones
}

// Scene is everything the dashboard's layout depends on: the deck's grid,
// the view showing, the page it was opened from, the regular pages and the
// app screens by their index in the pages, how far the screens row has been
// paged along, and the world clocks.
type Scene struct {
	Grid         layout.Grid
	View         View
	FromPage     int
	Regular      []int
	ScreenPages  []int
	ScreenOffset int
	Zones        []Zone
}

// Look is what drawing the dashboard needs beyond its layout: the renderer,
// every page's name by its index, the marker held, drawn as on its page, an
// app screen's application icon, nil for none, and the navigation keys'
// background, which the paging tiles in the same corners take.
type Look struct {
	Renderer             *render.Renderer
	PageNames            []string
	Marker               func() image.Image
	ScreenIcon           func(pageIndex int) image.Image
	NavigationBackground color.RGBA
}

// Tiles is the dashboard, by key index: its pages, or the world clocks.
func (scene Scene) Tiles() []Tile {
	if scene.View == WorldView {
		return scene.worldTiles()
	}

	tiles := make([]Tile, scene.Grid.KeyCount())
	columns := scene.Grid.Columns()

	for row, holds := range rows() {
		if row >= scene.Grid.Rows() {
			break
		}

		for column := range columns {
			position := layout.Position{Row: row, Column: column}
			if keyIndex, onGrid := scene.Grid.KeyIndex(position); onGrid {
				tiles[keyIndex] = scene.tileAt(holds, position)
			}
		}
	}

	return tiles
}

// row is what a row of the dashboard holds.
type row int

const (
	rowEmpty row = iota
	// rowClock: the way back in its first key, the local time across the
	// next six, and the world icon in its last.
	rowClock
	// rowPages: the regular pages, from the first.
	rowPages
	// rowScreens: the app screens, a row's worth from where the paging tiles
	// have moved it.
	rowScreens
	// rowPaging: the paging tiles in their corners, when there are more
	// screens than a row holds.
	rowPaging
)

// rows is the dashboard's rows, top to bottom; a deck with fewer rows shows
// the first of them.
func rows() []row {
	return []row{rowClock, rowPages, rowScreens, rowPaging}
}

// ScreenRows is how many rows' worth of app screens there are to page
// through.
func (scene Scene) ScreenRows() int {
	columns := scene.Grid.Columns()

	return (len(scene.ScreenPages) + columns - 1) / columns
}

// Image draws one key of the dashboard as it is at a moment.
func (scene Scene) Image(look Look, shown Tile, now time.Time) image.Image {
	renderer := look.Renderer

	switch shown.Role {
	case RoleBack:
		return look.Marker()
	case RolePage:
		number := slices.Index(scene.Regular, shown.PageIndex) + 1

		return renderer.Render(scene.markedFrom(shown.PageIndex, render.Appearance{
			Label: look.PageNames[shown.PageIndex], Icon: brick.PageMarkerIcon(number),
		}))
	case RoleScreen:
		icon := look.ScreenIcon(shown.PageIndex)

		return renderer.Render(scene.markedFrom(shown.PageIndex, render.Appearance{
			Label: look.PageNames[shown.PageIndex], IconImage: icon, IconMissing: icon == nil,
			BackgroundFromIcon: true,
		}))
	case RolePrevious, RoleNext:
		direction := render.DirectionForward
		if shown.Role == RolePrevious {
			direction = render.DirectionBack
		}

		return renderer.RenderNavigation(direction, layout.PageIndication{
			Number: scene.ScreenOffset/scene.Grid.Columns() + 1, Count: scene.ScreenRows(),
			CurrentName: "", PreviousName: "", NextName: "",
		}, true, look.NavigationBackground)
	case RoleDigit, RoleZoneLabel, RoleWorld, RoleDate:
		return scene.ClockImage(renderer, shown, now)
	case RoleEmpty, RoleReserved:
	}

	return renderer.RenderBlank()
}

// Figure is what a clock key shows at a moment: one figure of HHMMSS, local
// or in a world clock's zone, or a part of the local date.
func (scene Scene) Figure(shown Tile, now time.Time) string {
	if shown.Role == RoleDate {
		return now.Format(dateParts()[shown.Figure])
	}

	if shown.Zone >= 0 && shown.Zone < len(scene.Zones) {
		now = now.In(scene.Zones[shown.Zone].Location)
	}

	return now.Format(timeFigures)[shown.Figure : shown.Figure+1]
}

// ClockImage draws a clock key at a moment: a figure of a time, a pair's
// first large and its second a little smaller with a small colon after the
// hours' and the minutes', so HH MM SS read as pairs; a zone's label; the
// world icon; or a part of the date.
func (scene Scene) ClockImage(renderer *render.Renderer, shown Tile, now time.Time) image.Image {
	switch shown.Role { //nolint:exhaustive // only clock keys are drawn here; a figure is the rest
	case RoleZoneLabel:
		return renderer.RenderClockLabel(scene.Zones[shown.Zone].Label)
	case RoleWorld:
		return renderer.RenderClockIcon(worldIcon)
	case RoleDate:
		return renderer.RenderClockDate(scene.Figure(shown, now), dateWords()[shown.Figure])
	}

	figure := scene.Figure(shown, now)
	if shown.Figure%figuresPerPair == 0 {
		return renderer.RenderClockDigit(figure)
	}

	return renderer.RenderClockSecondDigit(figure, shown.Figure < ClockFigures-1)
}

// tileAt is what a key holds in a row.
func (scene Scene) tileAt(holds row, position layout.Position) Tile {
	switch holds {
	case rowClock:
		return scene.clockRowTile(position)
	case rowPages:
		return scene.pagesRowTile(position)
	case rowScreens:
		return scene.screensRowTile(position)
	case rowPaging:
		return scene.pagingRowTile(position)
	case rowEmpty:
	}

	return tile(RoleEmpty)
}

// clockRowTile: the way back in the first key, the local time across the
// next six, and the world icon in the last.
func (scene Scene) clockRowTile(position layout.Position) Tile {
	switch {
	case position.Column == 0:
		return Tile{Role: RoleBack, PageIndex: scene.FromPage, Zone: LocalZone, Figure: 0}
	case position.Column <= ClockFigures:
		return Tile{Role: RoleDigit, PageIndex: 0, Zone: LocalZone, Figure: position.Column - 1}
	case position.Column == scene.Grid.Columns()-1:
		return tile(RoleWorld)
	default:
		return tile(RoleReserved)
	}
}

// worldTiles is the world clocks: a row each from the top, its label and then
// its HHMMSS, the rows past them blank, with the world icon in the top right
// and the local date, YY, MM and DD, down the right column beneath it, however
// many clocks there are.
func (scene Scene) worldTiles() []Tile {
	tiles := make([]Tile, scene.Grid.KeyCount())
	last := scene.Grid.Columns() - 1

	for keyIndex := range tiles {
		tiles[keyIndex] = tile(RoleEmpty)
	}

	for row := range scene.Grid.Rows() {
		for column := range scene.Grid.Columns() {
			if keyIndex, onGrid := scene.Grid.KeyIndex(layout.Position{Row: row, Column: column}); onGrid {
				tiles[keyIndex] = worldTileAt(row, column, last, row < len(scene.Zones))
			}
		}
	}

	return tiles
}

// worldTileAt is what a key of the world clocks holds: its row's label or a
// figure of its time, when the row has a clock, the world icon, or a part of
// the date.
func worldTileAt(row int, column int, last int, hasClock bool) Tile {
	if column <= ClockFigures {
		return clockRowTileAt(row, column, hasClock)
	}

	switch {
	case row == 0 && column == last:
		return tile(RoleWorld)
	case column == last && row <= len(dateParts()):
		return Tile{Role: RoleDate, PageIndex: 0, Zone: LocalZone, Figure: row - 1}
	default:
		return tile(RoleEmpty)
	}
}

// clockRowTileAt is a key of a world clock's row: its label, or a figure of
// its time, or blank in a row without a clock.
func clockRowTileAt(row int, column int, hasClock bool) Tile {
	switch {
	case !hasClock:
		return tile(RoleEmpty)
	case column == 0:
		return Tile{Role: RoleZoneLabel, PageIndex: 0, Zone: row, Figure: 0}
	default:
		return Tile{Role: RoleDigit, PageIndex: 0, Zone: row, Figure: column - 1}
	}
}

// pagesRowTile: the regular pages, from the first.
func (scene Scene) pagesRowTile(position layout.Position) Tile {
	if position.Column < len(scene.Regular) {
		return Tile{Role: RolePage, PageIndex: scene.Regular[position.Column], Zone: LocalZone, Figure: 0}
	}

	return tile(RoleEmpty)
}

// screensRowTile: the app screens, from where the row has been paged to.
func (scene Scene) screensRowTile(position layout.Position) Tile {
	if at := scene.ScreenOffset + position.Column; at < len(scene.ScreenPages) {
		return Tile{Role: RoleScreen, PageIndex: scene.ScreenPages[at], Zone: LocalZone, Figure: 0}
	}

	return tile(RoleEmpty)
}

// pagingRowTile: the paging tiles in their corners, when there are more
// screens than a row holds.
func (scene Scene) pagingRowTile(position layout.Position) Tile {
	if len(scene.ScreenPages) > scene.Grid.Columns() {
		switch position {
		case scene.Grid.BackPosition():
			return tile(RolePrevious)
		case scene.Grid.ForwardPosition():
			return tile(RoleNext)
		}
	}

	return tile(RoleEmpty)
}

// markedFrom gives the page the dashboard was opened from its mark.
func (scene Scene) markedFrom(pageIndex int, appearance render.Appearance) render.Appearance {
	if pageIndex == scene.FromPage {
		appearance.Background, _ = render.ParseColour(MarkedBackground)
	}

	return appearance
}

// dateParts is how the local date is written down the world clocks' right
// column: YY, MM and DD.
func dateParts() []string {
	return []string{"06", "01", "02"}
}

// dateWords name the date's parts beneath their figures.
func dateWords() []string {
	return []string{"year", "month", "day"}
}

// Ticks reports a key the clock draws again as time passes: a figure of a
// time, or a part of the date.
func Ticks(shown Tile) bool {
	return shown.Role == RoleDigit || shown.Role == RoleDate
}
