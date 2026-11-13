package pagemarker_test

import (
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
