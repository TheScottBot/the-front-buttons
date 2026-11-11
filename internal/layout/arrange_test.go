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
