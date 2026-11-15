package editor

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// All pages: every page's grid drawn small, side by side, regular pages first
// and then app screens, so keys can be moved, copied and swapped between
// pages by dragging, without opening each in turn. Each grid's tiles are the
// pages screen's own, so the drag script works on them as it does there, and
// every drop is the same POST as on the pages screen. A key links to its page
// with the key marked, as a find result does.

// overviewPage is one page of the overview: its place in the file, its name,
// what a screen reader is told of it, and its grid.
type overviewPage struct {
	Index   int
	Name    string
	Caption string
	Rows    [][]cellView
	// Application is an app screen's application, empty for a regular
	// page, so the zoomed out pages screen keeps to a screen's own kind.
	Application string
}

// overviewView is the overview: every page, in the order shown.
type overviewView struct {
	baseView

	Pages []overviewPage
}

// showOverview draws every page's grid.
func (pages *pageHandlers) showOverview(writer http.ResponseWriter, request *http.Request) {
	configuration, problems := pages.configuration()
	view := overviewView{baseView: pages.base("All pages", request, problems), Pages: nil}

	regular := regularPages(configuration.Pages)
	order := slices.Clone(regular)

	for pageIndex, page := range configuration.Pages {
		if page.IsAppScreen() {
			order = append(order, pageIndex)
		}
	}

	for _, pageIndex := range order {
		view.Pages = append(view.Pages, pages.overviewPageOf(configuration.Pages, regular, pageIndex))
	}

	pages.render(writer, http.StatusOK, "overview.html", view)
}

// overviewPageOf is one page's grid for the overview, its keys linking to the
// page with the key marked.
func (pages *pageHandlers) overviewPageOf(allPages []config.Page, regular []int, pageIndex int) overviewPage {
	page := allPages[pageIndex]
	caption := fmt.Sprintf("Page %d of %d: %s", slices.Index(regular, pageIndex)+1, len(regular), page.Name)

	if page.IsAppScreen() {
		caption = pages.screenCaption(allPages, pageIndex)
	}

	rows := pages.gridRows(allPages, pageIndex)
	for _, row := range rows {
		for column := range row {
			if cell := &row[column]; cell.State == tileHoldsKey {
				cell.Href = fmt.Sprintf("/page?page=%d&%s=%s", pageIndex, foundQuery, cell.Select)
			}
		}
	}

	name := page.Name
	if page.IsAppScreen() {
		name = pages.screenName(allPages, pageIndex) + " screen"
	}

	return overviewPage{Index: pageIndex, Name: name, Caption: caption, Rows: rows, Application: page.Application}
}
