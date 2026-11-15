package editor

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/dashboard"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// Dashboard: the page dashboard as the deck draws it, its pages and its world
// clocks, each key its picture, drawn by the same code as the deck's. It is
// shown as opened from the page the deck shows, marked, with that page's
// marker in its top left; before the deck has said, from the first page with
// a page marker, or with none, which the page says, from the first page. The
// page's script draws each clock key again as often as its figure can change,
// asking for it at that second, and the pages again when the deck turns to
// another page.

// The dashboard's views, as the preview names them.
const (
	pagesViewName = "pages"
	worldViewName = "world"
)

// dashboardViewNames are the views by name, in the order shown.
func dashboardViewNames() map[string]dashboard.View {
	return map[string]dashboard.View{pagesViewName: dashboard.PagesView, worldViewName: dashboard.WorldView}
}

// dashboardPreview is one view of the preview: its name, its heading, and
// its keys, a row each.
type dashboardPreview struct {
	Name    string
	Heading string
	Rows    [][]previewKey
}

// previewKey is one key of a view: its index, and how many seconds pass
// before its figure may change, or none for a key that does not tick.
type previewKey struct {
	Index  int
	Period int
}

// dashboardView is the dashboard preview page.
type dashboardView struct {
	baseView

	Views []dashboardPreview
	// HasMarker reports a page marker to open the dashboard by.
	HasMarker bool
	// Clocks is the world clocks form.
	Clocks clocksForm
	// Showing is the page the deck shows, as /dashboard/showing says it.
	Showing string
}

// showDashboard draws the dashboard preview page, its clocks form filled in
// from the file.
func (pages *pageHandlers) showDashboard(writer http.ResponseWriter, request *http.Request) {
	configuration, problems := pages.configuration()
	pages.renderDashboard(writer, request, http.StatusOK, configuration, problems, clocksFormFor(configuration))
}

// renderDashboard draws the dashboard preview page with its clocks form as
// given.
func (pages *pageHandlers) renderDashboard(
	writer http.ResponseWriter, request *http.Request, status int,
	configuration config.Configuration, problems []string, form clocksForm,
) {
	_, _, hasMarker := firstMarker(configuration.Pages)

	pages.render(writer, status, "dashboard.html", dashboardView{
		baseView: pages.base("Dashboard", request, problems),
		Views: []dashboardPreview{
			{Name: pagesViewName, Heading: "Pages", Rows: pages.previewRows(request.Context(), configuration,
				dashboard.PagesView)},
			{Name: worldViewName, Heading: "World clocks", Rows: pages.previewRows(request.Context(), configuration,
				dashboard.WorldView)},
		},
		HasMarker: hasMarker,
		Clocks:    form,
		Showing:   pages.showingText(),
	})
}

// previewRows is a view's keys, a row each, with how often each ticks.
func (pages *pageHandlers) previewRows(
	rowsContext context.Context, configuration config.Configuration, view dashboard.View,
) [][]previewKey {
	grid := pages.options.Grid
	scene, _ := pages.dashboardScene(rowsContext, configuration, view)
	tiles := scene.Tiles()

	keyRows := make([][]previewKey, grid.Rows())
	for row := range keyRows {
		for column := range grid.Columns() {
			keyIndex, _ := grid.KeyIndex(layout.Position{Row: row, Column: column})
			keyRows[row] = append(keyRows[row], previewKey{Index: keyIndex, Period: tickPeriod(tiles[keyIndex])})
		}
	}

	return keyRows
}

// tickPeriod is how many seconds pass before a key's figure may change: each
// second for the seconds' second figure, each ten for their first, and each
// minute for the rest of a time, whatever a zone's offset, and for a part of
// the date; none for a key that does not tick.
func tickPeriod(shown dashboard.Tile) int {
	const (
		eachSecond = 1
		eachTen    = 10
		eachMinute = 60
	)

	switch {
	case !dashboard.Ticks(shown):
		return 0
	case shown.Role == dashboard.RoleDigit && shown.Figure == dashboard.ClockFigures-1:
		return eachSecond
	case shown.Role == dashboard.RoleDigit && shown.Figure == dashboard.ClockFigures-2:
		return eachTen
	default:
		return eachMinute
	}
}

// showingText is the page the deck shows, by its index, or empty before the
// deck has said.
func (pages *pageHandlers) showingText() string {
	page, showing := pages.options.Mirror.ShowingPage()
	if !showing {
		return ""
	}

	return strconv.Itoa(page)
}

