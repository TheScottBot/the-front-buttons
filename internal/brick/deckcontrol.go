package brick

import "fmt"

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

// FallbackLabel is what a key shows when it has no label of its own: a page
// marker its regular page's number, or on an app screen, numbered 0, the
// screen's name; a brick that names its own its name (a shortcut key its
// shortcut); a brightness key its percentage; an off key the word; and
// anything else nothing. The deck and the editor's previews both use it, so
// they agree.
func FallbackLabel(instance any, pageName string, pageNumber int) string {
	if _, isMarker := instance.(PageMarker); isMarker {
		if pageNumber == 0 {
			return pageName
		}

		return fmt.Sprintf("Page %d", pageNumber)
	}

	if labelled, isLabelled := instance.(DefaultLabelled); isLabelled {
		return labelled.DefaultLabel()
	}

	control, isControl := instance.(DeckControl)
	if !isControl {
		return ""
	}

	request := control.DeckControl()
	if request.Action == DeckControlBrightness {
		return fmt.Sprintf("%d%%", request.Percent)
	}

	return "Off"
}
