package editor

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// Finding a key: the words typed into Find a key are matched, case aside,
// against every key on every page and app screen, by what the grid calls it,
// its kind, the name the registry gives that kind, and its page's name; a key
// holding every word is listed, as its page and tile, linking to its page with
// the key marked found. It is a plain search form, so it works without the
// script, which only searches as one types.

// The find's fields, and the most results listed.
const (
	findQuery   = "find"
	foundQuery  = "found"
	mostResults = 50
)

// findResult is one key found: a link to its page, and where and what it is.
type findResult struct {
	Href  string
	Label string
}

// applyFind fills in a page's search and its results, and marks the key the
// page was opened to show.
func (pages *pageHandlers) applyFind(view *pageView, request *http.Request, allPages []config.Page) {
	query := request.URL.Query()
	view.Find = strings.TrimSpace(query.Get(findQuery))

	if view.Find != "" {
		view.FindResults = pages.findKeys(allPages, view.Find)
	}

	found := query.Get(foundQuery)

	for _, row := range view.Rows {
		for column := range row {
			cell := &row[column]
			cell.Found = found != "" && cell.Tile == fmt.Sprintf("%d.%s", view.PageIndex, found)
		}
	}
}

// findKeys lists the keys holding every word of a search, in file order.
func (pages *pageHandlers) findKeys(allPages []config.Page, search string) []findResult {
	words := strings.Fields(strings.ToLower(search))
	results := make([]findResult, 0)

	for pageIndex, page := range allPages {
		for _, key := range sortedKeys(page.Keys) {
			text, _ := describeKey(pages.options.Presets, page, key.Position)
			kindName := shownKind(pages.options.Presets, key)
			haystack := strings.ToLower(strings.Join([]string{
				text, kindName, strings.ReplaceAll(kindName, "_", " "), pages.options.Registry.Title(key.Kind), page.Name,
			}, " "))

			if !containsAll(haystack, words) {
				continue
			}

			tile := fmt.Sprintf("%d.%d", key.Position.Row, key.Position.Column)
			results = append(results, findResult{
				Href: fmt.Sprintf("/page?page=%d&%s=%s&%s=%s", pageIndex, foundQuery, tile, findQuery,
					url.QueryEscape(search)),
				Label: fmt.Sprintf("%s, %s: %s", pages.findPlaceName(allPages, pageIndex),
					tileWords(key.Position), pages.findKeyName(key.Kind, text, kindName)),
			})

			if len(results) == mostResults {
				return results
			}
		}
	}

	return results
}

// findKeyName names a key in a result: by its label and kind, or without a
// label by the name its kind is known by, if it has one, and its kind, or its
// kind alone.
func (pages *pageHandlers) findKeyName(kind string, text string, kindName string) string {
	if text == kindName {
		text = pages.options.Registry.Title(kind)
	}

	if text == "" || text == kindName {
		return kindName
	}

	return text + " (" + kindName + ")"
}

// findPlaceName names a page in a result: "Page 1, Work", numbered among the
// regular pages, or "the GoLand screen".
func (pages *pageHandlers) findPlaceName(allPages []config.Page, pageIndex int) string {
	name := pages.copyPlaceName(allPages, pageIndex)

	return strings.ToUpper(name[:1]) + name[1:]
}

// containsAll reports whether text holds every word.
func containsAll(text string, words []string) bool {
	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}

	return true
}
