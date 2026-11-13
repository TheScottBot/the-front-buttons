package editor_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/editor"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
)

// The navigation every paged gallery shows, named once.
const iconPagesNavigation = `<nav aria-label="Icon pages"`

// Where the larger catalogue's gallery says it is, and its second page.
const (
	onFirstPage   = "Page 1 of 3"
	onSecondPage  = "Page 2 of 3"
	onLastPage    = "Page 3 of 3"
	secondPageURL = "/icons?page=2"
)

// manyIcons is a catalogue larger than one gallery page: dice-000 to dice-299
// and sword-0 to sword-9, 310 in all, so three pages of 120, 120 and 70. The
// shared harness has only four icons, too few to need a second page.
type manyIcons struct{}

func (manyIcons) Names() []string {
	names := make([]string, 0, 310)

	for number := range 10 {
		names = append(names, fmt.Sprintf("sword-%d", number))
	}

	for number := range 300 {
		names = append(names, fmt.Sprintf("dice-%03d", number))
	}

	return names
}

// validFile is a store whose file on disk is always valid and empty; the
// gallery reads it only to show any problems with the file.
type validFile struct{}

func (validFile) OnDisk() (config.Configuration, error) {
	return config.Configuration{SchemaVersion: config.CurrentSchemaVersion, Pages: nil}, nil
}

func (validFile) Current() (config.Configuration, bool) {
	return config.Configuration{SchemaVersion: config.CurrentSchemaVersion, Pages: nil}, true
}

func (validFile) Save(config.Configuration) error { return nil }

func (validFile) LastRejection() error { return nil }

// getManyIcons makes a same origin GET of an editor over the larger catalogue.
// Only the gallery is asked for, so nothing else needs to be real.
func getManyIcons(t *testing.T, path string) string {
	t.Helper()

	handler := editor.NewHandler(editor.Options{
		Address:     testAddress,
		Token:       testToken,
		Store:       validFile{},
		Registry:    nil,
		Presets:     presets.Table{},
		Grid:        layout.Grid{},
		Renderer:    nil,
		Icons:       manyIcons{},
		IconFiles:   nil,
		AppIcons:    nil,
		Apps:        nil,
		Credentials: nil,
		Counters:    nil,
		Platform:    "",
		Logger:      slog.New(slog.DiscardHandler),
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, testOrigin+path, http.NoBody)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s: want 200, got %d", path, recorder.Code)
	}

	return recorder.Body.String()
}

// galleryPageWant is what one page of the gallery should show.
type galleryPageWant struct {
	count    int
	first    string
	last     string
	summary  string
	position string
	previous string
	next     string
}

// checkGalleryPage checks a page lists the right icons, says which they are and
// links to its neighbours, with no link where there is no neighbour.
func checkGalleryPage(t *testing.T, body string, want galleryPageWant) {
	t.Helper()

	icons := listedIcons(body)
	if len(icons) != want.count || icons[0] != want.first || icons[len(icons)-1] != want.last {
		t.Fatalf("want %d icons from %s to %s, got %d: %v", want.count, want.first, want.last, len(icons), icons)
	}

	if !strings.Contains(body, want.summary) {
		t.Errorf("want the summary %q", want.summary)
	}

	if !strings.Contains(body, want.position) {
		t.Errorf("want the position %q", want.position)
	}

	checkPageLink(t, body, "prev", want.previous)
	checkPageLink(t, body, "next", want.next)
}

// checkPageLink checks the previous or next link goes where it should, or is
// absent when want is empty, so no dead link is shown.
func checkPageLink(t *testing.T, body string, relation string, want string) {
	t.Helper()

	present := strings.Contains(body, `rel="`+relation+`"`)

	switch {
	case want == "" && present:
		t.Errorf("want no %s link", relation)
	case want != "" && !strings.Contains(body, `<a href="`+want+`" rel="`+relation+`"`):
		t.Errorf("want the %s link to %s", relation, want)
	}
}

