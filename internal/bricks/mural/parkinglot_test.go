package mural_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/mural"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// errUnreadable is a clipboard that cannot be read.
var errUnreadable = errors.New("the clipboard cannot be read")

// clipboardFake is a clipboard holding some text, or failing.
type clipboardFake struct {
	text    string
	failure error
}

func (fake clipboardFake) ReadClipboard(_ context.Context) (string, error) {
	return fake.text, fake.failure
}

// parkingLotKey builds a parking lot key for the board's area over the fakes.
//
//nolint:ireturn // keys are built as the interfaces the daemon uses them through
func parkingLotKey(t *testing.T, board *boardFake, clipboard platform.ClipboardReader, area string) brick.Pressable {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.ParkingLotKindName)

	constructed, constructError := kind.Construct(
		&mural.ParkingLotParameters{Appearance: noAppearance(), Link: boardLink, Area: area},
		brick.Ports{MuralBoard: board, ClipboardReader: clipboard})
	if constructError != nil {
		t.Fatal(constructError)
	}

	key, _ := constructed.(brick.Pressable)
	if key == nil {
		t.Fatal("want the parking lot key pressable")
	}

	return key
}

// TestParkingLot_PutsTheClipboardOnASticky: a press adds the clipboard's
// text, trimmed, as a sticky note in the area its title names, in the next
// free place of a grid five notes wide, after the notes already there.
func TestParkingLot_PutsTheClipboardOnASticky(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		already int
		want    string
	}{
		{0, `"Ask legal first" in 0-2 at 20,20`},
		{1, `"Ask legal first" in 0-2 at 170,20`},
		{7, `"Ask legal first" in 0-2 at 320,170`},
	} {
		board := newBoardFake(nil, false)
		board.area = parkingAreaID
		board.setStickies(test.already)

		pressError := parkingLotKey(t, board, clipboardFake{text: "  Ask legal first \n", failure: nil}, "Parking lot").
			Press(t.Context())
		if pressError != nil {
			t.Fatal(pressError)
		}

		calls := board.recorded()
		if want := "add " + boardID + " " + test.want; len(calls) != 3 || calls[0] != "find "+boardID+` "Parking lot"` ||
			calls[1] != "count "+boardID+` "0-2"` || calls[2] != want {
			t.Errorf("%d already: want the area found, counted and %s, got %v", test.already, want, calls)
		}
	}
}

// TestParkingLot_SaysWhyItAddedNothing: an empty clipboard, or one holding
// only spaces, shows clipboard_empty; a clipboard that cannot be read shows
// clipboard_failed; a missing area shows no_area; and Mural's refusals show
// why. None adds a note.
func TestParkingLot_SaysWhyItAddedNothing(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		clipboard clipboardFake
		area      string
		failure   error
		want      string
	}{
		{"empty", clipboardFake{text: "", failure: platform.ErrClipboardEmpty}, parkingAreaID, nil, "clipboard_empty"},
		{"spaces", clipboardFake{text: " \n\t", failure: nil}, parkingAreaID, nil, "clipboard_empty"},
		{"unreadable", clipboardFake{text: "", failure: errUnreadable}, parkingAreaID, nil, "clipboard_failed"},
		{"no area", clipboardFake{text: "idea", failure: nil}, "", nil, "no_area"},
		{
			"refused",
			clipboardFake{text: "idea", failure: nil},
			parkingAreaID,
			fmt.Errorf("wrapped: %w", murals.ErrRefused), refusedReason,
		},
	} {
		board := newBoardFake(test.failure, false)
		board.area = test.area

		failure := parkingLotKey(t, board, test.clipboard, "Parking lot").Press(t.Context())
		if category := brick.ReasonCategoryOf(failure); category != test.want {
			t.Errorf("%s: want %q, got %q", test.name, test.want, category)
		}

		for _, call := range board.recorded() {
			if len(call) > 3 && call[:4] == "add " {
				t.Errorf("%s: want no note added, got %v", test.name, board.recorded())
			}
		}
	}
}

// TestParkingLot_SaysWhatItDoesAndNeedsItsPorts: with no label of its own
// the key says Park it, and without the board or the clipboard it refuses to
// construct.
func TestParkingLot_SaysWhatItDoesAndNeedsItsPorts(t *testing.T) {
	t.Parallel()

	key := parkingLotKey(t, newBoardFake(nil, false), clipboardFake{text: "", failure: nil}, "Parking lot")

	if labelled, _ := key.(brick.DefaultLabelled); labelled == nil || labelled.DefaultLabel() != "Park it" {
		t.Error("want Park it as the default label")
	}

	registry := brick.NewRegistry()

	registerError := mural.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(mural.ParkingLotKindName)
	parameters := &mural.ParkingLotParameters{Appearance: noAppearance(), Link: boardLink, Area: "Parking lot"}

	for name, ports := range map[string]brick.Ports{
		"no board":     {ClipboardReader: clipboardFake{text: "", failure: nil}},
		"no clipboard": {MuralBoard: newBoardFake(nil, false)},
	} {
		_, constructError := kind.Construct(parameters, ports)
		if !errors.Is(constructError, brick.ErrPortMissing) {
			t.Errorf("%s: want the port missing, got %v", name, constructError)
		}
	}
}
