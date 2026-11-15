package openapplication_test

import (
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

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
			TriplePressToClose: false,
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

const signalID = "org.whispersystems.signal-desktop"

// unlabelledAppearance is a key's appearance with nothing set.
func unlabelledAppearance() brick.Appearance {
	return brick.Appearance{Label: "", Icon: "", Background: "", IconFile: ""}
}

// constructClosing returns the brick for Signal with Triple press to close
// ticked or not, launching and quitting through doubles over the recorder.
//
//nolint:ireturn // the brick is exercised through the interface the daemon uses
func constructClosing(t *testing.T, recorder *testsupport.LaunchRecorder, ticked bool) brick.Pressable {
	t.Helper()

	parameters := &openapplication.Parameters{
		Appearance:  unlabelledAppearance(),
		Application: signalID, UseAppIcon: false, AppScreen: false, ShowNotifications: false,
		TriplePressToClose: ticked,
	}

	constructed, constructError := kindOf(t).Construct(parameters, brick.Ports{
		ApplicationLauncher: testsupport.ApplicationLauncherTestDouble{Recorder: recorder},
		ApplicationQuitter:  testsupport.ApplicationQuitterTestDouble{Recorder: recorder},
	})
	if constructError != nil {
		t.Fatalf("constructing: %v", constructError)
	}

	pressable, _ := constructed.(brick.Pressable)

	return pressable
}

// pressAt presses the key once after each pause, failing on any press's error.
func pressAt(t *testing.T, key brick.Pressable, pauses ...time.Duration) {
	t.Helper()

	for _, pause := range pauses {
		time.Sleep(pause)

		pressError := key.Press(t.Context())
		if pressError != nil {
			t.Fatalf("pressing: %v", pressError)
		}
	}
}

// TestOpenApplication_TriplePressToCloseQuitsTheApplication: with the box
// ticked, the first press opens the application as ever, and the third
// within moments of the last quits it; the second does nothing more.
func TestOpenApplication_TriplePressToCloseQuitsTheApplication(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		recorder := testsupport.NewLaunchRecorder()
		pressAt(t, constructClosing(t, recorder, true), 0, 200*time.Millisecond, 200*time.Millisecond)

		want := []string{signalID, testsupport.QuittedPrefix + signalID}
		if got := recorder.Launched(); !slices.Equal(got, want) {
			t.Fatalf("want %v, got %v", want, got)
		}
	})
}

// TestOpenApplication_PressesJustAfterClosingDoNothing: a fourth press, or
// more, straight after the quit is ignored rather than opening the
// application again, until the key has been left alone a moment; then a
// press opens it as ever.
func TestOpenApplication_PressesJustAfterClosingDoNothing(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		recorder := testsupport.NewLaunchRecorder()
		key := constructClosing(t, recorder, true)

		pressAt(t, key, 0, 200*time.Millisecond, 200*time.Millisecond, // opens, then quits
			300*time.Millisecond, time.Second, time.Second) // each within the quiet of the last

		quitted := testsupport.QuittedPrefix + signalID

		if got := recorder.Launched(); !slices.Equal(got, []string{signalID, quitted}) {
			t.Fatalf("want only the open and the quit, got %v", got)
		}

		pressAt(t, key, 2*time.Second)

		if got := recorder.Launched(); !slices.Equal(got, []string{signalID, quitted, signalID}) {
			t.Fatalf("want a press after a pause to open it again, got %v", got)
		}
	})
}

// TestOpenApplication_SlowPressesOrAnUntickedBoxOnlyOpen: presses further
// apart than a triple press, or quick ones with the box unticked, each open
// the application and never quit it.
func TestOpenApplication_SlowPressesOrAnUntickedBoxOnlyOpen(t *testing.T) {
	t.Parallel()

	for name, check := range map[string]struct {
		ticked bool
		pause  time.Duration
	}{
		"slow presses": {ticked: true, pause: time.Second},
		"unticked":     {ticked: false, pause: 100 * time.Millisecond},
	} {
		synctest.Test(t, func(t *testing.T) {
			recorder := testsupport.NewLaunchRecorder()
			pressAt(t, constructClosing(t, recorder, check.ticked), 0, check.pause, check.pause, check.pause)

			if got := recorder.Launched(); !slices.Equal(got, []string{signalID, signalID, signalID, signalID}) {
				t.Errorf("%s: want four opens and no quit, got %v", name, got)
			}
		})
	}
}

// TestOpenApplication_ShowsQuitFailed_WhenTheApplicationWillNotQuit: a quit
// that fails shows on the key by its own reason.
func TestOpenApplication_ShowsQuitFailed_WhenTheApplicationWillNotQuit(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		recorder := testsupport.NewLaunchRecorder()
		key := constructClosing(t, recorder, true)
		pressAt(t, key, 0, 100*time.Millisecond)
		recorder.FailNext()

		pressError := key.Press(t.Context())
		if category := brick.ReasonCategoryOf(pressError); category != "quit_failed" {
			t.Fatalf("want quit_failed, got %q (%v)", category, pressError)
		}
	})
}

// TestOpenApplication_RefusesToConstructWithoutAQuitter_WhenClosingIsTicked:
// the box ticked needs the quitting port; unticked, the key does without it.
func TestOpenApplication_RefusesToConstructWithoutAQuitter_WhenClosingIsTicked(t *testing.T) {
	t.Parallel()

	for ticked, wantMissing := range map[bool]bool{true: true, false: false} {
		parameters := &openapplication.Parameters{
			Appearance:  unlabelledAppearance(),
			Application: signalID, UseAppIcon: false, AppScreen: false, ShowNotifications: false,
			TriplePressToClose: ticked,
		}

		_, constructError := kindOf(t).Construct(parameters, brick.Ports{
			ApplicationLauncher: testsupport.ApplicationLauncherTestDouble{Recorder: testsupport.NewLaunchRecorder()},
		})
		if errors.Is(constructError, brick.ErrPortMissing) != wantMissing {
			t.Errorf("ticked %v: want missing %v, got %v", ticked, wantMissing, constructError)
		}
	}
}
