package hidtransport_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/render/tablericons"
)

// TestExperiment1 paints every key with its index and logs every raw input
// report, so a person can confirm the key order, image orientation and input
// header on a real deck. Temporary, for experiment 1.
func TestExperiment1(t *testing.T) {
	if os.Getenv("THE_FRONT_BUTTONS_EXPERIMENT") != "1" {
		t.Skip("experiment 1 runs only on request")
	}

	watcher := hidtransport.NewWatcher()

	connections, enumerateError := watcher.Enumerate()
	if enumerateError != nil || len(connections) == 0 {
		t.Fatalf("no deck: %v", enumerateError)
	}

	model, _ := deck.LookupModel(connections[0].VendorID, connections[0].ProductID)

	device, openError := watcher.Open(connections[0])
	if openError != nil {
		t.Fatal(openError)
	}
	defer device.Close()

	icons, iconsError := tablericons.Load()
	if iconsError != nil {
		t.Fatal(iconsError)
	}

	renderer, rendererError := render.NewRenderer(model.Protocol.KeyPixels, icons)
	if rendererError != nil {
		t.Fatal(rendererError)
	}

	must(t, device.Write(deck.EncodeResetKeyStream(model)))

	brightness, _ := deck.EncodeBrightness(model, 70)
	must(t, device.SendFeatureReport(brightness))

	backgrounds := []string{"#1F4E79", "#7A1F1F", "#1F6B3A", "#5B2C83"}

	for keyIndex := range model.Rows * model.Columns {
		colour, _ := render.ParseColour(backgrounds[keyIndex/model.Columns])
		upright := renderer.Render(render.Appearance{
			Label: fmt.Sprintf("key %d", keyIndex), Icon: "arrow-up", IconImage: nil,
			Background: colour, Status: render.StatusNone, ErrorReason: "",
		})

		encoded, encodeError := deck.EncodeKeyImage(model, upright)
		must(t, encodeError)

		pages, pagesError := deck.EncodeKeyImagePages(model, keyIndex, encoded)
		must(t, pagesError)

		for _, page := range pages {
			must(t, device.Write(page))
		}
	}

	fmt.Fprintln(os.Stderr, "PAINTED: press keys now")

	deadline := time.Now().Add(experimentDuration())
	buffer := make([]byte, 512)

	for time.Now().Before(deadline) {
		readLength, readError := device.ReadInputReport(buffer, 250*time.Millisecond)
		if readError != nil {
			fmt.Fprintln(os.Stderr, "READ ERROR:", readError)

			return
		}

		if readLength == 0 {
			continue
		}

		var pressed []int

		for keyIndex := range model.Rows * model.Columns {
			offset := model.Protocol.InputHeaderLength + keyIndex
			if offset < readLength && buffer[offset] != 0 {
				pressed = append(pressed, keyIndex)
			}
		}

		nonZeroAfter := 0
		for offset := model.Protocol.InputReportLength; offset < readLength; offset++ {
			if buffer[offset] != 0 {
				nonZeroAfter++
			}
		}

		fmt.Fprintf(os.Stderr, "REPORT length=%d header=%s pressed=%v nonzero_after_36=%d\n",
			readLength, hex.EncodeToString(buffer[:min(readLength, 4)]), pressed, nonZeroAfter)
	}
}

func experimentDuration() time.Duration {
	seconds, parseError := time.ParseDuration(os.Getenv("THE_FRONT_BUTTONS_EXPERIMENT_FOR"))
	if parseError != nil {
		return 120 * time.Second
	}

	return seconds
}

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
