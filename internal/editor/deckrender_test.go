package editor_test

import (
	"net/http"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The deck render switch's cookie, and the tiles of the deck render tests.
const (
	deckRenderCookie = "deck_render"
	markerPreview    = "/preview.png?page=0&row=0&column=1"
	backPreview      = "/preview.png?page=0&row=3&column=0"
	forwardPreview   = "/preview.png?page=1&row=3&column=7"
	workName         = "Work"
	playName         = "Play"
)

// deckRenderConfiguration has Work, with a key at row 1, column 1 and a page
// marker with no label beside it, and Play, empty.
const deckRenderConfiguration = startingConfiguration + `      - row: 0
        column: 1
        kind: page_marker
  - name: Play
    keys: []
`

func newDeckRenderHarness(t *testing.T) *editorHarness {
	t.Helper()

	harness := newHarness(t, nil)
	harness.replaceFile(t, deckRenderConfiguration)

	return harness
}

// getRenderingLikeDeck asks for a page with the deck render switch on.
func getRenderingLikeDeck(t *testing.T, harness *editorHarness, path string) string {
	t.Helper()

	return harness.rawGet(t, path, func(request *http.Request) {
		withSession(request)
		withCookie(request, deckRenderCookie, "on")
	}).Body.String()
}

// TestEditor_RendersTheGridLikeTheDeck_WhenSwitchedOn: with the switch on,
// every key and paging tile on the grid is its picture as the deck draws it,
// in select mode too; without it, the grid is words. The switch, hidden until
// the script runs, says which it is.
func TestEditor_RendersTheGridLikeTheDeck_WhenSwitchedOn(t *testing.T) {
	t.Parallel()

	harness := newDeckRenderHarness(t)

	words := harness.get(t, "/page?page=0").Body.String()
	requireBody(t, words,
		`<button type="button" class="render-switch" role="switch" aria-checked="false" hidden>Render like deck</button>`)
	refuseBody(t, words, `class="deck-face"`)

	// Switched off again, the grid is words.
	switchedOff := harness.rawGet(t, "/page?page=0", func(request *http.Request) {
		withSession(request)
		withCookie(request, deckRenderCookie, "off")
	}).Body.String()
	refuseBody(t, switchedOff, `class="deck-face"`, `data-render="deck"`)

	pictures := getRenderingLikeDeck(t, harness, "/page?page=0")
	requireBody(t, pictures,
		`<button type="button" class="render-switch" role="switch" aria-checked="true" hidden>Render like deck</button>`,
		`<img class="deck-face" src="/preview.png?page=0&amp;row=0&amp;column=0&amp;live=1" alt="" width="96" height="96">`,
		`<img class="deck-face" src="/preview.png?page=0&amp;row=3&amp;column=0" alt="" width="96" height="96">`,
		`<html lang="en-GB" data-render="deck">`,
	)
	// An empty tile stays empty, as the deck leaves it dark.
	refuseBody(t, pictures, `src="/preview.png?page=0&amp;row=1&amp;column=1"`)

	selecting := getRenderingLikeDeck(t, harness, "/page?page=0&select=1")
	requireBody(t, selecting, `<img class="deck-face" src="/preview.png?page=0&amp;row=0&amp;column=0&amp;live=1"`,
		// The page marker, which is never copied, is still drawn.
		`<span class="select-empty" aria-label="Row 1, column 2: page_marker"><img class="deck-face"`)
}

// TestEditor_PreviewsAKeyWithTheLabelTheDeckGivesIt: a key with no label of
// its own is previewed with the one the deck shows, here a page marker's page
// number.
func TestEditor_PreviewsAKeyWithTheLabelTheDeckGivesIt(t *testing.T) {
	t.Parallel()

	harness := newDeckRenderHarness(t)

	want := previewRenderer(t).Render(render.Appearance{Label: "Page 1"})
	if !testsupport.SameImage(previewImage(t, harness, markerPreview), want) {
		t.Error("want the page marker previewed with its page number")
	}
}

// TestEditor_PreviewsThePagingTilesAsTheDeckDrawsThem: the paging tiles are
// previewed as the deck draws them, with the page's number and its
// neighbours' names.
func TestEditor_PreviewsThePagingTilesAsTheDeckDrawsThem(t *testing.T) {
	t.Parallel()

	harness := newDeckRenderHarness(t)
	renderer := previewRenderer(t)

	onWork := layout.PageIndication{Number: 1, Count: 2, CurrentName: workName, PreviousName: playName, NextName: playName}
	if !testsupport.SameImage(previewImage(t, harness, backPreview),
		renderer.RenderNavigation(render.DirectionBack, onWork, true)) {
		t.Error("want Work's back tile previewed as the deck draws it")
	}

	onPlay := layout.PageIndication{Number: 2, Count: 2, CurrentName: playName, PreviousName: workName, NextName: workName}
	if !testsupport.SameImage(previewImage(t, harness, forwardPreview),
		renderer.RenderNavigation(render.DirectionForward, onPlay, true)) {
		t.Error("want Play's forward tile previewed as the deck draws it")
	}
}

// placeholderConfiguration has Spotify's album and play keys, with no icon
// of their own, and a pipeline key labelled TFB.
const placeholderConfiguration = `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: spotify_album
      - row: 0
        column: 1
        kind: spotify_play_pause
      - row: 0
        column: 2
        kind: pipeline_status
        parameters:
          label: TFB
          repository: owner/name
          workflow: ci.yml
          branch: main
          credential: github
`

// TestEditor_PreviewsLiveKeysWithAPlaceholder: a key that draws its live
// state, such as album art or a pipeline's status, has none in the editor, so
// it is previewed with a placeholder icon rather than left blank.
func TestEditor_PreviewsLiveKeysWithAPlaceholder(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	harness.replaceFile(t, placeholderConfiguration)
	renderer := previewRenderer(t)

	for _, want := range []struct {
		path       string
		appearance render.Appearance
	}{
		{"/preview.png?page=0&row=0&column=0", render.Appearance{Icon: "vinyl"}},
		{"/preview.png?page=0&row=0&column=1", render.Appearance{Icon: "player-play"}},
		{"/preview.png?page=0&row=0&column=2", render.Appearance{Label: "TFB", Icon: "git-branch"}},
	} {
		if !testsupport.SameImage(previewImage(t, harness, want.path), renderer.Render(want.appearance)) {
			t.Errorf("%s: want the placeholder %+v", want.path, want.appearance)
		}
	}
}
