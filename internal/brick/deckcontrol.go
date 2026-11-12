package brick

// DeckControl is a brick whose press changes the deck itself, its brightness
// or whether its keys show, rather than anything outside it. The daemon owns
// the deck, so it carries the request out in its own loop instead of the
// brick reaching for a port.
type DeckControl interface {
	DeckControl() DeckControlRequest
}

// DeckControlAction is what a deck control asks for.
type DeckControlAction string

const (
	// DeckControlBrightness toggles the deck between its usual brightness and
	// the request's percentage.
	DeckControlBrightness DeckControlAction = "brightness"
	// DeckControlOff toggles every key but this one off, with the deck turned
	// down to the request's percentage.
	DeckControlOff DeckControlAction = "off"
)

// DeckControlRequest is one deck control's request: its action and the
// brightness, in percent, it sets.
type DeckControlRequest struct {
	Action  DeckControlAction
	Percent int
}
