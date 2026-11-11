package layout

// KeyIndex returns the device key number of a position. Keys are numbered row by
// row from the top left, starting at zero.
//
// That order is inferred from the reference source and is unverified until
// experiment 1 (Appendix A, deviation DV1). It is computed here and nowhere else,
// so a correction is one change.
func (grid Grid) KeyIndex(position Position) (int, bool) {
	if !grid.Contains(position) {
		return 0, false
	}

	return position.Row*grid.columns + position.Column, true
}

// PositionOfKey returns the position of a device key number.
func (grid Grid) PositionOfKey(keyIndex int) (Position, bool) {
	if keyIndex < 0 || keyIndex >= grid.KeyCount() {
		return Position{Row: 0, Column: 0}, false
	}

	return Position{Row: keyIndex / grid.columns, Column: keyIndex % grid.columns}, true
}

// KeyCount is the number of keys on the deck.
func (grid Grid) KeyCount() int {
	return grid.rows * grid.columns
}
