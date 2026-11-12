package openapplication_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const teamsAppID = "MSTeams_8wekyb3d8bbwe!MSTeams"

func kindOf(t *testing.T) brick.Kind {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := openapplication.Register(registry)
	if registerError != nil {
		t.Fatalf("registering: %v", registerError)
	}

	kind, _ := registry.Kind(openapplication.KindName)

	return kind
}

// constructWith returns the brick as the interface the daemon presses it through.
//
//nolint:ireturn // the brick is exercised through the interface the daemon uses
func constructWith(t *testing.T, recorder *testsupport.LaunchRecorder, identifier string) brick.Pressable {
	t.Helper()

	kind := kindOf(t)
	parameters := kind.Schema().NewParameters()
	applicationField, _ := kind.Schema().Field("application")

	assignError := kind.Schema().Assign(parameters, applicationField, identifier)
	if assignError != nil {
		t.Fatalf("assigning: %v", assignError)
	}

	constructed, constructError := kind.Construct(parameters, brick.Ports{
		URLOpener:           nil,
		ApplicationLauncher: testsupport.ApplicationLauncherTestDouble{Recorder: recorder},
		PipelineProvider:    nil,
	})
	if constructError != nil {
		t.Fatalf("constructing: %v", constructError)
	}

	pressable, _ := constructed.(brick.Pressable)

	return pressable
}

func TestOpenApplication_LaunchesItsApplicationWhenPressed(t *testing.T) {
	t.Parallel()

	recorder := testsupport.NewLaunchRecorder()

	pressError := constructWith(t, recorder, teamsAppID).Press(t.Context())
	if pressError != nil || len(recorder.Launched()) != 1 || recorder.Launched()[0] != teamsAppID {
		t.Fatalf("want %s launched, got %v (%v)", teamsAppID, recorder.Launched(), pressError)
	}
}

func TestOpenApplication_ShowsOpenFailed_WhenTheApplicationWillNotLaunch(t *testing.T) {
	t.Parallel()

	recorder := testsupport.NewLaunchRecorder()
	recorder.FailNext()

	pressError := constructWith(t, recorder, teamsAppID).Press(t.Context())
	if category := brick.ReasonCategoryOf(pressError); category != "open_failed" {
		t.Fatalf("want open_failed, got %q (%v)", category, pressError)
	}
}

func TestOpenApplication_RefusesToConstructWithoutALauncher(t *testing.T) {
	t.Parallel()

	kind := kindOf(t)

	noPorts := brick.Ports{URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil}

	_, constructError := kind.Construct(kind.Schema().NewParameters(), noPorts)
	if !errors.Is(constructError, brick.ErrPortMissing) {
		t.Fatalf("want ErrPortMissing, got %v", constructError)
	}
}
