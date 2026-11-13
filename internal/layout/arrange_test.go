package layout_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

func TestArrange_PutsNavigationOnEveryPage_IncludingEmptyOnes(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	pages := [][]layout.Position{
		{},
		{{Row: 0, Column: 0}},
		{{Row: 3, Column: 1}, {Row: 3, Column: 6}, {Row: 2, Column: 7}},
	}

	for pageIndex, brickPositions := range pages {
		slots, arrangeError := grid.Arrange(brickPositions)
		if arrangeError != nil {
			t.Fatalf("page %d: %v", pageIndex, arrangeError)
		}

		if len(slots) != grid.KeyCount() {
			t.Fatalf("page %d: want %d slots, got %d", pageIndex, grid.KeyCount(), len(slots))
		}

		if slots[24].Kind != layout.SlotBack || slots[31].Kind != layout.SlotForward {
			t.Fatalf("page %d: want back at 24 and forward at 31, got %+v and %+v", pageIndex, slots[24], slots[31])
		}
	}
}

func TestArrange_PlacesEachBrickAtItsKey_AndLeavesTheRestEmpty(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	slots, arrangeError := grid.Arrange([]layout.Position{{Row: 0, Column: 3}, {Row: 2, Column: 0}})
	if arrangeError != nil {
		t.Fatalf("arranging: %v", arrangeError)
	}

	brickSlots := map[int]int{3: 0, 16: 1}
	emptyCount := 0

	for keyIndex, slot := range slots {
		wantBrick, isBrickKey := brickSlots[keyIndex]

		switch {
		case isBrickKey:
			if slot.Kind != layout.SlotBrick || slot.BrickIndex != wantBrick {
				t.Errorf("key %d: want brick %d, got %+v", keyIndex, wantBrick, slot)
			}
		case slot.Kind == layout.SlotEmpty:
			emptyCount++
		}
	}

	if emptyCount != 32-2-2 {
		t.Fatalf("want 28 empty keys, got %d", emptyCount)
	}
}

func TestArrange_RefusesBricksOnReservedOffGridOrSharedPositions(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	refusedPages := map[string][]layout.Position{
		"back key":      {{Row: 3, Column: 0}},
		"forward key":   {{Row: 3, Column: 7}},
		"off the grid":  {{Row: 4, Column: 0}},
		"same position": {{Row: 1, Column: 1}, {Row: 1, Column: 1}},
	}

	for caseName, brickPositions := range refusedPages {
		_, arrangeError := grid.Arrange(brickPositions)
		if !errors.Is(arrangeError, layout.ErrUnplaceableBrick) {
			t.Errorf("%s: want ErrUnplaceableBrick, got %v", caseName, arrangeError)
		}
	}
}

func TestArrange_SlotAt_FindsTheSlotForAPressedKey(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	slots, arrangeError := grid.Arrange([]layout.Position{{Row: 1, Column: 2}})
	if arrangeError != nil {
		t.Fatalf("arranging: %v", arrangeError)
	}

	if slot, found := layout.SlotAt(slots, 10); !found || slot.Kind != layout.SlotBrick {
		t.Fatalf("want the brick at key 10, got %+v (%v)", slot, found)
	}

	if _, found := layout.SlotAt(slots, 32); found {
		t.Fatal("want no slot for a key the deck does not have")
	}
}

// TestArrangeApplication_ReservesTheReturnTile: an app screen keeps the top
// left tile for the application's own icon, which returns to the regular page,
// as well as the paging tiles every page has (evaluation, app screens).
func TestArrangeApplication_ReservesTheReturnTile(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	if grid.ReturnPosition() != (layout.Position{Row: 0, Column: 0}) {
		t.Fatalf("want the return tile at the top left, got %+v", grid.ReturnPosition())
	}

	if !grid.IsReservedOn(grid.ReturnPosition(), true) || grid.IsReservedOn(grid.ReturnPosition(), false) {
		t.Error("want the top left reserved on an app screen only")
	}

	if !grid.IsReservedOn(grid.BackPosition(), false) || !grid.IsReservedOn(grid.ForwardPosition(), true) {
		t.Error("want the paging tiles reserved on every page")
	}
}

// TestArrangeApplication_DrawsTheApplicationTopLeft: an app screen's top left
// slot is the application's, and no brick may cover it; a regular page's top
// left stays free for a brick.
func TestArrangeApplication_DrawsTheApplicationTopLeft(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	slots, arrangeError := grid.ArrangeApplication([]layout.Position{{Row: 1, Column: 1}})
	if arrangeError != nil {
		t.Fatal(arrangeError)
	}

	if slots[0].Kind != layout.SlotApplication || slots[9].Kind != layout.SlotBrick {
		t.Errorf("want the application slot top left and the brick at row 2 column 2, got %v and %v",
			slots[0].Kind, slots[9].Kind)
	}

	_, coverError := grid.ArrangeApplication([]layout.Position{{Row: 0, Column: 0}})
	if !errors.Is(coverError, layout.ErrUnplaceableBrick) {
		t.Errorf("want a brick on the return tile refused, got %v", coverError)
	}

	regular, _ := grid.Arrange([]layout.Position{{Row: 0, Column: 0}})
	if regular[0].Kind != layout.SlotBrick {
		t.Error("want the top left free on a regular page")
	}
}
