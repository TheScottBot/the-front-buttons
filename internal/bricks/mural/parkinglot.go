package mural

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The parking lot key, mural_parking_lot, puts the text on the clipboard on a
// sticky note in one area of a board, named by its title, such as a parking
// lot for questions to come back to. Each note goes in the next free place of
// a grid five notes wide, after the notes already in the area, so notes added
// from the deck do not pile on one another.

// ParkingLotKindName is the parking lot key's name in configuration.
const ParkingLotKindName = "mural_parking_lot"

// The grid the notes are placed in, in pixels inside the area: a margin from
// its top left corner, and the step from one note to the next, a note's 138
// pixels and a gap.
const (
	parkingMargin  = 20
	parkingStep    = 150
	parkingColumns = 5
)

// The reasons a parking lot key shows when the clipboard gives it nothing to
// add, or cannot be read.
const (
	clipboardEmptyCategory = "clipboard_empty"
	clipboardReadCategory  = clipboardFailedCategory
)

// errNothingToPark reports a clipboard holding only spaces.
var errNothingToPark = errors.New("the clipboard holds only spaces")

// ParkingLotParameters is the parking lot key's one declaration of its
// parameters.
type ParkingLotParameters struct {
	brick.Appearance

	Link string `brick:"link,required" bounds:"1,2048" format:"mural_board"                                                                        description:"The board's address, as its browser shows it; the note goes on that board."`
	Area string `brick:"area,required" bounds:"1,200"  description:"The title of the area the note goes in, as the board shows it, ignoring case."`
}

// registerParkingLot adds the parking lot kind to a registry.
func registerParkingLot(registry *brick.Registry) error {
	return brick.Register(registry, ParkingLotKindName, func( //nolint:wrapcheck // Register already names the kind
		parameters ParkingLotParameters, ports brick.Ports,
	) (any, error) {
		if ports.MuralBoard == nil || ports.ClipboardReader == nil {
			return nil, fmt.Errorf("%w: %s needs Mural's API and the clipboard", brick.ErrPortMissing,
				ParkingLotKindName)
		}

		// The link was checked as a board's address when the configuration
		// was read, so it always names one.
		muralID, _ := murals.BoardID(parameters.Link)

		return parkingLot{
			muralID: muralID, area: strings.TrimSpace(parameters.Area), board: ports.MuralBoard,
			clipboard: ports.ClipboardReader,
		}, nil
	})
}

// parkingLot is one parking lot key.
type parkingLot struct {
	muralID   string
	area      string
	board     murals.Board
	clipboard platform.ClipboardReader
}

// DefaultLabel says what a press does.
func (parkingLot) DefaultLabel() string { return "Park it" }

// Press reads the clipboard and adds its text, trimmed, as a sticky note in
// the next free place in the area, showing why on the key when it cannot.
func (key parkingLot) Press(pressContext context.Context) error {
	text, readError := key.clipboard.ReadClipboard(pressContext)
	if readError != nil {
		category := clipboardReadCategory
		if errors.Is(readError, platform.ErrClipboardEmpty) {
			category = clipboardEmptyCategory
		}

		return fmt.Errorf("reading the clipboard: %w", brick.Categorised(category, readError))
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("reading the clipboard: %w", brick.Categorised(clipboardEmptyCategory, errNothingToPark))
	}

	addError := key.add(pressContext, text)
	if addError != nil {
		return fmt.Errorf("adding the sticky note: %w", brick.Categorised(muralFailureCategory(addError), addError))
	}

	return nil
}

// add finds the area, counts the notes in it, and adds the text as a note in
// the next free place of the grid.
func (key parkingLot) add(addContext context.Context, text string) error {
	areaID, findError := key.board.FindArea(addContext, key.muralID, key.area)
	if findError != nil {
		return findError //nolint:wrapcheck // the press shows its category
	}

	already, countError := key.board.CountStickyNotes(addContext, key.muralID, areaID)
	if countError != nil {
		return countError //nolint:wrapcheck // as above
	}

	return key.board.AddStickyNote(addContext, key.muralID, murals.StickyNote{ //nolint:wrapcheck // as above
		Text: text, AreaID: areaID,
		X: parkingMargin + (already%parkingColumns)*parkingStep,
		Y: parkingMargin + (already/parkingColumns)*parkingStep,
	})
}
