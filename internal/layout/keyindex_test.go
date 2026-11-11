package layout_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

func TestGrid_KeyIndex_NumbersEveryPositionRowByRowFromTheTopLeft(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)
	seenIndices := make(map[int]bool)

	for row := range 4 {
		for column := range 8 {
			position := layout.Position{Row: row, Column: column}

			keyIndex, indexFound := grid.KeyIndex(position)
			if !indexFound || keyIndex != row*8+column {
				t.Fatalf("want %+v at key %d, got %d (%v)", position, row*8+column, keyIndex, indexFound)
			}

			if seenIndices[keyIndex] {
				t.Fatalf("key %d assigned twice", keyIndex)
			}

			seenIndices[keyIndex] = true

			roundTripped, positionFound := grid.PositionOfKey(keyIndex)
			if !positionFound || roundTripped != position {
				t.Fatalf("key %d maps back to %+v, want %+v", keyIndex, roundTripped, position)
			}
		}
	}

	if len(seenIndices) != grid.KeyCount() || grid.KeyCount() != 32 {
		t.Fatalf("want 32 distinct keys, got %d of %d", len(seenIndices), grid.KeyCount())
	}
}

func TestGrid_KeyIndex_RefusesPositionsOffTheGrid(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	if _, indexFound := grid.KeyIndex(layout.Position{Row: 4, Column: 0}); indexFound {
		t.Error("want no key below the grid")
	}

	for _, keyIndex := range []int{-1, 32} {
		if _, positionFound := grid.PositionOfKey(keyIndex); positionFound {
			t.Errorf("want no position for key %d", keyIndex)
		}
	}
}

func TestGrid_NavigationKeys_AreTheLastRowsFirstAndLastKeys(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	backIndex, _ := grid.KeyIndex(grid.BackPosition())
	forwardIndex, _ := grid.KeyIndex(grid.ForwardPosition())

	if backIndex != 24 || forwardIndex != 31 {
		t.Fatalf("want back at key 24 and forward at key 31, got %d and %d", backIndex, forwardIndex)
	}
}
