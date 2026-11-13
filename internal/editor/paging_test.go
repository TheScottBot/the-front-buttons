package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// threePages has One, Two and Three, with no keys.
const threePages = `schema_version: 1
pages:
  - name: One
    keys: []
  - name: Two
    keys: []
  - name: Three
    keys: []
`

// pagingLink is the grid's link on a paging tile, if it is one.
func pagingLink(body string, tile string) string {
	return regexp.MustCompile(`<td data-tile="` + regexp.QuoteMeta(tile) + `" data-state="reserved"><a [^>]*>`).
		FindString(body)
}

// TestEditor_PagingTilesPageLikeTheDeck: the author asked for the grid's
// back and forward tiles to page as the deck's do, wrapping round at either
// end. They stay reserved tiles: nothing can be dropped on them or dragged
// from them.
func TestEditor_PagingTilesPageLikeTheDeck(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, threePages)

	for pageIndex, want := range map[string][2]string{
		"0": {`href="/page?page=2"`, `href="/page?page=1"`},
		"2": {`href="/page?page=1"`, `href="/page?page=0"`},
	} {
		body := harness.get(t, "/page?page="+pageIndex).Body.String()
		back, forward := pagingLink(body, pageIndex+".3.0"), pagingLink(body, pageIndex+".3.7")

		if !strings.Contains(back, want[0]) || !strings.Contains(forward, want[1]) {
			t.Errorf("page %s: want back %s and forward %s, got %q and %q", pageIndex, want[0], want[1], back, forward)
		}

		for _, link := range []string{back, forward} {
			if !strings.Contains(link, `aria-label="`) || strings.Contains(link, "draggable") {
				t.Errorf("page %s: want a named paging link that cannot be dragged, got %q", pageIndex, link)
			}
		}
	}

	if !strings.Contains(pagingLink(harness.get(t, "/page?page=0").Body.String(), "0.3.7"), "Two") {
		t.Error("want the forward tile to name the page it goes to")
	}

	if single := newHarness(t, nil).get(t, "/page?page=0").Body.String(); pagingLink(single, "0.3.0") != "" {
		t.Error("want no paging link with only one page to go to")
	}
}
