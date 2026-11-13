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

// TestOpenApplication_OffersUseAppIcon_AsACheckbox: the author asked for a
// check box to use the application's own icon; a boolean field is drawn as one.
func TestOpenApplication_OffersUseAppIcon_AsACheckbox(t *testing.T) {
	t.Parallel()

	field, found := kindOf(t).Schema().Field("use_app_icon")
	if !found || field.Type != brick.FieldTypeBoolean || field.Required || field.Description == "" {
		t.Fatalf("want an optional, described boolean use_app_icon, got %+v (found %v)", field, found)
	}
}

func TestOpenApplication_AsksForTheApplicationsIcon_OnlyWhenTicked(t *testing.T) {
	t.Parallel()

	kind := kindOf(t)
	schema := kind.Schema()

	for _, ticked := range []bool{false, true} {
		parameters := schema.NewParameters()
		applicationField, _ := schema.Field("application")
		useField, _ := schema.Field("use_app_icon")

		assignError := errors.Join(schema.Assign(parameters, applicationField, teamsAppID),
			schema.Assign(parameters, useField, ticked))
		if assignError != nil {
			t.Fatalf("assigning: %v", assignError)
		}

		source, isSource := parameters.(brick.ApplicationIconSource)
		if !isSource {
			t.Fatalf("want the parameters to offer the application's icon, got %T", parameters)
		}

		identifier, wanted := source.ApplicationIcon()
		if wanted != ticked || identifier != teamsAppID {
			t.Errorf("ticked %v: got %q, wanted %v", ticked, identifier, wanted)
		}
	}
}

// TestParameters_OfferTheAppScreenWhenTicked: the App screen check box makes
// the key open the application's screen as well as the application (the
// author's request, evaluation, app screens); unticked, the key only launches,
// as before.
func TestParameters_OfferTheAppScreenWhenTicked(t *testing.T) {
	t.Parallel()

	unlabelled := brick.Appearance{Label: "", Icon: "", Background: "", IconFile: ""}

	for ticked, wantWanted := range map[bool]bool{true: true, false: false} {
		parameters := &openapplication.Parameters{
			Appearance:  unlabelled,
			Application: "com.jetbrains.goland", UseAppIcon: false, AppScreen: ticked, ShowNotifications: false,
		}

		source, isSource := any(parameters).(brick.ApplicationScreenSource)
		if !isSource {
			t.Fatal("want the parameters to offer an app screen")
		}

		application, wanted := source.ApplicationScreen()
		if application != "com.jetbrains.goland" || wanted != wantWanted {
			t.Errorf("ticked %v: want %v for GoLand, got %v for %q", ticked, wantWanted, wanted, application)
		}
	}
}
