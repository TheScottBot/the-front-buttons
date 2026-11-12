package hidtransport_test

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
)

// TestExperimentWrites paints full pages repeatedly, first alone and then with
// a reader running as the daemon's does, logging every write error with its
// text and how long the writes took. Temporary, for the paging investigation.
func TestExperimentWrites(t *testing.T) {
	if os.Getenv("THE_FRONT_BUTTONS_EXPERIMENT") != "1" {
		t.Skip("runs only on request")
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

	must(t, device.Write(deck.EncodeResetKeyStream(model)))

	keyCount := model.Rows * model.Columns
	images := make([][][]byte, keyCount)
	totalPages := 0

	for keyIndex := range keyCount {
		upright := image.NewRGBA(image.Rect(0, 0, model.Protocol.KeyPixels, model.Protocol.KeyPixels))
		for x := range model.Protocol.KeyPixels {
			for y := range model.Protocol.KeyPixels {
				upright.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 2), B: uint8(keyIndex * 8), A: 255})
			}
		}

		encoded, encodeError := deck.EncodeKeyImage(model, upright)
		must(t, encodeError)

		pages, pagesError := deck.EncodeKeyImagePages(model, keyIndex, encoded)
		must(t, pagesError)

		images[keyIndex] = pages
		totalPages += len(pages)
	}

	fmt.Fprintf(os.Stderr, "pages per full repaint: %d\n", totalPages)

	paint := func(label string, repaints int) {
		failures := 0
		start := time.Now()
		slowest := time.Duration(0)

		for range repaints {
			for _, pages := range images {
				for _, page := range pages {
					writeStart := time.Now()

					writeError := device.Write(page)
					slowest = max(slowest, time.Since(writeStart))

					if writeError != nil {
						failures++
						fmt.Fprintf(os.Stderr, "%s WRITE ERROR after %v: %v\n", label, time.Since(writeStart), writeError)
					}
				}
			}
		}

		fmt.Fprintf(os.Stderr, "%s: %d repaints in %v (%v each), slowest write %v, %d failures\n",
			label, repaints, time.Since(start), time.Since(start)/time.Duration(repaints), slowest, failures)
	}

	paint("alone", 10)

	stop := make(chan struct{})

	var group sync.WaitGroup

	group.Go(func() {
		buffer := make([]byte, model.Protocol.InputReportLength)

		for {
			select {
			case <-stop:
				return
			default:
			}

			_, readError := device.ReadInputReport(buffer, 50*time.Millisecond)
			if readError != nil {
				fmt.Fprintln(os.Stderr, "READ ERROR:", readError)

				return
			}
		}
	})

	paint("with reader", 3)
	close(stop)
	group.Wait()
}
