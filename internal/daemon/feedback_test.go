package daemon_test

import (
	"image"
	"image/color"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// signalApplication is the application the triple press key quits.
const signalApplication = "org.whispersystems.signal-desktop"

// closingKeyIndex is where the triple press key sits, beside the first page's
// button.
const closingKeyIndex = 2

// keyDown presses keys and keeps them down while the daemon catches up.
func keyDown(device *testsupport.DeckDeviceTestDouble, keyIndices ...int) {
	device.PressKeys(keyIndices...)
	time.Sleep(testReadTimeout)
	synctest.Wait()
}

// quickTap presses and releases a key as a quick hand does, well within a
// triple press's gap.
func quickTap(device *testsupport.DeckDeviceTestDouble, keyIndex int) {
	keyDown(device, keyIndex)
	keyDown(device)
}

// countWhere counts the pixels in an area of a key's picture that pass a
// test.
func countWhere(picture image.Image, area image.Rectangle, counted func(colour color.RGBA) bool) int {
	count := 0

	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			colour, _ := color.RGBAModel.Convert(picture.At(x, y)).(color.RGBA)
			if counted(colour) {
				count++
			}
		}
	}

	return count
}

// near reports one channel within the deck's compression of another.
func near(channel uint8, want uint8) bool {
	const tolerance = 48

	return int(channel) > int(want)-tolerance && int(channel) < int(want)+tolerance
}

// nearPink reports the hold bar's pink, as compressed for the deck.
func nearPink(colour color.RGBA) bool {
	return near(colour.R, 0xf8) && near(colour.G, 0x94) && near(colour.B, 0xf8)
}

// light reports a pixel nearer white than black.
func light(colour color.RGBA) bool { return colour.R > 0x80 && colour.G > 0x80 && colour.B > 0x80 }

// TestDaemon_DrawsAKeyPressedWhileItIsDown: a key held down is drawn
// pressed, dimmed and pushed in, and let go it is drawn as it was.
func TestDaemon_DrawsAKeyPressedWhileItIsDown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := deckAttached(t, running)
		before := drawnImage(t, device, 0)
		pressed := running.renderer.Pressed(before, 0)

		keyDown(device, 0)
		requireCloser(t, device, 0, pressed, before)

		keyDown(device)
		requireCloser(t, device, 0, before, pressed)

		assertPresses(t, running, "first")
		running.stop(t)
	})
}

// closingConfiguration is the first page's button, with Signal's key beside
// it, Triple press to close ticked.
func closingConfiguration() config.Configuration {
	closing := config.Key{
		Position: layout.Position{Row: 0, Column: closingKeyIndex}, Kind: openapplication.KindName,
		Parameters: &openapplication.Parameters{
			Appearance: noAppearance(), Application: signalApplication,
			UseAppIcon: false, AppScreen: false, ShowNotifications: false, TriplePressToClose: true,
		},
		Preset: nil,
	}

	return config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion, WorldClocks: nil, PlatformBridge: "",
		NavigationBackground: "",
		Pages: []config.Page{
			{Name: firstPageName, Application: "", Keys: []config.Key{button("first", 0, false), closing}},
		},
	}
}

// TestDaemon_ShowsATriplePressesCountOnItsKey: a triple press key shows a dot
// lit top left for each press made, as soon as the press is made, and the
// dots go once the next press is no longer awaited.
func TestDaemon_ShowsATriplePressesCountOnItsKey(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- closingConfiguration()

		device := deckAttached(t, running)
		topLeft := image.Rect(0, 0, 48, 13)
		lit := func() int { return countWhere(drawnImage(t, device, closingKeyIndex), topLeft, light) }

		if before := lit(); before != 0 {
			t.Fatalf("want nothing top left before a press, got %d light pixels", before)
		}

		keyDown(device, closingKeyIndex)

		one := lit()
		if one == 0 {
			t.Fatal("after one press, still down: want its dot lit")
		}

		keyDown(device)

		quickTap(device, closingKeyIndex)

		if two := lit(); two <= one {
			t.Errorf("after two presses: want more lit than after one, got %d then %d", one, two)
		}

		time.Sleep(time.Second)
		synctest.Wait()

		if after := lit(); after != 0 {
			t.Errorf("once the next press is no longer awaited: want the dots gone, got %d light pixels", after)
		}

		running.stop(t)
	})
}

// TestDaemon_DrawsAHoldsBarWhileAKeyIsHeldTowardsIt: a page marker held down
// shows a pink bar filling along its bottom towards the hold, from a moment
// in, so a quick tap never flashes one; let go early, the bar goes.
func TestDaemon_DrawsAHoldsBarWhileAKeyIsHeldTowardsIt(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)
		whole := image.Rect(0, 0, 96, 96)
		bottomLeft, bottomRight := image.Rect(0, 76, 38, 96), image.Rect(67, 76, 96, 96)
		pink := func(area image.Rectangle) int {
			return countWhere(drawnImage(t, device, markerKeyIndex), area, nearPink)
		}

		keyDown(device, markerKeyIndex)

		if early := pink(whole); early != 0 {
			t.Errorf("a moment in: want no bar yet, got %d pink pixels", early)
		}

		time.Sleep(200 * time.Millisecond)
		synctest.Wait()

		if pink(bottomLeft) == 0 || pink(bottomRight) != 0 {
			t.Errorf("most of the way: want the bar pink from the left only, got %d left and %d right",
				pink(bottomLeft), pink(bottomRight))
		}

		keyDown(device)

		if after := pink(whole); after != 0 {
			t.Errorf("let go early: want the bar gone, got %d pink pixels", after)
		}

		running.stop(t)
	})
}
