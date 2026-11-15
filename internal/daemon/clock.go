package daemon

import (
	"context"
	"image"
	"log/slog"
	"time"

	// Embedded so the world clocks' zones load on a machine without the
	// zone files, as Windows has none, as Thyme does.
	_ "time/tzdata"
)

// The clocks: the dashboard's top row shows the local time as HHMMSS, a figure
// to a key, and the world icon in its corner shows the world clocks, Thyme's
// four zones a row each, its label and then its HHMMSS, the world icon kept in
// the top right to come back. Both tick on each second, and only the figures
// that change are drawn again.

// worldIcon is the bundled icon the world clocks are opened and left by.
const worldIcon = "world"

// timeFigures is how a time is written across its keys, HHMMSS.
const timeFigures = "150405"

// worldZone is one world clock: its label and where it is.
type worldZone struct {
	label    string
	location *time.Location
}

// worldZoneNames are Thyme's own four zones, by its labels, from its
// settings' defaults.
func worldZoneNames() [][2]string {
	return [][2]string{
		{"UTC", "UTC"}, {"LDN", "Europe/London"}, {"BER", "Europe/Berlin"}, {"SAO", "America/Sao_Paulo"},
	}
}

// loadWorldZones loads the world clocks, leaving out, and logging, any zone
// that will not load, so a wrong time is never shown under a right label.
func loadWorldZones(logger *slog.Logger) []worldZone {
	zones := make([]worldZone, 0, len(worldZoneNames()))

	for _, named := range worldZoneNames() {
		location, loadError := time.LoadLocation(named[1])
		if loadError != nil {
			logger.Warn("world clock not loaded", slog.String("zone", named[0]))

			continue
		}

		zones = append(zones, worldZone{label: named[0], location: location})
	}

	return zones
}

// clockTickEvent says a second has passed on the dashboard's clock.
type clockTickEvent struct{}

// startClock has the loop told of each second, on the second, until stopped.
func (state *loopState) startClock(runContext context.Context) chan struct{} {
	stop := make(chan struct{})

	go func() {
		for {
			now := time.Now()

			select {
			case <-time.After(now.Truncate(time.Second).Add(time.Second).Sub(now)):
				state.send(runContext, clockTickEvent{})
			case <-stop:
				return
			case <-runContext.Done():
				return
			}
		}
	}()

	return stop
}

// figureOf is the figure a clock key shows now: one of HHMMSS, local or in a
// world clock's zone.
func (state *loopState) figureOf(tile dashboardTile) string {
	now := time.Now()
	if tile.zone >= 0 && tile.zone < len(state.zones) {
		now = now.In(state.zones[tile.zone].location)
	}

	return now.Format(timeFigures)[tile.figure : tile.figure+1]
}

// clockImage draws a clock key's figure: a pair's first figure large, and its
// second a little smaller, with a small colon after the hours' and the
// minutes', so HH MM SS read as pairs.
func (state *loopState) clockImage(tile dashboardTile, figure string) image.Image {
	if tile.figure%figuresPerPair == 0 {
		return state.options.Renderer.RenderClockDigit(figure)
	}

	return state.options.Renderer.RenderClockSecondDigit(figure, tile.figure < clockFigures-1)
}

// figuresPerPair is how many figures each of the hours, minutes and seconds
// take.
const figuresPerPair = 2

// tickClock draws again the clock keys whose figures changed.
func (state *loopState) tickClock(runContext context.Context) {
	if !state.dashboard.open || state.connected == nil {
		return
	}

	for keyIndex, tile := range state.dashboardTiles() {
		if tile.role != dashboardDigit {
			continue
		}

		figure := state.figureOf(tile)
		if state.dashboard.drawn[keyIndex] == figure {
			continue
		}

		writeError := state.connected.SetKeyImage(keyIndex, state.clockImage(tile, figure))
		if writeError != nil {
			state.disconnect(runContext, "write failed")

			return
		}

		state.dashboard.drawn[keyIndex] = figure
	}
}
