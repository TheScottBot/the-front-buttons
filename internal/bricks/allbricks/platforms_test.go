package allbricks_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/allbricks"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/spotify"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/windowcycle"
)

// TestAllBricks_SayWhatDoesNotRunOnWindows: the kinds whose ports are only
// written for macOS, Spotify's and window cycling's, are marked as not
// running on Windows with why, window cycling pointing to Alt and Tab
// instead; so is the check box for the Dock's badges. Every kind runs on a
// Mac.
func TestAllBricks_SayWhatDoesNotRunOnWindows(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := allbricks.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	notOnWindows := []string{
		spotify.AlbumKindName, spotify.PlayPauseKindName, spotify.PreviousKindName, spotify.NextKindName,
		windowcycle.NextKindName, windowcycle.PreviousKindName,
	}

	for _, kindName := range registry.KindNames() {
		reason := registry.NotOn(kindName, brick.PlatformWindows)

		if want := slices.Contains(notOnWindows, kindName); want != (reason != "") {
			t.Errorf("%s: want marked not on Windows %v, got %q", kindName, want, reason)
		}

		if onMac := registry.NotOn(kindName, brick.PlatformMac); onMac != "" {
			t.Errorf("%s: want it to run on a Mac, got %q", kindName, onMac)
		}
	}

	for _, kindName := range []string{windowcycle.NextKindName, windowcycle.PreviousKindName} {
		if reason := registry.NotOn(kindName, brick.PlatformWindows); !strings.Contains(reason, "Alt+Tab") {
			t.Errorf("%s: want Alt+Tab offered instead, got %q", kindName, reason)
		}
	}

	kind, _ := registry.Kind(openapplication.KindName)

	badges, found := kind.Schema().Field("show_notifications")
	if !found || !slices.Equal(badges.NotOn, []string{brick.PlatformWindows}) {
		t.Errorf("show_notifications: want it marked not on Windows, got %+v", badges.NotOn)
	}
}
