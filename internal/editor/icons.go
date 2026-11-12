package editor

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"net/http"
	"slices"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The gallery shows at most this many icons at once, so a page of thousands
// stays quick to load and to read; a search narrows the rest.
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

// matching returns the names holding every word of the query, in order, at
// most limit of them, and how many matched in all.
func (index iconIndex) matching(query string, limit int) ([]string, int) {
	words := strings.Fields(strings.ToLower(query))

	var matched []string

	total := 0

	for _, name := range index.names {
		if !holdsEveryWord(name, words) {
			continue
		}

		total++

		if len(matched) < limit {
			matched = append(matched, name)
		}
	}

	return matched, total
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

	Query   string
	Icons   []string
	Total   int
	Summary string
}

// showIcons is the icon gallery: the author had no way to know which icon names
// exist (evaluation, deck testing). It lists them, drawn as on a key, filtered
// by a search, with the count, so a name can be copied into a key's Icon field.
func (pages *pageHandlers) showIcons(writer http.ResponseWriter, request *http.Request) {
	_, problems := pages.configuration()

	query := strings.TrimSpace(request.URL.Query().Get("q"))
	if len(query) > maximumQueryLength {
		query = query[:maximumQueryLength]
	}

	icons, matched := pages.icons.matching(query, galleryLimit)

	view := iconsView{
		baseView: pages.base("Icons", request, problems),
		Query:    query,
		Icons:    icons,
		Total:    len(pages.icons.names),
		Summary:  gallerySummary(query, len(icons), matched),
	}

	pages.render(writer, http.StatusOK, "icons.html", view)
}

func gallerySummary(query string, shown int, matched int) string {
	switch {
	case matched == 0:
		return fmt.Sprintf("No icon name holds %q.", query)
	case shown < matched:
		return fmt.Sprintf("Showing the first %d of %d; type more of a name to narrow them.", shown, matched)
	case query == "":
		return fmt.Sprintf("Showing all %d.", matched)
	default:
		return fmt.Sprintf("%d match %q.", matched, query)
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
