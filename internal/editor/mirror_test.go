package editor_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
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
func openEvents(
	requestContext context.Context, t *testing.T, client *http.Client, baseURL string, render string,
) *http.Response {
	t.Helper()

	request, requestError := http.NewRequestWithContext(requestContext, http.MethodGet, baseURL+eventsPath, http.NoBody)
	if requestError != nil {
		t.Fatal(requestError)
	}

	withSession(request)

	if render != "" {
		withCookie(request, deckRenderCookie, render)
	}

	response, responseError := client.Do(request)
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

	response := openEvents(t.Context(), t, server.Client(), server.URL, "on")
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
		response := openEvents(t.Context(), t, server.Client(), server.URL, render)
		_ = response.Body.Close()

		if response.StatusCode != http.StatusNotFound {
			t.Errorf("render %q: want the stream refused, got %d", render, response.StatusCode)
		}
	}

	for range harnessMirrorSubscribers {
		_, cancel, _ := harness.mirror.Subscribe()
		defer cancel()
	}

	response := openEvents(t.Context(), t, server.Client(), server.URL, "on")
	_ = response.Body.Close()

	if response.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("want the stream refused while full, got %d", response.StatusCode)
	}
}

// TestEditor_KeepsTheEventStreamPastTheServersTimeouts: the editor's server
// bounds each request's reading and writing, as the binary sets it; the stream
// outlives both and still carries events. The connection is in memory, so the
// server's timeouts run on synctest's fake clock.
func TestEditor_KeepsTheEventStreamPastTheServersTimeouts(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		harness := newHarness(t, nil)
		listener := newPipeListener()

		server := new(http.Server)
		server.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			request.Host = testAddress
			harness.handler.ServeHTTP(writer, request)
		})
		server.ReadTimeout = 200 * time.Millisecond
		server.WriteTimeout = 200 * time.Millisecond

		served := make(chan error, 1)

		go func() { served <- server.Serve(listener) }()

		transport, _ := http.DefaultTransport.(*http.Transport)
		transport = transport.Clone()
		transport.DialContext = listener.dial

		streamContext, cancel := context.WithCancel(t.Context())
		response := openEvents(streamContext, t, &http.Client{
			Transport: transport, CheckRedirect: nil, Jar: nil, Timeout: 0,
		}, "http://"+testAddress, "on")

		// Past both of the server's timeouts on the fake clock.
		time.Sleep(time.Second)
		harness.mirror.Press(docsTile)

		heard := false

		for reader := bufio.NewScanner(response.Body); !heard && reader.Scan(); {
			heard = reader.Text() == "event: press"
		}

		cancel()

		_ = response.Body.Close()

		transport.CloseIdleConnections()

		closeError := server.Close()

		serveError := <-served
		if closeError != nil || !errors.Is(serveError, http.ErrServerClosed) {
			t.Errorf("want the server closed cleanly, got %v and %v", closeError, serveError)
		}

		if !heard {
			t.Fatal("want the press heard after the server's timeouts, the stream ended")
		}
	})
}

// pipeListener serves connections made in memory by its dial, so a server and
// its client share synctest's fake clock.
type pipeListener struct {
	connections chan net.Conn
	closed      chan struct{}
	closeOnce   sync.Once
}

func newPipeListener() *pipeListener {
	return &pipeListener{connections: make(chan net.Conn), closed: make(chan struct{}), closeOnce: sync.Once{}}
}

func (listener *pipeListener) Accept() (net.Conn, error) {
	select {
	case connection := <-listener.connections:
		return connection, nil
	case <-listener.closed:
		return nil, net.ErrClosed
	}
}

func (listener *pipeListener) Close() error {
	listener.closeOnce.Do(func() { close(listener.closed) })

	return nil
}

func (listener *pipeListener) Addr() net.Addr { return pipeAddress{} }

// dial makes a connection in memory and hands the server its other end.
func (listener *pipeListener) dial(dialContext context.Context, _, _ string) (net.Conn, error) {
	client, served := net.Pipe()

	select {
	case listener.connections <- served:
		return client, nil
	case <-listener.closed:
		return nil, net.ErrClosed
	case <-dialContext.Done():
		return nil, fmt.Errorf("dialling in memory: %w", context.Cause(dialContext))
	}
}

// pipeAddress names the in memory connections.
type pipeAddress struct{}

func (pipeAddress) Network() string { return "pipe" }
func (pipeAddress) String() string  { return testAddress }
