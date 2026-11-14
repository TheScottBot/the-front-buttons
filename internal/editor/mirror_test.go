package editor_test

import (
	"bufio"
	"context"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deckmirror"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The Docs key of the starting configuration, as the mirror knows it.
const (
	eventsPath = "/events"
	livePath   = "/preview.png?page=0&row=0&column=0&live=1"
	staticPath = "/preview.png?page=0&row=0&column=0"
	docsKind   = "open_url"
)

//nolint:gochecknoglobals // a fixed tile
var docsTile = deckmirror.Tile{Page: 0, Position: layout.Position{Row: 0, Column: 0}}

// livePicture is a picture no preview would draw, so a test can tell it.
func livePicture() image.Image {
	drawn := image.NewRGBA(image.Rect(0, 0, 96, 96))

	for y := range 96 {
		for x := range 96 {
			drawn.Set(x, y, color.RGBA{R: 200, G: 30, B: 90, A: 255})
		}
	}

	return drawn
}

// TestEditor_PreviewsALiveKeyFromTheMirror: asked for a key as it is live, the
// editor gives the daemon's own picture of it, if the mirror has one for that
// kind of key there; otherwise, and when not asked, its own preview.
func TestEditor_PreviewsALiveKeyFromTheMirror(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	static := previewImage(t, harness, staticPath)

	if !testsupport.SameImage(previewImage(t, harness, livePath), static) {
		t.Error("want the editor's own preview while the mirror has none")
	}

	harness.mirror.Set(docsTile, "spotify_album", livePicture())

	if !testsupport.SameImage(previewImage(t, harness, livePath), static) {
		t.Error("want another kind's picture on the tile ignored")
	}

	harness.mirror.Set(docsTile, docsKind, livePicture())

	if !testsupport.SameImage(previewImage(t, harness, livePath), livePicture()) {
		t.Error("want the mirror's picture when asked for the key live")
	}

	if !testsupport.SameImage(previewImage(t, harness, staticPath), static) {
		t.Error("want the editor's own preview when not asked for the key live")
	}
}

// TestEditor_AsksForKeysLive_WhenRenderingLikeTheDeck: with the switch on,
// each key's picture is asked for live; paging tiles, which the deck draws
// from the page alone, are not.
func TestEditor_AsksForKeysLive_WhenRenderingLikeTheDeck(t *testing.T) {
	t.Parallel()

	body := getRenderingLikeDeck(t, newDeckRenderHarness(t), "/page?page=0")
	requireBody(t, body,
		`<img class="deck-face" src="/preview.png?page=0&amp;row=0&amp;column=0&amp;live=1"`,
		`<img class="deck-face" src="/preview.png?page=0&amp;row=3&amp;column=0" alt=""`,
	)
}

// eventServer serves the editor over a real connection, as a stream needs.
func eventServer(t *testing.T, harness *editorHarness) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.Host = testAddress
		harness.handler.ServeHTTP(writer, request)
	}))
	t.Cleanup(server.Close)

	return server
}

// openEvents asks for the event stream with the session and the deck render
// switch as given.
func openEvents(requestContext context.Context, t *testing.T, server *httptest.Server, render string) *http.Response {
	t.Helper()

	request, requestError := http.NewRequestWithContext(requestContext, http.MethodGet, server.URL+eventsPath, http.NoBody)
	if requestError != nil {
		t.Fatal(requestError)
	}

	withSession(request)

	if render != "" {
		withCookie(request, deckRenderCookie, render)
	}

	response, responseError := server.Client().Do(request)
	if responseError != nil {
		t.Fatal(responseError)
	}

	return response
}

// TestEditor_StreamsTileEventsFromTheMirror: with the switch on, the event
// stream tells the page each press, redraw and reload by tile alone.
func TestEditor_StreamsTileEventsFromTheMirror(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	server := eventServer(t, harness)

	streamContext, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	response := openEvents(streamContext, t, server, "on")
	defer func() { _ = response.Body.Close() }()

	contentType := response.Header.Get("Content-Type")
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("want an event stream, got %d %s", response.StatusCode, contentType)
	}

	harness.mirror.Press(docsTile)
	harness.mirror.Release(docsTile)
	harness.mirror.Set(deckmirror.Tile{Page: 1, Position: layout.Position{Row: 2, Column: 3}}, docsKind, livePicture())
	harness.mirror.Replace(nil)

	want := []string{
		"event: press", "data: 0.0.0", "event: release", "data: 0.0.0", "event: frame", "data: 1.2.3", "event: reload",
	}
	reader := bufio.NewScanner(response.Body)

	var heard []string

	for len(heard) < len(want) && reader.Scan() {
		if line := reader.Text(); line != "" && !strings.HasPrefix(line, ":") && line != "data: " {
			heard = append(heard, line)
		}
	}

	if strings.Join(heard, "|") != strings.Join(want, "|") {
		t.Fatalf("want %q, got %q", want, heard)
	}
}

// TestEditor_RefusesTheEventStream_WhenOffOrFull: the stream is only for a
// page drawn like the deck, and only so many may listen at once.
func TestEditor_RefusesTheEventStream_WhenOffOrFull(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	server := eventServer(t, harness)

	for _, render := range []string{"", "off"} {
		response := openEvents(t.Context(), t, server, render)
		_ = response.Body.Close()

		if response.StatusCode != http.StatusNotFound {
			t.Errorf("render %q: want the stream refused, got %d", render, response.StatusCode)
		}
	}

	for range harnessMirrorSubscribers {
		_, cancel, _ := harness.mirror.Subscribe()
		defer cancel()
	}

	response := openEvents(t.Context(), t, server, "on")
	_ = response.Body.Close()

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("want the stream refused while full, got %d", response.StatusCode)
	}
}

// TestEditor_KeepsTheEventStreamPastTheServersTimeouts: the editor's server
// bounds each request's reading and writing, as the binary sets it; the stream
// outlives both and still carries events.
func TestEditor_KeepsTheEventStreamPastTheServersTimeouts(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.Host = testAddress
		harness.handler.ServeHTTP(writer, request)
	}))
	server.Config.ReadTimeout = 200 * time.Millisecond
	server.Config.WriteTimeout = 200 * time.Millisecond
	server.Start()
	t.Cleanup(server.Close)

	streamContext, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	response := openEvents(streamContext, t, server, "on")
	defer func() { _ = response.Body.Close() }()

	time.Sleep(time.Second)
	harness.mirror.Press(docsTile)

	reader := bufio.NewScanner(response.Body)
	for reader.Scan() {
		if reader.Text() == "event: press" {
			return
		}
	}

	t.Fatalf("want the press heard after the server's timeouts, the stream ended: %v", reader.Err())
}
