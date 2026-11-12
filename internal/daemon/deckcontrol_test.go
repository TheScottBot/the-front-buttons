package daemon_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/deckcontrol"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The deck control keys' places and levels in these tests: on the first row a
// button at 0, brightness at 2 and off at 3; a live key first on the second.
const (
	liveControlKey  = 8
	brightnessKey   = 2
	offKey          = 3
	dimPercent      = 40
	offPercent      = 25
	usualPercent    = 70
	brightnessByte  = 2
	blankDifference = 1.0
)

func noAppearance() brick.Appearance {
	return brick.Appearance{Label: "", Icon: "", Background: "", IconFile: ""}
}

func controlPage() config.Configuration {
	return config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion,
		Pages: []config.Page{{Name: "Controls", Keys: []config.Key{
			button("first", 0, false),
			liveKey("controls", 1, 0, ""),
			{
				Position:   layout.Position{Row: 0, Column: brightnessKey},
				Kind:       deckcontrol.BrightnessKindName,
				Parameters: &deckcontrol.BrightnessParameters{Appearance: noAppearance(), Level: dimPercent},
				Preset:     nil,
			},
			{
				Position:   layout.Position{Row: 0, Column: offKey},
				Kind:       deckcontrol.OffKindName,
				Parameters: &deckcontrol.OffParameters{Appearance: noAppearance(), Level: offPercent},
				Preset:     nil,
			},
		}}},
	}
}

// lastBrightness is the percentage the deck was last set to.
func lastBrightness(t *testing.T, device *testsupport.DeckDeviceTestDouble) int {
	t.Helper()

	command := xlModel(t).Protocol.BrightnessCommand

	for _, report := range slices.Backward(device.Features()) {
		if report[1] == command {
			return int(report[brightnessByte])
		}
	}

	t.Fatal("want the brightness set")

	return 0
}

// isBlank reports whether a key shows the blank key.
func isBlank(t *testing.T, running *harness, device *testsupport.DeckDeviceTestDouble, keyIndex int) bool {
	t.Helper()

	return distance(drawnImage(t, device, keyIndex), running.renderer.RenderBlank()) < blankDifference
}

// assertShowing checks each key is showing, or off.
func assertShowing(
	t *testing.T,
	running *harness,
	device *testsupport.DeckDeviceTestDouble,
	showing bool,
	keyIndices ...int,
) {
	t.Helper()

	for _, keyIndex := range keyIndices {
		if isBlank(t, running, device, keyIndex) == showing {
			t.Errorf("key %d: want showing %v", keyIndex, showing)
		}
	}
}

func connectControls(t *testing.T) (*harness, *testsupport.DeckDeviceTestDouble) {
	t.Helper()

	running := startHarness(t)
	running.configurations <- controlPage()

	device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
	running.watcher.Attach("deck-a", device)
	time.Sleep(settle)
	synctest.Wait()

	return running, device
}

// TestDaemon_BrightnessKeyTogglesBetweenUsualAndItsLevel: the author asked for
// a key that toggles the deck between its usual brightness and a set one.
func TestDaemon_BrightnessKeyTogglesBetweenUsualAndItsLevel(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := connectControls(t)

		tap(device, brightnessKey)

		if got := lastBrightness(t, device); got != dimPercent {
			t.Fatalf("want the deck dimmed to %d, got %d", dimPercent, got)
		}

		tap(device, brightnessKey)

		if got := lastBrightness(t, device); got != usualPercent {
			t.Fatalf("want the deck back at its usual %d, got %d", usualPercent, got)
		}

		if isBlank(t, running, device, brightnessKey) {
			t.Error("want the brightness key to show its percentage, not a blank key")
		}

		running.stop(t)
	})
}

// TestDaemon_OffKeyBlanksTheOthers_AndAnyPressBringsThemBack: the author
// asked for a key that turns every other key off and the deck down low. While
// off, a press only wakes the deck, since what it would do cannot be seen, and
// waking restores the brightness in force before, dimmed or not.
func TestDaemon_OffKeyBlanksTheOthers_AndAnyPressBringsThemBack(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := connectControls(t)

		tap(device, brightnessKey)
		tap(device, offKey)

		if got := lastBrightness(t, device); got != offPercent {
			t.Fatalf("want the deck turned down to %d while off, got %d", offPercent, got)
		}

		assertShowing(t, running, device, false, 0, liveControlKey, brightnessKey, forwardKey)
		assertShowing(t, running, device, true, offKey)

		// News for a live key while off draws nothing.
		running.live.Set("controls", brick.LiveState{Status: render.StatusInProgress, ErrorReason: "", TargetURL: ""})
		time.Sleep(settle)
		synctest.Wait()
		assertShowing(t, running, device, false, liveControlKey)

		tap(device, 0)

		if got := lastBrightness(t, device); got != dimPercent {
			t.Fatalf("want the dimmed %d back on waking, got %d", dimPercent, got)
		}

		if presses := running.recorder.Presses(); len(presses) != 0 {
			t.Errorf("want the waking press to do nothing else, got presses %v", presses)
		}

		assertShowing(t, running, device, true, 0, liveControlKey, forwardKey)

		tap(device, offKey)
		tap(device, offKey)

		if got := lastBrightness(t, device); got != dimPercent {
			t.Fatalf("want the off key to wake the deck too, at %d, got %d", dimPercent, got)
		}

		running.stop(t)
	})
}

// TestDaemon_ReconnectingDeckKeepsTheBrightnessChosen: the dock moving the
// deck away and back is normal; the deck returns as it was left.
func TestDaemon_ReconnectingDeckKeepsTheBrightnessChosen(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := connectControls(t)

		tap(device, brightnessKey)

		running.watcher.Detach("deck-a")
		time.Sleep(settle)
		synctest.Wait()

		returning := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-b", returning)
		time.Sleep(settle)
		synctest.Wait()

		if got := lastBrightness(t, returning); got != dimPercent {
			t.Fatalf("want the returning deck dimmed to %d, got %d", dimPercent, got)
		}

		running.stop(t)
	})
}
