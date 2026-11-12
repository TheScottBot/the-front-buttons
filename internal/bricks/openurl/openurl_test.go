package openurl_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openurl"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const discordAddress = "https://discord.com/channels/123/456"

// constructWith returns the brick as the interface the daemon presses it through.
//
//nolint:ireturn // the brick is exercised through the interface the daemon uses
func constructWith(t *testing.T, recorder *testsupport.LaunchRecorder, address string) brick.Pressable {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := openurl.Register(registry)
	if registerError != nil {
		t.Fatalf("registering: %v", registerError)
	}

	kind, _ := registry.Kind(openurl.KindName)
	parameters := kind.Schema().NewParameters()
	urlField, _ := kind.Schema().Field("url")

	assignError := kind.Schema().Assign(parameters, urlField, address)
	if assignError != nil {
		t.Fatalf("assigning: %v", assignError)
	}

	constructed, constructError := kind.Construct(parameters, brick.Ports{
		URLOpener:           testsupport.URLOpenerTestDouble{Recorder: recorder},
		ApplicationLauncher: nil,
	})
	if constructError != nil {
		t.Fatalf("constructing: %v", constructError)
	}

	pressable, isPressable := constructed.(brick.Pressable)
	if !isPressable {
		t.Fatalf("want open_url pressable, got %T", constructed)
	}

	return pressable
}

func TestOpenURL_OpensItsURLWhenPressed(t *testing.T) {
	t.Parallel()

	recorder := testsupport.NewLaunchRecorder()

	pressError := constructWith(t, recorder, discordAddress).Press(t.Context())
	if pressError != nil || len(recorder.Launched()) != 1 || recorder.Launched()[0] != discordAddress {
		t.Fatalf("want %s opened, got %v (%v)", discordAddress, recorder.Launched(), pressError)
	}
}

func TestOpenURL_ShowsOpenFailedOnTheKey_WhenTheSystemCannotOpenIt(t *testing.T) {
	t.Parallel()

	recorder := testsupport.NewLaunchRecorder()
	recorder.FailNext()

	pressError := constructWith(t, recorder, discordAddress).Press(t.Context())

	if category := brick.ReasonCategoryOf(pressError); category != "open_failed" {
		t.Fatalf("want the open_failed category, got %q (%v)", category, pressError)
	}
}

func TestOpenURL_SchemaRequiresAnHTTPSURL(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := openurl.Register(registry)
	if registerError != nil {
		t.Fatalf("registering: %v", registerError)
	}

	kind, _ := registry.Kind(openurl.KindName)
	urlField, found := kind.Schema().Field("url")

	if !found || !urlField.Required || urlField.Format != brick.FormatHTTPSURL {
		t.Fatalf("want a required https url field, got %+v", urlField)
	}
}

func TestOpenURL_RefusesToConstructWithoutAURLOpener(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := openurl.Register(registry)
	if registerError != nil {
		t.Fatalf("registering: %v", registerError)
	}

	kind, _ := registry.Kind(openurl.KindName)

	noPorts := brick.Ports{URLOpener: nil, ApplicationLauncher: nil}

	_, constructError := kind.Construct(kind.Schema().NewParameters(), noPorts)
	if !errors.Is(constructError, brick.ErrPortMissing) {
		t.Fatalf("want ErrPortMissing, got %v", constructError)
	}
}
