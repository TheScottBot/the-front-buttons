package editor

import (
	"fmt"
	"image"
	"log/slog"
	"net/http"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/deckmirror"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// The deck as it runs, for Render like deck: a key's picture asked for live
// is the daemon's own, from the mirror, where its look depends on live state,
// such as album art or a pipeline's status; and an event stream tells the page
// which tile was pressed or redrawn, by tile alone, so it can flash the tile
// or fetch its picture again. Both are refused with the switch off, so a page
// drawn in words never asks for anything live. The stream is a long request
// behind the same guard as every other: the editor's host, the session and
// its own origin; and only so many may be open at once.

// liveQuery asks for a key's picture as it is live.
const liveQuery = "live"

// keepAliveInterval is how often an idle stream sends a comment, so a closed
// page is noticed and its place freed.
const keepAliveInterval = 25 * time.Second

// liveFrame is the mirror's picture of a stored key, if it has one for that
// kind of key there.
func (pages *pageHandlers) liveFrame(
	configuration config.Configuration, pageIndex int, position layout.Position,
) (image.Image, bool) {
	key, found := keyAt(pageAt(configuration, pageIndex), position)
	if !found {
		return nil, false
	}

	return pages.options.Mirror.Frame(deckmirror.Tile{Page: pageIndex, Position: position}, key.Kind)
}

// events streams the mirror's events to a page drawn like the deck.
func (pages *pageHandlers) events(writer http.ResponseWriter, request *http.Request) {
	if pages.options.Mirror == nil || !rendersLikeDeck(request) {
		http.Error(writer, "the deck's events are only for a page drawn like the deck", http.StatusNotFound)

		return
	}

	events, cancel, subscribed := pages.options.Mirror.Subscribe()
	if !subscribed {
		pages.options.Logger.WarnContext(request.Context(), "editor stream refused",
			slog.String("reason", "too many pages following the deck"))
		http.Error(writer, "too many pages are following the deck; close one", http.StatusServiceUnavailable)

		return
	}

	defer cancel()

	// The stream outlives the server's write timeout, so this response alone
	// has none; a closed page ends it through the request's context.
	controller := http.NewResponseController(writer)
	_ = controller.SetWriteDeadline(time.Time{})

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)

	_ = controller.Flush()

	keepAlive := time.NewTicker(keepAliveInterval)
	defer keepAlive.Stop()

	for {
		var line string

		select {
		case <-request.Context().Done():
			return
		case <-keepAlive.C:
			line = ": still here\n\n"
		case event := <-events:
			line = eventLines(event)
		}

		_, writeError := fmt.Fprint(writer, line)
		if writeError != nil || controller.Flush() != nil {
			return
		}
	}
}

// eventLines is one event as the stream sends it: its kind, and its tile as
// page, row and column, or nothing for a reload.
func eventLines(event deckmirror.Event) string {
	if event.Kind == deckmirror.Reloaded {
		return "event: " + string(event.Kind) + "\ndata: \n\n"
	}

	return fmt.Sprintf("event: %s\ndata: %d.%d.%d\n\n",
		event.Kind, event.Tile.Page, event.Tile.Position.Row, event.Tile.Position.Column)
}
