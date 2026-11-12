package editor

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The gallery shows at most this many icons on a page, so a set of thousands
// stays quick to load and to read; the rest are on later pages.
const galleryLimit = 120

// maximumQueryLength bounds a gallery search; no icon name is longer.
const maximumQueryLength = 64

// iconIndex is every bundled icon's name, sorted, and a set for checking a name
// before drawing it.
type iconIndex struct {
	names []string
	known map[string]bool
}

func newIconIndex(catalogue IconCatalogue) iconIndex {
	index := iconIndex{names: nil, known: map[string]bool{}}
	if catalogue == nil {
		return index
	}

	index.names = slices.Clone(catalogue.Names())
	slices.Sort(index.names)

	for _, name := range index.names {
		index.known[name] = true
	}

	return index
}

// matching returns the names holding every word of the query, in order.
func (index iconIndex) matching(query string) []string {
	words := strings.Fields(strings.ToLower(query))

	var matched []string

	for _, name := range index.names {
		if holdsEveryWord(name, words) {
			matched = append(matched, name)
		}
	}

	return matched
}

func holdsEveryWord(name string, words []string) bool {
	for _, word := range words {
		if !strings.Contains(name, word) {
			return false
		}
	}

	return true
}

type iconsView struct {
	baseView

	Query        string
	Icons        []string
	Total        int
	Summary      string
	PageNumber   int
	PageCount    int
	PreviousHref string
	NextHref     string
}

// showIcons is the icon gallery: the author had no way to know which icon names
// exist (evaluation, deck testing). It lists them, drawn as on a key, filtered
// by a search, a page at a time, so a name can be copied into a key's Icon
// field.
func (pages *pageHandlers) showIcons(writer http.ResponseWriter, request *http.Request) {
	_, problems := pages.configuration()

	query := strings.TrimSpace(request.URL.Query().Get("q"))
	if len(query) > maximumQueryLength {
		query = query[:maximumQueryLength]
	}

	matched := pages.icons.matching(query)
	pageCount := max(1, (len(matched)+galleryLimit-1)/galleryLimit)
	pageNumber := min(max(requestedPage(request.URL.Query().Get("page")), 1), pageCount)
	first := (pageNumber - 1) * galleryLimit
	icons := matched[first:min(first+galleryLimit, len(matched))]

	view := iconsView{
		baseView:     pages.base("Icons", request, problems),
		Query:        query,
		Icons:        icons,
		Total:        len(pages.icons.names),
		Summary:      gallerySummary(query, first, len(icons), len(matched)),
		PageNumber:   pageNumber,
		PageCount:    pageCount,
		PreviousHref: "",
		NextHref:     "",
	}

	// A link is given only where there is a page to go to, so none is dead.
	if pageNumber > 1 {
		view.PreviousHref = galleryHref(query, pageNumber-1)
	}

	if pageNumber < pageCount {
		view.NextHref = galleryHref(query, pageNumber+1)
	}

	pages.render(writer, http.StatusOK, "icons.html", view)
}

// requestedPage reads the page asked for. Anything that is not a number counts
// as the first page; a number too large to hold comes back as the largest, so
// it lands on the last page once clamped.
func requestedPage(value string) int {
	number, parseError := strconv.Atoi(value)
	if parseError != nil && !errors.Is(parseError, strconv.ErrRange) {
		return 1
	}

	return number
}

// galleryHref links to one page of the gallery, keeping the search.
func galleryHref(query string, pageNumber int) string {
	values := url.Values{"page": {strconv.Itoa(pageNumber)}}
	if query != "" {
		values.Set("q", query)
	}

	return "/icons?" + values.Encode()
}

func gallerySummary(query string, first int, shown int, matched int) string {
	switch {
	case matched == 0:
		return fmt.Sprintf("No icon name holds %q.", query)
	case query == "":
		return fmt.Sprintf("Showing %d to %d of %d.", first+1, first+shown, matched)
	default:
		return fmt.Sprintf("Showing %d to %d of %d matching %q.", first+1, first+shown, matched, query)
	}
}

// iconPreview draws one bundled icon alone on a key, as the deck would show it.
// Only a known icon's name is drawn; anything else is not found.
func (pages *pageHandlers) iconPreview(writer http.ResponseWriter, request *http.Request) {
	name := request.URL.Query().Get("name")
	if !pages.icons.known[name] {
		http.NotFound(writer, request)

		return
	}

	var encoded bytes.Buffer

	encodeError := png.Encode(&encoded, pages.options.Renderer.Render(render.Appearance{
		Label: "", Icon: name, IconImage: nil, Background: color.RGBA{R: 0, G: 0, B: 0, A: 0},
		Status: render.StatusNone, ErrorReason: "",
	}))
	if encodeError != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	writer.Header().Set("Content-Type", "image/png")
	_, _ = writer.Write(encoded.Bytes())
}
