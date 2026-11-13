package layout

import (
	"errors"
	"fmt"
)

// minimumColumns is the fewest columns a grid may have: the bottom left and
// bottom right keys are both reserved for navigation (specification section 1),
// so with one column they would be the same key and paging would be impossible.
const minimumColumns = 2

// minimumRows is the fewest rows a grid may have.
const minimumRows = 1

// ErrGridTooSmall reports a grid that cannot hold both navigation keys.
var ErrGridTooSmall = errors.New("grid too small for two navigation keys")

// Position is a key's place on the deck, counted from zero at the top left.
type Position struct {
	Row    int
	Column int
}

// Grid is the shape of a deck's keys. It takes its size from the deck model table
// rather than assuming one model, so the reserved positions follow the model.
type Grid struct {
	rows    int
	columns int
}

// NewGrid returns a grid of the given shape, refusing one too small to hold two
// distinct navigation keys.
func NewGrid(rows int, columns int) (Grid, error) {
	if rows < minimumRows || columns < minimumColumns {
		return Grid{rows: 0, columns: 0}, fmt.Errorf("%w: %d rows by %d columns", ErrGridTooSmall, rows, columns)
	}

	return Grid{rows: rows, columns: columns}, nil
}

// Rows is the number of rows.
func (grid Grid) Rows() int { return grid.rows }

// Columns is the number of columns.
func (grid Grid) Columns() int { return grid.columns }

// BackPosition is the bottom left key, reserved on every page for paging back.
func (grid Grid) BackPosition() Position {
	return Position{Row: grid.rows - 1, Column: 0}
}

// ForwardPosition is the bottom right key, reserved on every page for paging
// forward.
func (grid Grid) ForwardPosition() Position {
	return Position{Row: grid.rows - 1, Column: grid.columns - 1}
}

// IsReserved reports whether a position belongs to navigation and so may never
// be assigned a brick by configuration.
func (grid Grid) IsReserved(position Position) bool {
	return position == grid.BackPosition() || position == grid.ForwardPosition()
}

// ReturnPosition is the top left key, reserved on an app screen for the
// application's own icon, which returns to the regular page (evaluation, app
// screens). On a regular page it is free.
func (grid Grid) ReturnPosition() Position {
	return Position{Row: 0, Column: 0}
}

// IsReservedOn reports whether a position is kept from bricks on a page: the
// paging keys on every page, and the return key on an app screen.
func (grid Grid) IsReservedOn(position Position, appScreen bool) bool {
	return grid.IsReserved(position) || (appScreen && position == grid.ReturnPosition())
}

// Contains reports whether a position is on the grid.
func (grid Grid) Contains(position Position) bool {
	return position.Row >= 0 && position.Row < grid.rows && position.Column >= 0 && position.Column < grid.columns
}
