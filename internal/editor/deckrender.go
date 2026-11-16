package editor

import (
	"net/http"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// Render like deck: a switch under the theme switch draws every key and
// paging tile on the grid as its picture, as the deck draws it, from the same
// previews the key form shows, in place of words. Its choice is kept in a
// cookie the server reads, as the theme's is, so the grid is drawn either way
// from the start. A key's live state, such as a pipeline's status, is the
// deck's alone and is not drawn.

// deckRenderCookie is where the switch keeps its choice.
const deckRenderCookie = "deck_render"

// rendersLikeDeck reports whether the request's cookie has the switch on.
func rendersLikeDeck(request *http.Request) bool {
	cookie, cookieError := request.Cookie(deckRenderCookie)

	return cookieError == nil && cookie.Value == "on"
}

// pageIndication is what a page's paging tiles show: its number among the
// regular pages and its neighbours' names. An app screen shows none, as which
// regular page it was opened from is the deck's to know.
func pageIndication(allPages []config.Page, pageIndex int) (layout.PageIndication, bool) {
	regular := regularPages(allPages)
	place := slices.Index(regular, pageIndex)

	if place < 0 {
		return screenIndication(allPages, pageIndex)
	}

	names := make([]string, 0, len(regular))
	for _, index := range regular {
		names = append(names, allPages[index].Name)
	}

	paging := layout.NewPaging(len(regular))
	for range place {
		paging = paging.Forward()
	}

	return paging.Indication(names)
}

// fillDeckDefaults gives a key's preview what the deck would draw that its
// own appearance leaves out: the label and icon the deck gives a key without
// them, such as a page marker's page number, and for a key that draws live
// state the editor never sees, such as album art, its placeholder icon in
// place of a blank.
func (pages *pageHandlers) fillDeckDefaults(
	appearance *render.Appearance, allPages []config.Page, pageIndex int, kindName string, parameters any,
) {
	label, icon := pages.deckFallbacks(allPages, pageIndex, kindName, parameters)

	if appearance.Label == "" {
		appearance.Label = label
	}

	if appearance.Icon == "" && appearance.IconImage == nil {
		appearance.Icon = icon
	}

	if appearance.Icon == "" && appearance.IconImage == nil && pages.options.Registry != nil {
		appearance.Icon = pages.options.Registry.PlaceholderIcon(kindName)
	}
}

// deckFallbacks is the label and icon the deck gives a key that has none of
// its own, such as a page marker's page number and its icon.
func (pages *pageHandlers) deckFallbacks(
	allPages []config.Page, pageIndex int, kindName string, parameters any,
) (string, string) {
	if pages.options.Registry == nil {
		return "", ""
	}

	kind, known := pages.options.Registry.Kind(kindName)
	if !known {
		return "", ""
	}

	instance, constructError := kind.Construct(parameters, brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, ApplicationQuitter: nil, PipelineProvider: nil,
		EditorAddress: nil, KeyPresser: nil,
		MediaPlayer: nil, WindowCycler: nil, ShortcutPresser: nil, ClipboardWriter: nil,
		MuralTimer: nil,
	})
	if constructError != nil {
		return pages.options.Registry.Title(kindName), ""
	}

	page := pageAt(config.Configuration{
		SchemaVersion: 0, WorldClocks: nil, PlatformBridge: "", Pages: allPages,
	}, pageIndex)
	pageNumber := slices.Index(regularPages(allPages), pageIndex) + 1

	return brick.FallbackLabel(instance, page.Name, pageNumber), brick.FallbackIcon(instance, pageNumber)
}

// screenIndication is what an app screen's paging tiles show when its
// application has more than one: which screen it is and the names either
// side, as the deck shows them. A lone screen shows none.
func screenIndication(allPages []config.Page, pageIndex int) (layout.PageIndication, bool) {
	page := pageAt(config.Configuration{
		SchemaVersion: 0, WorldClocks: nil, PlatformBridge: "", Pages: allPages,
	}, pageIndex)
	screens := screensOf(allPages, page.Application)
	showing := slices.Index(screens, pageIndex)

	if !page.IsAppScreen() || len(screens) < pagedScreens || showing < 0 {
		return layout.PageIndication{Number: 0, Count: 0, CurrentName: "", PreviousName: "", NextName: ""}, false
	}

	count := len(screens)

	return layout.PageIndication{
		Number: showing + 1, Count: count, CurrentName: page.Name,
		PreviousName: allPages[screens[(showing+count-1)%count]].Name, NextName: allPages[screens[(showing+1)%count]].Name,
	}, true
}
