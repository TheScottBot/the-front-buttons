package hidtransport_test

import (
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/render/tablericons"
)

// The experiments run only on request, with a deck attached and a person at
// it: experiment 1 needs keys pressed, and both draw over whatever the deck
// shows. TESTING.md says how to run them; the evaluation record holds the
// MacBook's results, which the Dell's are to be compared with.
const (
	experimentVariable      = "THE_FRONT_BUTTONS_EXPERIMENT"
	experimentForVariable   = "THE_FRONT_BUTTONS_EXPERIMENT_FOR"
	defaultExperimentLength = 2 * time.Minute
	experimentPollTimeout   = 250 * time.Millisecond
	experimentBrightness    = 70
	rawReportBufferLength   = 512
	headerBytesShown        = 4
	paintsAlone             = 10
	paintsWithReader        = 3
	readerTimeout           = 50 * time.Millisecond
)

// rowBackgrounds colours each row of experiment 1's keys differently, so a
// mirrored or rotated layout shows at a glance.
var rowBackgrounds = [...]string{"#1F4E79", "#7A1F1F", "#1F6B3A", "#5B2C83"} //nolint:gochecknoglobals // a fixed table

// openExperimentDeck skips unless experiments were asked for, and otherwise
// opens the first attached deck with its key stream reset. The deck is closed,
// and the close checked, when the test ends.
//
//nolint:ireturn // the port's own type is what the experiments drive
func openExperimentDeck(t *testing.T) (deck.Model, deck.DeckDevice) {
	t.Helper()

	if os.Getenv(experimentVariable) != "1" {
		t.Skipf("set %s=1 with a deck attached and a person at it", experimentVariable)
	}

	watcher := hidtransport.NewWatcher()

	connections, enumerateError := watcher.Enumerate()
	if enumerateError != nil || len(connections) == 0 {
		t.Fatalf("want an attached deck, found %d (%v)", len(connections), enumerateError)
	}

	model, _ := deck.LookupModel(connections[0].VendorID, connections[0].ProductID)

	device, openError := watcher.Open(connections[0])
	if openError != nil {
		t.Fatal(openError)
	}

	t.Cleanup(func() {
		closeError := device.Close()
		if closeError != nil {
			t.Errorf("closing the deck: %v", closeError)
		}
	})

	failOn(t, device.Write(deck.EncodeResetKeyStream(model)))

	return model, device
}

// TestExperiment1_PaintsKeyIndexesAndLogsRawReports is experiment 1: every key
// shows its index with an up arrow, each row in its own colour, and every raw
// input report is logged while a person presses keys. The person checks the
// order and orientation by eye; the log gives the report length, its header and
// the indexes read as pressed.
//
//nolint:paralleltest // drives the one physical deck, which two experiments cannot share
func TestExperiment1_PaintsKeyIndexesAndLogsRawReports(t *testing.T) {
	model, device := openExperimentDeck(t)

	brightness, encodeError := deck.EncodeBrightness(model, experimentBrightness)
	failOn(t, encodeError)
	failOn(t, device.SendFeatureReport(brightness))

	paintKeyIndexes(t, model, device)

	length := experimentLength()
	t.Logf("painted; press keys for %v", length)

	deadline := time.Now().Add(length)
	buffer := make([]byte, rawReportBufferLength)

	for time.Now().Before(deadline) {
		readLength, readError := device.ReadInputReport(buffer, experimentPollTimeout)
		failOn(t, readError)

		if readLength > 0 {
			t.Log(describeReport(model, buffer[:readLength]))
		}
	}
}

func paintKeyIndexes(t *testing.T, model deck.Model, device deck.DeckDevice) {
	t.Helper()

	icons, iconsError := tablericons.Load()
	failOn(t, iconsError)

	renderer, rendererError := render.NewRenderer(model.Protocol.KeyPixels, icons)
	failOn(t, rendererError)

	for keyIndex := range model.Rows * model.Columns {
		background, colourError := render.ParseColour(rowBackgrounds[keyIndex/model.Columns%len(rowBackgrounds)])
		failOn(t, colourError)

		upright := renderer.Render(render.Appearance{
			Label: fmt.Sprintf("key %d", keyIndex), Icon: "arrow-up", IconImage: nil,
			Background: background, Status: render.StatusNone, ErrorReason: "",
		})

		writeImage(t, model, device, keyIndex, upright)
	}
}

