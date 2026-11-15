package daemon_test

import (
	"image"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The clock's keys on the XL: the time between the corners of the top row,
// the world icon in its top right corner.
const (
	firstDigitKey = 1
	worldKeyIndex = 7
	keysPerRow    = 8
)

// otherFigure is a figure that is not the one given, to tell the two apart.
func otherFigure(figure byte) string {
	return strconv.Itoa((int(figure-'0') + 1) % 10)
}

// requireTime fails unless the six keys from a key show a time's HHMMSS, each
// figure drawn closer to itself than to another, the deck's pictures being
// compressed.
func requireTime(t *testing.T, running *harness, device *testsupport.DeckDeviceTestDouble, first int, at time.Time) {
	t.Helper()

	figures := at.Format("150405")
	for offset := range len(figures) {
		drawn := drawnImage(t, device, first+offset)
		right := figureImage(running, offset, figures[offset:offset+1])
		wrong := figureImage(running, offset, otherFigure(figures[offset]))

		if distance(drawn, right) >= distance(drawn, wrong) {
			t.Errorf("key %d: want %s of %s", first+offset, figures[offset:offset+1], figures)
		}
	}
}

// figureImage is how a figure of HHMMSS is drawn at its place: a pair's first
// large, its second smaller, a colon after the hours' and minutes'.
func figureImage(running *harness, offset int, figure string) image.Image {
	if offset%2 == 0 {
		return running.renderer.RenderClockDigit(figure)
	}

	return running.renderer.RenderClockSecondDigit(figure, offset < 5)
}

// requireCloser fails unless a key is drawn closer to one picture than to
// another.
func requireCloser(
	t *testing.T, device *testsupport.DeckDeviceTestDouble, keyIndex int, want image.Image, not image.Image,
) {
	t.Helper()

	drawn := drawnImage(t, device, keyIndex)
	if distance(drawn, want) >= distance(drawn, not) {
		t.Errorf("key %d: want it drawn as expected", keyIndex)
	}
}

// TestDaemon_DashboardShowsTheTimeAndTicks: the dashboard's top row shows the
// local time as HHMMSS, a figure to a key, in Thyme's look, with the world
// icon in its corner, and it ticks on each second.
func TestDaemon_DashboardShowsTheTimeAndTicks(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)

		holdMarker(device)
		requireTime(t, running, device, firstDigitKey, time.Now())
		requireCloser(t, device, worldKeyIndex,
			running.renderer.RenderClockIcon("world"), running.renderer.RenderBlank())

		time.Sleep(3 * time.Second)
		synctest.Wait()
		requireTime(t, running, device, firstDigitKey, time.Now())

		running.stop(t)
	})
}

// TestDaemon_WorldIconShowsFourClocks: the world icon shows UTC, London,
// Berlin and Sao Paulo, a row each, its label and then its HHMMSS, with the
// world icon kept in the top right; pressed again, it goes back to the
// dashboard.
func TestDaemon_WorldIconShowsFourClocks(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startDashboardHarness(t, 2)

		holdMarker(device)
		tap(device, worldKeyIndex)

		now := time.Now()

		for row, zone := range []struct {
			label string
			name  string
		}{{"UTC", "UTC"}, {"LDN", "Europe/London"}, {"BER", "Europe/Berlin"}, {"SAO", "America/Sao_Paulo"}} {
			location, loadError := time.LoadLocation(zone.name)
			if loadError != nil {
				t.Fatal(loadError)
			}

			requireCloser(t, device, row*keysPerRow,
				running.renderer.RenderClockLabel(zone.label), running.renderer.RenderClockLabel("XXX"))
			requireTime(t, running, device, row*keysPerRow+1, now.In(location))
		}

		requireCloser(t, device, worldKeyIndex,
			running.renderer.RenderClockIcon("world"), running.renderer.RenderBlank())

		tap(device, worldKeyIndex)
		tap(device, dashboardPageRow+1) // Second, on the dashboard again
		tap(device, besideKeyIndex)

		assertPresses(t, running, "second")

		running.stop(t)
	})
}
