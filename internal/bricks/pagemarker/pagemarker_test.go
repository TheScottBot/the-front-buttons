package pagemarker_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/pagemarker"
)

// TestPageMarker_MarksItsPageAndDoesNothingWhenPressed: the author marks each
// page with a key in its corner, and asked for it to be a brick of its own
// that copying a page leaves out (evaluation, deck testing).
func TestPageMarker_MarksItsPageAndDoesNothingWhenPressed(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := pagemarker.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, found := registry.Kind(pagemarker.KindName)
	if !found {
		t.Fatalf("want the %s kind registered", pagemarker.KindName)
	}

	constructed, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
		MediaPlayer: nil,
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	if _, isMarker := constructed.(brick.PageMarker); !isMarker {
		t.Errorf("want a page marker, got %T", constructed)
	}

	if _, isPressable := constructed.(brick.Pressable); isPressable {
		t.Error("want a page marker to do nothing when pressed")
	}

	if !brick.IsPageMarker(pagemarker.KindName) {
		t.Error("want the kind known as a page marker by name, so copying can leave it out")
	}
}

// TestPageMarker_ShowsItsPagesNumberUntilToldOtherwise: the marker's page
// number box starts ticked and, while ticked, stands in for the label and
// icons, so the marker follows its page as pages are reordered.
func TestPageMarker_ShowsItsPagesNumberUntilToldOtherwise(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := pagemarker.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(pagemarker.KindName)

	field, found := kind.Schema().Field("page_number")
	if !found || field.Default != "true" || !slices.Equal(field.Replaces, []string{"label", "icon", "icon_file"}) {
		t.Fatalf("want a page number box, ticked to start, standing in for the label and icons, got %+v", field)
	}

	ticked := pagemarker.Parameters{Label: "Old", Icon: "", Background: "", IconFile: "", PageNumber: true}
	if !ticked.ShowsPageNumber() {
		t.Error("want a ticked marker to show its page's number")
	}

	if (pagemarker.Parameters{Appearance: ticked.Appearance, PageNumber: false}).ShowsPageNumber() {
		t.Error("want an unticked marker to show its own")
	}
}

// TestPageMarker_IconIsItsPagesNumber: a marker's icon is its page's number,
// as large as the bundled icons draw it, and none on an app screen.
func TestPageMarker_IconIsItsPagesNumber(t *testing.T) {
	t.Parallel()

	for pageNumber, want := range map[int]string{0: "", 1: "number-1", 11: "number-11", 12: "number-12-small", 100: ""} {
		if got := brick.PageMarkerIcon(pageNumber); got != want {
			t.Errorf("page %d: want %q, got %q", pageNumber, want, got)
		}
	}
}

// TestPageMarker_SaysItOpensTheDashboard: the marker tells the editor that
// holding it opens the page dashboard.
func TestPageMarker_SaysItOpensTheDashboard(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := pagemarker.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	if description := registry.Description(pagemarker.KindName); !strings.Contains(description, "page dashboard") {
		t.Errorf("want the marker to say it opens the dashboard, got %q", description)
	}
}