// describeReport names a raw report's length, its first bytes, the key indexes
// read as pressed, and how many bytes past the model's report length are not
// zero, which would mean the deck says more than the driver reads.
func describeReport(model deck.Model, report []byte) string {
	var pressed []int

	for keyIndex := range model.Rows * model.Columns {
		offset := model.Protocol.InputHeaderLength + keyIndex
		if offset < len(report) && report[offset] != 0 {
			pressed = append(pressed, keyIndex)
		}
	}

	nonZeroBeyond := 0

	for _, extraByte := range report[min(len(report), model.Protocol.InputReportLength):] {
		if extraByte != 0 {
			nonZeroBeyond++
		}
	}

	return fmt.Sprintf("report length=%d header=%s pressed=%v non_zero_beyond_%d=%d",
		len(report), hex.EncodeToString(report[:min(len(report), headerBytesShown)]), pressed,
		model.Protocol.InputReportLength, nonZeroBeyond)
}

// TestExperimentWrites_DrawsFullPagesQuickly_WithAndWithoutAReader times full
// pages written alone and with a reader running as the daemon's does, which is
// how the paging delay and the false disconnects were found (evaluation, deck
// testing). Any write that fails after the transport's retries fails the test;
// the timings are logged for comparison between machines.
//
//nolint:paralleltest // drives the one physical deck, which two experiments cannot share
func TestExperimentWrites_DrawsFullPagesQuickly_WithAndWithoutAReader(t *testing.T) {
	model, device := openExperimentDeck(t)
	pages := encodeTestPattern(t, model)

	t.Logf("pages per full repaint: %d", len(pages))
	timePaints(t, "alone", device, pages, paintsAlone)

	stop := make(chan struct{})

	var reader sync.WaitGroup

	reader.Go(func() { readUntilStopped(t, model, device, stop) })

	timePaints(t, "with a reader", device, pages, paintsWithReader)
	close(stop)
	reader.Wait()
}

// encodeTestPattern encodes a gradient for every key, a worst case for the
// JPEG size, and returns every page of every key in drawing order.
func encodeTestPattern(t *testing.T, model deck.Model) [][]byte {
	t.Helper()

	keyPixels := model.Protocol.KeyPixels

	var pages [][]byte

	for keyIndex := range model.Rows * model.Columns {
		upright := image.NewRGBA(image.Rect(0, 0, keyPixels, keyPixels))

		for row := range keyPixels {
			for column := range keyPixels {
				upright.Set(column, row, color.RGBA{
					R: uint8(column * 255 / keyPixels), G: uint8(row * 255 / keyPixels), //nolint:gosec // below 256
					B: uint8(keyIndex * 255 / (model.Rows * model.Columns)), A: 255, //nolint:gosec // below 256
				})
			}
		}

		encoded, encodeError := deck.EncodeKeyImage(model, upright)
		failOn(t, encodeError)

		keyPages, pagesError := deck.EncodeKeyImagePages(model, keyIndex, encoded)
		failOn(t, pagesError)

		pages = append(pages, keyPages...)
	}

	return pages
}

func timePaints(t *testing.T, label string, device deck.DeckDevice, pages [][]byte, repaints int) {
	t.Helper()

	start := time.Now()
	slowest := time.Duration(0)

	for range repaints {
		for _, page := range pages {
			writeStart := time.Now()
			writeError := device.Write(page)
			slowest = max(slowest, time.Since(writeStart))

			if writeError != nil {
				t.Errorf("%s: a write failed after the transport's retries: %v", label, writeError)
			}
		}
	}

	took := time.Since(start)
	t.Logf("%s: %d repaints in %v, %v each, slowest write %v",
		label, repaints, took, took/time.Duration(repaints), slowest)
}

func readUntilStopped(t *testing.T, model deck.Model, device deck.DeckDevice, stop <-chan struct{}) {
	t.Helper()

	buffer := make([]byte, model.Protocol.InputReportLength)

	for {
		select {
		case <-stop:
			return
		default:
		}

		_, readError := device.ReadInputReport(buffer, readerTimeout)
		if readError != nil {
			t.Errorf("reading alongside the writes: %v", readError)

			return
		}
	}
}

func writeImage(t *testing.T, model deck.Model, device deck.DeckDevice, keyIndex int, upright image.Image) {
	t.Helper()

	encoded, encodeError := deck.EncodeKeyImage(model, upright)
	failOn(t, encodeError)

	pages, pagesError := deck.EncodeKeyImagePages(model, keyIndex, encoded)
	failOn(t, pagesError)

	for _, page := range pages {
		failOn(t, device.Write(page))
	}
}

// experimentLength is how long experiment 1 listens for presses: two minutes,
// or the duration THE_FRONT_BUTTONS_EXPERIMENT_FOR gives, such as 3m.
func experimentLength() time.Duration {
	length, parseError := time.ParseDuration(os.Getenv(experimentForVariable))
	if parseError != nil || length <= 0 {
		return defaultExperimentLength
	}

	return length
}

func failOn(t *testing.T, failure error) {
	t.Helper()

	if failure != nil {
		t.Fatal(failure)
	}
}
