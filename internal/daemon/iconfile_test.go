package daemon_test

import (
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The one icon file the fake folder holds: a solid red square.
const redIconFile = "red.png"

var redIcon = func() image.Image { //nolint:gochecknoglobals // a fixed test picture
	picture := image.NewRGBA(image.Rect(0, 0, 32, 32))

	for x := range 32 {
		for y := range 32 {
			picture.Set(x, y, color.RGBA{R: 0xff, G: 0, B: 0, A: 0xff})
		}
	}

	return picture
}()

// iconFilesTestDouble is an icons folder holding red.png and nothing else.
type iconFilesTestDouble struct{}

func (iconFilesTestDouble) Load(fileName string) (image.Image, error) {
	if fileName == redIconFile {
		return redIcon, nil
	}

	return nil, fmt.Errorf("no icon file %q: %w", fileName, fs.ErrNotExist)
}

func iconFileKey(column int, fileName string) config.Key {
	return config.Key{
		Position: layout.Position{Row: 0, Column: column},
		Kind:     testsupport.ButtonKindName,
		Parameters: &testsupport.ButtonParameters{
			Label: buildLabel, Icon: "", Background: "", IconFile: fileName, Name: fileName, Fails: false,
		},
		Preset: nil,
	}
}

// TestDaemon_DrawsAKeysOwnIconFile_OrTheMissingMarker: a key may name a PNG of
// the person's own in the icons folder (the author's request, evaluation, deck
// testing). It is drawn on the key; a file that cannot be loaded shows the
// missing icon marker, as an unknown bundled icon does.
func TestDaemon_DrawsAKeysOwnIconFile_OrTheMissingMarker(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- config.Configuration{
			SchemaVersion: config.CurrentSchemaVersion,
			Pages: []config.Page{{Name: "Icons", Keys: []config.Key{
				iconFileKey(0, redIconFile), iconFileKey(1, "gone.png"),
			}}},
		}

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		withIcon := running.renderer.Render(render.Appearance{Label: buildLabel, IconImage: redIcon})
		missing := running.renderer.Render(render.Appearance{Label: buildLabel, IconMissing: true})

		if drawn := drawnImage(t, device, 0); distance(drawn, withIcon) >= distance(drawn, missing) {
			t.Error("want red.png drawn on the first key")
		}

		if drawn := drawnImage(t, device, 1); distance(drawn, missing) >= distance(drawn, withIcon) {
			t.Error("want the missing marker on the key whose file is gone")
		}

		running.stop(t)
	})
}
