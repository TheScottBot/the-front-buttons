package editor_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// The overview's route.
const overviewPath = "/overview"

// TestEditor_OverviewShowsEveryPageAtOnce: All pages draws every page's grid,
// small, side by side, regular pages first and then app screens, each named
// and linked to its page; a key links to its page with it marked, and can be
// picked up.
func TestEditor_OverviewShowsEveryPageAtOnce(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	response := harness.get(t, overviewPath)
	if response.Code != http.StatusOK {
		t.Fatalf("want the overview, got %d", response.Code)
	}

	body := response.Body.String()
	for _, want := range []string{
		`<div class="overview">`,
		`<h2 id="overview-0"><a href="/page?page=0">Work</a></h2>`,
		`<table class="grid mini" data-overview-page="0" data-application="">`,
		`<td data-tile="0.0.1" data-state="key"><a href="/page?page=0&amp;found=0.1" ` +
			`aria-label="Row 1, column 2: Cut (open_url)" draggable="true">`,
		`<td data-tile="3.0.0" data-state="empty">`,
		`<div id="key-bin" class="key-bin" hidden>`,
		// The drag script sends every change with the editor's token.
		`<input type="hidden" name="token" value="` + testToken + `">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("overview: want %s", want)
		}
	}

	order := regexp.MustCompile(`<h2 id="overview-(\d+)">`).FindAllStringSubmatch(body, -1)

	got := make([]string, 0, len(order))
	for _, match := range order {
		got = append(got, match[1])
	}

	if strings.Join(got, ",") != "0,3,1,2" {
		t.Errorf("want Work and Play, then the GoLand and PyCharm screens, got %v", got)
	}

	if !strings.Contains(harness.get(t, firstPagePath).Body.String(), `<li><a href="/overview">All pages</a></li>`) {
		t.Error("want All pages in the editor's sections")
	}
}

// TestEditor_OverviewDragsKeysBetweenPages: the overview loads the drag
// script for its grids, and after a change refreshes them in place rather
// than going to the page the change landed on.
func TestEditor_OverviewDragsKeysBetweenPages(t *testing.T) {
	t.Parallel()

	harness := newCopyKeysHarness(t)

	script := harness.get(t, "/static/editor.js").Body.String()
	for _, want := range []string{"data-overview-page", "refreshOverview"} {
		if !strings.Contains(script, want) {
			t.Errorf("script: want %s", want)
		}
	}

	if !strings.Contains(harness.get(t, "/static/editor.css").Body.String(), ".grid.mini td {") {
		t.Error("stylesheet: want the overview's grids small")
	}
}
