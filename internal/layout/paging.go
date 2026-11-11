package layout

// noPage is the current page when no pages are configured.
const noPage = -1

// Paging is which page the deck shows. Paging wraps around in both directions
// (specification section 1), so the back and forward keys always do something.
// It is a value: each move returns a new Paging, which keeps the daemon's state
// changes explicit and easy to test.
type Paging struct {
	pageCount int
	current   int
}

// NewPaging returns paging over pageCount pages, on the first page.
func NewPaging(pageCount int) Paging {
	if pageCount <= 0 {
		return Paging{pageCount: 0, current: noPage}
	}

	return Paging{pageCount: pageCount, current: 0}
}

// Forward moves to the next page, wrapping from the last to the first.
func (paging Paging) Forward() Paging {
	if !paging.HasPage() {
		return paging
	}

	return Paging{pageCount: paging.pageCount, current: (paging.current + 1) % paging.pageCount}
}

// Back moves to the previous page, wrapping from the first to the last.
func (paging Paging) Back() Paging {
	if !paging.HasPage() {
		return paging
	}

	return Paging{pageCount: paging.pageCount, current: (paging.current + paging.pageCount - 1) % paging.pageCount}
}

// Current is the zero based index of the page shown.
func (paging Paging) Current() int { return paging.current }

// HasPage reports whether there is a page to show.
func (paging Paging) HasPage() bool { return paging.current != noPage }

// Resize adjusts to a new page count after a configuration reload, staying on
// the same page when it still exists and moving to the last page when it does
// not, so a reload never jumps the deck back to the start for no reason.
func (paging Paging) Resize(pageCount int) Paging {
	if pageCount <= 0 {
		return NewPaging(0)
	}

	return Paging{pageCount: pageCount, current: min(max(paging.current, 0), pageCount-1)}
}

// PageIndication is what the navigation keys show, so the current page is never
// ambiguous (specification section 2). Number is counted from one, as a person
// counts pages. TE4 decides how the keys draw it.
type PageIndication struct {
	Number       int
	Count        int
	CurrentName  string
	PreviousName string
	NextName     string
}

// Indication describes the current page and the pages either side of it, given
// the page names in order. It is absent when there are no pages or the names do
// not match the page count, rather than guessing at a name.
func (paging Paging) Indication(pageNames []string) (PageIndication, bool) {
	if !paging.HasPage() || len(pageNames) != paging.pageCount {
		return PageIndication{Number: 0, Count: 0, CurrentName: "", PreviousName: "", NextName: ""}, false
	}

	return PageIndication{
		Number:       paging.current + 1,
		Count:        paging.pageCount,
		CurrentName:  pageNames[paging.current],
		PreviousName: pageNames[paging.Back().current],
		NextName:     pageNames[paging.Forward().current],
	}, true
}
