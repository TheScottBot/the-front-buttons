package openeditor_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openeditor"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const editorAddress = "http://127.0.0.1:47123/"

func kindOf(t *testing.T) brick.Kind {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := openeditor.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, _ := registry.Kind(openeditor.KindName)

	return kind
}

func TestOpenEditor_OpensTheEditorsAddressWhenPressed(t *testing.T) {
	t.Parallel()

	kind := kindOf(t)
	recorder := testsupport.NewLaunchRecorder()

	constructed, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener:           testsupport.URLOpenerTestDouble{Recorder: recorder},
		ApplicationLauncher: nil,
		PipelineProvider:    nil,
		EditorAddress:       func() string { return editorAddress },
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	pressable, _ := constructed.(brick.Pressable)

	pressError := pressable.Press(t.Context())
	if pressError != nil || len(recorder.Launched()) != 1 || recorder.Launched()[0] != editorAddress {
		t.Fatalf("want the editor opened, got %v (%v)", recorder.Launched(), pressError)
	}
}

func TestOpenEditor_NeedsNoParametersBeyondItsAppearance(t *testing.T) {
	t.Parallel()

	for _, field := range kindOf(t).Schema().Fields() {
		if field.Required {
			t.Errorf("want no required fields, %s is required", field.Name)
		}
	}
}

func TestOpenEditor_RefusesToConstructWithoutTheEditorAddress(t *testing.T) {
	t.Parallel()

	kind := kindOf(t)

	_, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener:           testsupport.URLOpenerTestDouble{Recorder: testsupport.NewLaunchRecorder()},
		ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil,
	})
	if !errors.Is(constructError, brick.ErrPortMissing) {
		t.Fatalf("want ErrPortMissing, got %v", constructError)
	}
}
