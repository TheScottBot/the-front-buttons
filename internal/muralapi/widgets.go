package muralapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/murals"
)

// The widgets' address, under a mural's own, as Mural documents it: a page of
// a board's widgets, of the types asked for and inside the parent asked for,
// with a token for the next page.
const widgetsPath = "/widgets"

// The widget types asked for, as Mural names them in its filter.
const (
	areasFilter       = "areas"
	stickyNotesFilter = "sticky notes"
)

// widgetPagesMax limits the pages followed when listing widgets, so a next
// token that never ends cannot keep a key reading.
const widgetPagesMax = 50

// widgetsAnswer is one page of widgets: each one's ID and title, as much as
// the keys need, and the token for the next page, empty on the last.
type widgetsAnswer struct {
	Value []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"value"`
	Next string `json:"next"`
}

// FindArea finds the first area on a board whose title matches, ignoring
// case and the spaces around it, and gives back its ID.
func (client *Client) FindArea(boardContext context.Context, muralID string, title string) (string, error) {
	wanted := strings.TrimSpace(title)
	found := ""

	listError := client.listWidgets(boardContext, muralID, areasFilter, "", func(page widgetsAnswer) bool {
		for _, area := range page.Value {
			if strings.EqualFold(strings.TrimSpace(area.Title), wanted) {
				found = area.ID

				return false
			}
		}

		return true
	})
	if listError != nil {
		return "", listError
	}

	if found == "" {
		return "", fmt.Errorf("%w: %q", murals.ErrNoArea, wanted)
	}

	return found, nil
}

// CountStickyNotes counts the sticky notes on a board, or in one area when
// areaID is not empty.
func (client *Client) CountStickyNotes(boardContext context.Context, muralID string, areaID string) (int, error) {
	count := 0

	listError := client.listWidgets(boardContext, muralID, stickyNotesFilter, areaID, func(page widgetsAnswer) bool {
		count += len(page.Value)

		return true
	})
	if listError != nil {
		return 0, listError
	}

	return count, nil
}

// listWidgets reads a board's widgets of one type, inside a parent if one is
// given, page by page, handing each page to take until it reports false or
// the pages run out.
func (client *Client) listWidgets(
	listContext context.Context, muralID string, widgetType string, parentID string, take func(widgetsAnswer) bool,
) error {
	next := ""

	for range widgetPagesMax {
		query := url.Values{"limit": {strconv.Itoa(resultsPageSize)}, "type": {widgetType}}
		if parentID != "" {
			query.Set("parentId", parentID)
		}

		if next != "" {
			query.Set("next", next)
		}

		// Mural's filter names hold spaces, written %20 as an address's
		// query has them rather than as a form's plus.
		path := muralsPath + muralID + widgetsPath + "?" + strings.ReplaceAll(query.Encode(), "+", "%20")

		answer, callError := client.call(listContext, http.MethodGet, path, muralID, nil)
		if callError != nil {
			return callError
		}

		var page widgetsAnswer

		decodeError := json.Unmarshal(answer, &page)
		if decodeError != nil {
			return fmt.Errorf("%w: the answer is not widgets", murals.ErrUnreachable)
		}

		if !take(page) || page.Next == "" {
			return nil
		}

		next = page.Next
	}

	return nil
}

// stickyNotePath is the address sticky notes are added at, under a mural's
// own, as Mural documents it: a post of an array of notes.
const stickyNotePath = "/widgets/sticky-note"

// stickyNoteShape is the shape every added note takes.
const stickyNoteShape = "rectangle"

// stickyNoteBody is one sticky note as adding posts it: plain text, so
// nothing in it is read as markup, at a place inside its area.
type stickyNoteBody struct {
	Shape    string `json:"shape"`
	Text     string `json:"text"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	ParentID string `json:"parentId"`
}

// AddStickyNote adds one sticky note to an area of a board.
func (client *Client) AddStickyNote(boardContext context.Context, muralID string, note murals.StickyNote) error {
	_, callError := client.call(boardContext, http.MethodPost, muralsPath+muralID+stickyNotePath, muralID,
		[]stickyNoteBody{{Shape: stickyNoteShape, Text: note.Text, X: note.X, Y: note.Y, ParentID: note.AreaID}})

	return callError
}
