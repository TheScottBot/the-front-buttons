package layout_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// Page names for the indication tests, written once.
const (
	workPage  = "Work"
	playPage  = "Play"
	adminPage = "Admin"
	onlyPage  = "Only"
)

func TestPaging_WrapsForwardFromTheLastPageToTheFirst(t *testing.T) {
	t.Parallel()

	paging := layout.NewPaging(3)

	visited := make([]int, 0, 4)
	visited = append(visited, paging.Current())

	for range 3 {
		paging = paging.Forward()
		visited = append(visited, paging.Current())
	}

	want := []int{0, 1, 2, 0}
	for visitIndex, wantPage := range want {
		if visited[visitIndex] != wantPage {
			t.Fatalf("want forward visits %v, got %v", want, visited)
		}
	}
}

func TestPaging_WrapsBackFromTheFirstPageToTheLast(t *testing.T) {
	t.Parallel()

	paging := layout.NewPaging(3).Back()

	if paging.Current() != 2 {
		t.Fatalf("want back from the first page to land on page 2, got %d", paging.Current())
	}

	if paging.Back().Current() != 1 {
		t.Fatalf("want back from page 2 to land on page 1, got %d", paging.Back().Current())
	}
}

func TestPaging_StaysPut_WhenThereIsOnePage(t *testing.T) {
	t.Parallel()

	paging := layout.NewPaging(1)

	if paging.Forward().Current() != 0 || paging.Back().Current() != 0 {
		t.Fatal("want a single page to stay on page 0 in both directions")
	}
}

func TestPaging_HasNoCurrentPage_WhenThereAreNoPages(t *testing.T) {
	t.Parallel()

	paging := layout.NewPaging(0)

	if paging.HasPage() || paging.Forward().HasPage() || paging.Back().HasPage() {
		t.Fatal("want no current page with no pages configured")
	}
}

func TestPaging_Resize_KeepsThePageWhenItStillExists_AndClampsWhenItDoesNot(t *testing.T) {
	t.Parallel()

	onPageTwo := layout.NewPaging(5).Forward().Forward()

	if resized := onPageTwo.Resize(4); resized.Current() != 2 {
		t.Fatalf("want page 2 kept after shrinking to 4 pages, got %d", resized.Current())
	}

	if resized := onPageTwo.Resize(2); resized.Current() != 1 {
		t.Fatalf("want the last page after shrinking below the current one, got %d", resized.Current())
	}

	if resized := onPageTwo.Resize(0); resized.HasPage() {
		t.Fatal("want no page after shrinking to zero pages")
	}
}

func TestPaging_Indication_NamesTheCurrentPageAndBothNeighbours(t *testing.T) {
	t.Parallel()

	pageNames := []string{workPage, playPage, adminPage}

	indicationCases := []struct {
		forwardSteps int
		want         layout.PageIndication
	}{
		{0, layout.PageIndication{Number: 1, Count: 3, CurrentName: workPage, PreviousName: adminPage, NextName: playPage}},
		{1, layout.PageIndication{Number: 2, Count: 3, CurrentName: playPage, PreviousName: workPage, NextName: adminPage}},
		{2, layout.PageIndication{Number: 3, Count: 3, CurrentName: adminPage, PreviousName: playPage, NextName: workPage}},
	}

	for _, indicationCase := range indicationCases {
		paging := layout.NewPaging(len(pageNames))
		for range indicationCase.forwardSteps {
			paging = paging.Forward()
		}

		got, hasIndication := paging.Indication(pageNames)
		if !hasIndication || got != indicationCase.want {
			t.Errorf("after %d steps want %+v, got %+v", indicationCase.forwardSteps, indicationCase.want, got)
		}
	}
}

func TestPaging_Indication_OnASinglePage_NamesItselfAsBothNeighbours(t *testing.T) {
	t.Parallel()

	got, hasIndication := layout.NewPaging(1).Indication([]string{onlyPage})

	want := layout.PageIndication{Number: 1, Count: 1, CurrentName: onlyPage, PreviousName: onlyPage, NextName: onlyPage}
	if !hasIndication || got != want {
		t.Fatalf("want %+v, got %+v", want, got)
	}
}

func TestPaging_Indication_IsAbsent_WhenNamesDoNotMatchThePageCount(t *testing.T) {
	t.Parallel()

	if _, hasIndication := layout.NewPaging(2).Indication([]string{onlyPage}); hasIndication {
		t.Fatal("want no indication when the names and the page count disagree")
	}

	if _, hasIndication := layout.NewPaging(0).Indication(nil); hasIndication {
		t.Fatal("want no indication with no pages")
	}
}