// dashboardShowing says the page the deck shows, so the preview's script
// can draw the pages again when the deck turns to another.
func (pages *pageHandlers) dashboardShowing(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = writer.Write([]byte(pages.showingText()))
}

// dashboardKey draws one key of a view of the dashboard.
func (pages *pageHandlers) dashboardKey(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	view, knownView := dashboardViewNames()[query.Get("view")]
	keyIndex, indexError := strconv.Atoi(query.Get("key"))

	if !knownView || indexError != nil || keyIndex < 0 || keyIndex >= pages.options.Grid.KeyCount() {
		http.NotFound(writer, request)

		return
	}

	// A key asked for at a second is drawn as it is then, so the page's
	// script draws each second's figures on the second.
	moment := time.Now()

	at, atError := strconv.ParseInt(query.Get("at"), 10, 64)
	if atError == nil {
		moment = time.Unix(at, 0)
	}

	configuration, _ := pages.configuration()
	scene, look := pages.dashboardScene(request.Context(), configuration, view)

	writePNG(writer, scene.Image(look, scene.Tiles()[keyIndex], moment))
}

// dashboardScene is the dashboard as the deck would show it for a
// configuration, opened from the page the deck shows, and what to draw it
// with.
func (pages *pageHandlers) dashboardScene(
	sceneContext context.Context, configuration config.Configuration, view dashboard.View,
) (dashboard.Scene, dashboard.Look) {
	regular := regularPages(configuration.Pages)
	fromPage, markerAt, hasMarker := pages.openedFrom(configuration.Pages)

	var screens []int

	names := make([]string, 0, len(configuration.Pages))

	for pageIndex, page := range configuration.Pages {
		names = append(names, page.Name)

		if page.IsAppScreen() {
			screens = append(screens, pageIndex)
		}
	}

	scene := dashboard.Scene{
		Grid: pages.options.Grid, View: view, FromPage: fromPage, Regular: regular, ScreenPages: screens,
		ScreenOffset: 0, Zones: dashboard.LoadZones(configuration.WorldClocksOrDefault(), pages.options.Logger),
	}
	look := dashboard.Look{
		Renderer:  pages.options.Renderer,
		PageNames: names,
		Marker: func() image.Image {
			if !hasMarker {
				return pages.options.Renderer.RenderBlank()
			}

			return pages.previewPicture(sceneContext, configuration, fromPage, markerAt, url.Values{})
		},
		ScreenIcon: func(pageIndex int) image.Image {
			return pages.loadAppIcon(sceneContext, configuration.Pages[pageIndex].Application)
		},
	}

	return scene, look
}

// openedFrom is the page the preview is opened from, the place of its marker
// and whether it has one: the page the deck shows, once the deck has said and
// while the file still has it; otherwise the first page with a marker, or with
// none, the first page.
func (pages *pageHandlers) openedFrom(allPages []config.Page) (int, layout.Position, bool) {
	if page, showing := pages.options.Mirror.ShowingPage(); showing && page >= 0 && page < len(allPages) {
		markerAt, hasMarker := markerOn(allPages[page])

		return page, markerAt, hasMarker
	}

	fromPage, markerAt, hasMarker := firstMarker(allPages)
	if regular := regularPages(allPages); !hasMarker && len(regular) > 0 {
		fromPage = regular[0]
	}

	return fromPage, markerAt, hasMarker
}

// firstMarker is the first page marker, on the regular pages in order and
// then the app screens: its page's index and its place.
func firstMarker(allPages []config.Page) (int, layout.Position, bool) {
	order := regularPages(allPages)

	for pageIndex, page := range allPages {
		if page.IsAppScreen() {
			order = append(order, pageIndex)
		}
	}

	for _, pageIndex := range order {
		if markerAt, hasMarker := markerOn(allPages[pageIndex]); hasMarker {
			return pageIndex, markerAt, true
		}
	}

	return 0, layout.Position{Row: 0, Column: 0}, false
}

// markerOn is the place of a page's first page marker, if it has one.
func markerOn(page config.Page) (layout.Position, bool) {
	markerAt := slices.IndexFunc(page.Keys, func(key config.Key) bool { return brick.IsPageMarker(key.Kind) })
	if markerAt < 0 {
		return layout.Position{Row: 0, Column: 0}, false
	}

	return page.Keys[markerAt].Position, true
}

// writePNG answers with a picture as a PNG.
func writePNG(writer http.ResponseWriter, picture image.Image) {
	var encoded bytes.Buffer

	encodeError := png.Encode(&encoded, picture)
	if encodeError != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	writer.Header().Set("Content-Type", "image/png")
	_, _ = writer.Write(encoded.Bytes())
}
