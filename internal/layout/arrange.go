package layout

import (
	"errors"
	"fmt"
)

// ErrUnplaceableBrick reports a brick position that cannot go on the deck. The
// configuration decoder refuses these first; this is the layout's own guard, so
// a bug elsewhere cannot cover a navigation key.
var ErrUnplaceableBrick = errors.New("brick cannot be placed")

// SlotKind is what occupies a key on the current page.
type SlotKind string

// The kinds of slot.
const (
	// SlotEmpty is a key with nothing on it, drawn blank (specification
	// section 2), not as an error.
	SlotEmpty SlotKind = "empty"
	// SlotBrick is a key showing a brick.
	SlotBrick SlotKind = "brick"
	// SlotBack is the reserved key that pages back.
	SlotBack SlotKind = "back"
	// SlotForward is the reserved key that pages forward.
	SlotForward SlotKind = "forward"
	// SlotApplication is an app screen's reserved key, showing the
	// application's icon, that returns to the regular page.
	SlotApplication SlotKind = "application"
)

// Slot is what one key holds. BrickIndex is the brick's index in the page's list
// of bricks, meaningful only for SlotBrick.
type Slot struct {
	Kind       SlotKind
	BrickIndex int
}

// Arrange lays out one page: the navigation keys in their reserved positions, as
// on every page, each brick at its position, and every other key empty. The
// result is indexed by device key number.
func (grid Grid) Arrange(brickPositions []Position) ([]Slot, error) {
	return grid.arrange(brickPositions, false)
}

// ArrangeApplication lays out an app screen: as Arrange, with the return key
// in its reserved position too.
func (grid Grid) ArrangeApplication(brickPositions []Position) ([]Slot, error) {
	return grid.arrange(brickPositions, true)
}

func (grid Grid) arrange(brickPositions []Position, appScreen bool) ([]Slot, error) {
	slots := make([]Slot, grid.KeyCount())
	for keyIndex := range slots {
		slots[keyIndex] = Slot{Kind: SlotEmpty, BrickIndex: 0}
	}

	backIndex, _ := grid.KeyIndex(grid.BackPosition())
	forwardIndex, _ := grid.KeyIndex(grid.ForwardPosition())
	slots[backIndex] = Slot{Kind: SlotBack, BrickIndex: 0}
	slots[forwardIndex] = Slot{Kind: SlotForward, BrickIndex: 0}

	if appScreen {
		returnIndex, _ := grid.KeyIndex(grid.ReturnPosition())
		slots[returnIndex] = Slot{Kind: SlotApplication, BrickIndex: 0}
	}

	for brickIndex, position := range brickPositions {
		keyIndex, onGrid := grid.KeyIndex(position)

		switch {
		case !onGrid:
			return nil, fmt.Errorf("%w: row %d column %d is off the grid", ErrUnplaceableBrick, position.Row, position.Column)
		case slots[keyIndex].Kind != SlotEmpty:
			return nil, fmt.Errorf("%w: row %d column %d already holds %s",
				ErrUnplaceableBrick, position.Row, position.Column, slots[keyIndex].Kind)
		}

		slots[keyIndex] = Slot{Kind: SlotBrick, BrickIndex: brickIndex}
	}

	return slots, nil
}

// SlotAt returns the slot for a pressed key, and false for a key number the
// arrangement does not have.
func SlotAt(slots []Slot, keyIndex int) (Slot, bool) {
	if keyIndex < 0 || keyIndex >= len(slots) {
		return Slot{Kind: SlotEmpty, BrickIndex: 0}, false
	}

	return slots[keyIndex], true
}
