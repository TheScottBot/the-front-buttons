package brick

// PageMarkerKindName is the page marker's kind name. It is named here, beside
// the interface, so code that handles configuration, such as copying a page,
// can leave markers out without depending on the brick itself.
const PageMarkerKindName = "page_marker"

// PageMarker is a key that marks the page it sits on and does nothing when
// pressed. Without a label of its own it shows its page's number.
type PageMarker interface {
	MarksPage()
}

// IsPageMarker reports whether a kind is the page marker, which copying a
// page leaves behind: the page copied onto keeps its own (the author's
// request, evaluation, deck testing).
func IsPageMarker(kindName string) bool {
	return kindName == PageMarkerKindName
}