// TestEditor_IconGalleryPages_ShowEveryIconInTurn: the gallery showed only the
// first 120 icons with no way to see the rest. Each page shows the next 120 in
// the same order, says which they are, and links to the pages either side.
func TestEditor_IconGalleryPages_ShowEveryIconInTurn(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]galleryPageWant{
		"/icons": {
			count: 120, first: "dice-000", last: "dice-119", summary: "Showing 1 to 120 of 310.",
			position: onFirstPage, previous: "", next: secondPageURL,
		},
		secondPageURL: {
			count: 120, first: "dice-120", last: "dice-239", summary: "Showing 121 to 240 of 310.",
			position: onSecondPage, previous: "/icons?page=1", next: "/icons?page=3",
		},
		"/icons?page=3": {
			count: 70, first: "dice-240", last: "sword-9", summary: "Showing 241 to 310 of 310.",
			position: onLastPage, previous: secondPageURL, next: "",
		},
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			checkGalleryPage(t, getManyIcons(t, path), want)
		})
	}
}

// TestEditor_IconGalleryPages_ClampAPageOutOfRange: a hand typed or stale page
// number should still show icons, not an empty page or an error.
func TestEditor_IconGalleryPages_ClampAPageOutOfRange(t *testing.T) {
	t.Parallel()

	for page, want := range map[string]string{
		"0":                       onFirstPage,
		"-3":                      onFirstPage,
		"abc":                     onFirstPage,
		"2.5":                     onFirstPage,
		"":                        onFirstPage,
		"4":                       onLastPage,
		"999999999999999999999":   onLastPage,
		"%2B2":                    onSecondPage,
		"00000000000000000000003": onLastPage,
	} {
		body := getManyIcons(t, "/icons?page="+page)

		if !strings.Contains(body, want) || len(listedIcons(body)) == 0 {
			t.Errorf("page %q: want %q with icons shown", page, want)
		}
	}
}

// TestEditor_IconGalleryPages_KeepTheSearch: paging through a search must stay
// within it, with the query encoded in the links.
func TestEditor_IconGalleryPages_KeepTheSearch(t *testing.T) {
	t.Parallel()

	// "Dice 1" matches the 138 dice holding a 1.
	checkGalleryPage(t, getManyIcons(t, "/icons?q=Dice+1"), galleryPageWant{
		count: 120, first: "dice-001", last: "dice-201", summary: "Showing 1 to 120 of 138 matching &#34;Dice 1&#34;.",
		position: "Page 1 of 2", previous: "", next: "/icons?page=2&amp;q=Dice&#43;1",
	})

	checkGalleryPage(t, getManyIcons(t, "/icons?q=Dice+1&page=2"), galleryPageWant{
		count: 18, first: "dice-210", last: "dice-291", summary: "Showing 121 to 138 of 138 matching &#34;Dice 1&#34;.",
		position: "Page 2 of 2", previous: "/icons?page=1&amp;q=Dice&#43;1", next: "",
	})
}

// TestEditor_IconGalleryPages_NoNavigationForOnePage: page links are noise when
// every match already shows.
func TestEditor_IconGalleryPages_NoNavigationForOnePage(t *testing.T) {
	t.Parallel()

	for _, body := range []string{
		newHarness(t, nil).get(t, "/icons").Body.String(),
		getManyIcons(t, "/icons?q=sword"),
		getManyIcons(t, "/icons?q=no-such-one&page=2"),
	} {
		if strings.Contains(body, iconPagesNavigation) {
			t.Errorf("want no page navigation for one page, got %s", body)
		}
	}

	for path, want := range map[string]string{
		"/icons?q=sword":       "Showing 1 to 10 of 10 matching &#34;sword&#34;.",
		"/icons?q=no-such-one": "No icon name holds &#34;no-such-one&#34;.",
	} {
		if body := getManyIcons(t, path); !strings.Contains(body, want) {
			t.Errorf("%s: want the summary %q", path, want)
		}
	}
}

// TestEditor_IconGalleryPages_ANewSearchStartsOnPageOne: the search box, live or
// submitted without JavaScript, sends only the query, and the live search swaps
// only the results, so the page navigation must sit inside them.
func TestEditor_IconGalleryPages_ANewSearchStartsOnPageOne(t *testing.T) {
	t.Parallel()

	body := getManyIcons(t, secondPageURL)

	form := body[strings.Index(body, "<form"):strings.Index(body, "</form>")]
	if strings.Contains(form, `name="page"`) {
		t.Error("want the search form to send no page, so a new search starts on page 1")
	}

	results := strings.Index(body, `<div id="icon-results">`)
	navigation := strings.Index(body, iconPagesNavigation)
	resultsEnd := results + strings.Index(body[results:], "</div>")

	if results < 0 || navigation < results || navigation > resultsEnd {
		t.Error("want the page navigation inside #icon-results, so the live search updates it")
	}
}
