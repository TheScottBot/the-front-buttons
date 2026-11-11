package layout_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// streamDeckXLTestGrid is the XL's 4 by 8 grid as read from source in the Plan
// stage. TE5's model table becomes the real source of this value.
func streamDeckXLTestGrid(t *testing.T) layout.Grid {
	t.Helper()

	grid, gridError := layout.NewGrid(4, 8)
	if gridError != nil {
		t.Fatalf("building the test grid: %v", gridError)
	}

	return grid
}

func TestGrid_ReservesBottomLeftAndBottomRight(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)

	if grid.BackPosition() != (layout.Position{Row: 3, Column: 0}) {
		t.Errorf("want back at row 3 column 0, got %+v", grid.BackPosition())
	}

	if grid.ForwardPosition() != (layout.Position{Row: 3, Column: 7}) {
		t.Errorf("want forward at row 3 column 7, got %+v", grid.ForwardPosition())
	}
}

func TestGrid_IsReserved_OnlyForTheTwoNavigationPositions(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)
	reservedCount := 0

	for row := range 4 {
		for column := range 8 {
			if grid.IsReserved(layout.Position{Row: row, Column: column}) {
				reservedCount++
			}
		}
	}

	if reservedCount != 2 {
		t.Fatalf("want exactly 2 reserved positions, got %d", reservedCount)
	}
}

func TestGrid_Contains_RejectsPositionsOutsideTheGrid(t *testing.T) {
	t.Parallel()

	grid := streamDeckXLTestGrid(t)
	outsidePositions := []layout.Position{
		{Row: -1, Column: 0},
		{Row: 0, Column: -1},
		{Row: 4, Column: 0},
		{Row: 0, Column: 8},
	}

	for _, outsidePosition := range outsidePositions {
		if grid.Contains(outsidePosition) {
			t.Errorf("want %+v outside the grid", outsidePosition)
		}
	}

	if !grid.Contains(layout.Position{Row: 3, Column: 7}) {
		t.Error("want the bottom right corner inside the grid")
	}
}

func TestNewGrid_RejectsGridsTooSmallForTwoNavigationKeys(t *testing.T) {
	t.Parallel()

	impossibleSizes := []struct {
		rows    int
		columns int
	}{
		{rows: 0, columns: 8},
		{rows: 4, columns: 1},
		{rows: -1, columns: -1},
	}

	for _, impossibleSize := range impossibleSizes {
		_, gridError := layout.NewGrid(impossibleSize.rows, impossibleSize.columns)
		if gridError == nil {
			t.Errorf("want an error for %d rows by %d columns", impossibleSize.rows, impossibleSize.columns)
		}
	}
}
