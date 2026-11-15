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
// clocks side by side, each key its picture, drawn by the same code as the
// deck's. It is shown as opened from the first page with a page marker,
// that marker in its top left; with none, the deck cannot open it, which the
// page says, and it is shown from the first page. Its clocks show the time
// the page was loaded.

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
// its keys, a row of key indices each.
type dashboardPreview struct {
	Name    string
	Heading string
	Rows    [][]int
}

// dashboardView is the dashboard preview page.
type dashboardView struct {
	baseView

	Views []dashboardPreview
	// HasMarker reports a page marker to open the dashboard by.
	HasMarker bool
}

// showDashboard draws the dashboard preview page.
func (pages *pageHandlers) showDashboard(writer http.ResponseWriter, request *http.Request) {
	configuration, problems := pages.configuration()
	grid := pages.options.Grid
	_, _, hasMarker := firstMarker(configuration.Pages)

	keyRows := make([][]int, grid.Rows())
	for row := range keyRows {
		for column := range grid.Columns() {
			keyIndex, _ := grid.KeyIndex(layout.Position{Row: row, Column: column})
			keyRows[row] = append(keyRows[row], keyIndex)
		}
	}

	pages.render(writer, http.StatusOK, "dashboard.html", dashboardView{
		baseView: pages.base("Dashboard", request, problems),
		Views: []dashboardPreview{
			{Name: pagesViewName, Heading: "Pages", Rows: keyRows},
			{Name: worldViewName, Heading: "World clocks", Rows: keyRows},
		},
		HasMarker: hasMarker,
	})
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

	configuration, _ := pages.configuration()
	scene, look := pages.dashboardScene(request.Context(), configuration, view)

	writePNG(writer, scene.Image(look, scene.Tiles()[keyIndex], time.Now()))
}

// dashboardScene is the dashboard as the deck would show it for a
// configuration, opened from its first page marker, and what to draw it with.
func (pages *pageHandlers) dashboardScene(
	sceneContext context.Context, configuration config.Configuration, view dashboard.View,
) (dashboard.Scene, dashboard.Look) {
	regular := regularPages(configuration.Pages)
	fromPage, markerAt, hasMarker := firstMarker(configuration.Pages)

	if !hasMarker && len(regular) > 0 {
		fromPage = regular[0]
	}

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
		markerAt := slices.IndexFunc(allPages[pageIndex].Keys, func(key config.Key) bool {
			return brick.IsPageMarker(key.Kind)
		})
		if markerAt >= 0 {
			return pageIndex, allPages[pageIndex].Keys[markerAt].Position, true
		}
	}

	return 0, layout.Position{Row: 0, Column: 0}, false
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
