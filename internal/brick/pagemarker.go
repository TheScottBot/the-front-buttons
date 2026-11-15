package brick

import "fmt"

// PageMarkerKindName is the page marker's kind name. It is named here, beside
// the interface, so code that handles configuration, such as copying a page,
// can leave markers out without depending on the brick itself.
const PageMarkerKindName = "page_marker"

// PageMarker is a key that marks the page it sits on and does nothing when
// pressed. Without a label or icon of its own it shows its page's number.
type PageMarker interface {
	MarksPage()
}

// IsPageMarker reports whether a kind is the page marker, which copying a
// page leaves behind: the page copied onto keeps its own (the author's
// request, evaluation, deck testing).
func IsPageMarker(kindName string) bool {
	return kindName == PageMarkerKindName
}

// PageNumbered is implemented by a page marker's parameters: whether its
// page's number and number icon stand in for a label and icons of its own,
// so the marker follows its page as pages are reordered (the author's
// request, evaluation, deck testing).
type PageNumbered interface {
	ShowsPageNumber() bool
}

// The page numbers the bundled icons draw: up to 11 at full size, and on to
// 99 smaller.
const (
	largestFullSizeNumber = 11
	largestSmallNumber    = 99
)

// PageMarkerIcon is the bundled icon of a regular page's number, counted from
// one, or none for an app screen, numbered 0, or a page past the icons.
func PageMarkerIcon(pageNumber int) string {
	switch {
	case pageNumber < 1 || pageNumber > largestSmallNumber:
		return ""
	case pageNumber <= largestFullSizeNumber:
		return fmt.Sprintf("number-%d", pageNumber)
	default:
		return fmt.Sprintf("number-%d-small", pageNumber)
	}
}
