package editor_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The reorder route and its field naming the page on show.
const (
	reorderPath  = "/page/reorder"
	showingField = "showing"
)

// pageNames lists the configuration's pages by name, in file order.
func pageNames(t *testing.T, harness *editorHarness) []string {
	t.Helper()

	saved := harness.onDisk(t)

	names := make([]string, 0, len(saved.Pages))
	for _, page := range saved.Pages {
		names = append(names, page.Name)
	}

	return names
}

// TestEditor_ReordersAPageIntoAnothersPlace: a page dropped on another's name
// takes its place, the pages between shuffling along, and the editor stays on
// the page it was showing, wherever that went.
func TestEditor_ReordersAPageIntoAnothersPlace(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, reorderPath,
		url.Values{pageField: {playPage}, toField: {workPage}, showingField: {workPage}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/page?page=1&saved=reordered" {
		t.Fatalf("want a redirect to Work in its new place, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}

	if got := strings.Join(pageNames(t, harness), ","); got != "Play,Work,GoLand,PyCharm" {
		t.Fatalf("want Play first, got %s", got)
	}

	if !strings.Contains(harness.get(t, "/page?page=1&saved=reordered").Body.String(), "Page order saved.") {
		t.Error("want the page to say the order was saved")
	}
}

// TestEditor_ReordersAnAppScreenAmongItsApplicationsScreens: an app screen
// moves among the app screens, which is the order the deck pages them in.
func TestEditor_ReordersAnAppScreenAmongItsApplicationsScreens(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.post(t, reorderPath, url.Values{
		pageField: {pycharmScreen}, toField: {golandScreen}, showingField: {pycharmScreen},
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/page?page=1&saved=reordered" {
		t.Fatalf("want a redirect to PyCharm in its new place, got %d %s: %s",
			response.Code, response.Header().Get("Location"), response.Body.String())
	}

	if got := strings.Join(pageNames(t, harness), ","); got != "Work,PyCharm,GoLand,Play" {
		t.Fatalf("want PyCharm before GoLand, got %s", got)
	}
}

// TestEditor_RefusesToReorderAcrossTheTwoSortsOfPage: a regular page cannot
// take an app screen's place, nor the other way, as the two are listed apart.
func TestEditor_RefusesToReorderAcrossTheTwoSortsOfPage(t *testing.T) {
	t.Parallel()

	for _, move := range [][2]string{{playPage, golandScreen}, {golandScreen, workPage}, {"9", workPage}} {
		harness := newCopyKeysHarness(t)
		before := harness.onDisk(t)

		response := harness.post(t, reorderPath,
			url.Values{pageField: {move[0]}, toField: {move[1]}, showingField: {workPage}})
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "Error: ") {
			t.Errorf("%v: want the reorder refused, got %d", move, response.Code)
		}

		requireUnchanged(t, harness, before)
	}
}

// TestEditor_UndoStepsBackThroughAReorder: a removal and then a reorder are
// undone in turn: first the pages go back to their order, then the key comes
// back on its page.
func TestEditor_UndoStepsBackThroughAReorder(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)
	before := harness.onDisk(t)

	harness.post(t, removeKeysPath, url.Values{pageField: {workPage}, tileField: {cutTile}})
	harness.post(t, reorderPath, url.Values{pageField: {playPage}, toField: {workPage}, showingField: {workPage}})

	harness.post(t, undoPath, url.Values{})

	if got := strings.Join(pageNames(t, harness), ","); got != "Work,GoLand,PyCharm,Play" {
		t.Fatalf("want the order undone first, got %s", got)
	}

	harness.post(t, undoPath, url.Values{})
	requireUnchanged(t, harness, before)
}

// TestEditor_PageOffersAMoveToEveryPlaceOfItsSort: the keyboard's way to
// reorder: a list of the places among the pages of its own sort, its own
// chosen, sent as the drag sends it.
func TestEditor_PageOffersAMoveToEveryPlaceOfItsSort(t *testing.T) {
	t.Parallel()

	body := newCopyKeysHarness(t).get(t, firstPagePath).Body.String()

	for _, want := range []string{
		`<form method="post" action="/page/reorder" class="inline">`,
		`<input type="hidden" name="showing" value="0">`,
		`<label for="reorder-to">Move this page to</label>`,
		`<option value="0" selected>Place 1, now Work</option>`,
		`<option value="3">Place 2, now Play</option>`,
		`<button type="submit">Move page</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page: want %s", want)
		}
	}

	if strings.Contains(body, `Place 3`) {
		t.Error("want only the regular pages' places on a regular page")
	}
}

// TestEditor_ScriptReordersPagesByDraggingTheirNames: the script lets a
// page's name be dragged onto another's, posting the same reorder, and the
// drag hint, shown only with the script, says so.
func TestEditor_ScriptReordersPagesByDraggingTheirNames(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"application/x-front-buttons-page", reorderPath, "reorder-moving", "animate"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	if !strings.Contains(harness.get(t, firstPagePath).Body.String(),
		"Drag a page's name along the list to move the page.") {
		t.Error("want the drag hint to say pages can be reordered")
	}
}
