package daemon_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const (
	liveRunURL = "https://github.com/octo-org/hello-world/actions/runs/8"
	buildLabel = "Build"
)

func liveKey(source string, row int, column int, credential string) config.Key {
	return config.Key{
		Position: layout.Position{Row: row, Column: column},
		Kind:     testsupport.LiveKindName,
		Parameters: &testsupport.LiveParameters{
			Label: buildLabel, Icon: "", Background: "", IconFile: "", Source: source, Credential: credential,
		},
		Preset: nil,
	}
}

// livePages puts a live key on the second page, so a test can update it while
// the first page is showing.
func livePages() config.Configuration {
	return config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion,
		Pages: []config.Page{
			{Name: "First", Keys: []config.Key{button("first", 0, false), liveKey("shared", 0, 1, "")}},
			{Name: "Second", Keys: []config.Key{liveKey("shared", 0, 0, "")}},
		},
	}
}

// drawnImage reassembles the last image written to a key and turns it back the
// right way up, so a test can see what the deck shows.
func drawnImage(t *testing.T, device *testsupport.DeckDeviceTestDouble, keyIndex int) image.Image {
	t.Helper()

	var payload []byte

	for _, report := range device.Writes() {
		if report[0] != 0x02 || report[1] != 0x07 || int(report[2]) != keyIndex {
			continue
		}

		if report[6] == 0 && report[7] == 0 {
			payload = nil
		}

		length := int(report[4]) | int(report[5])<<8
		payload = append(payload, report[8:8+length]...)
	}

	flipped, decodeError := jpeg.Decode(bytes.NewReader(payload))
	if decodeError != nil {
		t.Fatalf("decoding key %d: %v", keyIndex, decodeError)
	}

	size := flipped.Bounds().Dx()
	upright := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := range size {
		for x := range size {
			upright.Set(x, y, flipped.At(size-1-x, size-1-y))
		}
	}

	return upright
}

// distance is the mean squared difference between two images' grey levels.
func distance(first image.Image, second image.Image) float64 {
	total := 0.0
	bounds := first.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			firstGrey, _ := color.GrayModel.Convert(first.At(x, y)).(color.Gray)
			secondGrey, _ := color.GrayModel.Convert(second.At(x, y)).(color.Gray)
			difference := float64(firstGrey.Y) - float64(secondGrey.Y)
			total += difference * difference
		}
	}

	return total / float64(bounds.Dx()*bounds.Dy())
}

func TestDaemon_RepaintsALiveKey_WhenItsSourcePublishes(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- livePages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		drawnBefore := device.DrawnKeys()[1]

		running.live.Set("shared", brick.LiveState{Status: render.StatusInProgress, ErrorReason: "", TargetURL: ""})
		time.Sleep(settle)
		synctest.Wait()

		if drawnAfter := device.DrawnKeys()[1]; drawnAfter != drawnBefore+1 {
			t.Fatalf("want the live key redrawn once, drawn %d then %d", drawnBefore, drawnAfter)
		}

		running.stop(t)
	})
}

func TestDaemon_ShowsAnOffPageUpdate_TheMomentThePageIsShown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- livePages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		running.live.Set("shared", brick.LiveState{Status: render.StatusFailure, ErrorReason: "", TargetURL: liveRunURL})
		time.Sleep(settle)
		synctest.Wait()

		tap(device, forwardKey)

		shown := drawnImage(t, device, 0)
		failure := running.renderer.Render(render.Appearance{Label: buildLabel, Status: render.StatusFailure})
		success := running.renderer.Render(render.Appearance{Label: buildLabel, Status: render.StatusSuccess})

		if distance(shown, failure) >= distance(shown, success) {
			t.Fatal("want the second page's live key to show the failure published while it was off page")
		}

		running.stop(t)
	})
}

func TestDaemon_PressesALiveKeyWithItsLatestState(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- livePages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		running.live.Set("shared", brick.LiveState{Status: render.StatusFailure, ErrorReason: "", TargetURL: liveRunURL})
		time.Sleep(settle)
		synctest.Wait()

		tap(device, 1)

		if pressed := running.live.Pressed(); len(pressed) != 1 || pressed[0] != liveRunURL {
			t.Fatalf("want the live key pressed with its run, got %v", pressed)
		}

		running.stop(t)
	})
}

func TestDaemon_RunsLiveSourcesWithNoDeck_AndSharesThemBetweenKeys(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- livePages()

		time.Sleep(settle)
		synctest.Wait()

		if watching := running.live.Watching("shared"); watching != 1 {
			t.Fatalf("want one watch for two keys on one source, with no deck, got %d", watching)
		}

		running.stop(t)
	})
}

func TestDaemon_LogsAConfiguredCredentialThatIsMissing(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- config.Configuration{
			SchemaVersion: config.CurrentSchemaVersion,
			Pages:         []config.Page{{Name: "Only", Keys: []config.Key{liveKey("shared", 0, 0, "never-stored")}}},
		}

		time.Sleep(settle)
		synctest.Wait()

		logs := running.logs.String()
		if !strings.Contains(logs, `"msg":"credential missing"`) || !strings.Contains(logs, `"credential":"never-stored"`) {
			t.Fatalf("want the missing credential named in the log, got:\n%s", logs)
		}

		running.stop(t)
	})
}

func TestDaemon_CountsPressesByKind(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, 0)
		tap(device, 0)

		snapshot := running.counters.Snapshot()
		if snapshot["key_presses."+testsupport.ButtonKindName] != 2 || snapshot["deck.connects"] != 1 {
			t.Fatalf("want two presses and one connect counted, got %v", snapshot)
		}

		running.stop(t)
	})
}

// TestDaemon_NewerLiveNewsReplacesAnOlderPressError presses a live key before
// its source has published, which fails with no run, then publishes a failed
// run. On the real deck the press error stayed on the key over the newer run
// state, so a failed pipeline read "no run" (evaluation, deck testing). The
// newer news wins: the key shows the failure.
func TestDaemon_NewerLiveNewsReplacesAnOlderPressError(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- livePages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, 1)
		time.Sleep(settle)
		synctest.Wait()

		noRun := running.renderer.Render(render.Appearance{Label: buildLabel, ErrorReason: testsupport.LiveNoRunReason})
		plain := running.renderer.Render(render.Appearance{Label: buildLabel})

		if early := drawnImage(t, device, 1); distance(early, noRun) >= distance(early, plain) {
			t.Fatal("want the early press to show its no run error first")
		}

		running.live.Set("shared", brick.LiveState{Status: render.StatusFailure, ErrorReason: "", TargetURL: liveRunURL})
		time.Sleep(settle)
		synctest.Wait()

		shown := drawnImage(t, device, 1)
		failure := running.renderer.Render(render.Appearance{Label: buildLabel, Status: render.StatusFailure})

		if distance(shown, failure) >= distance(shown, noRun) {
			t.Fatal("want the published failure shown over the older press error")
		}

		running.stop(t)
	})
}
