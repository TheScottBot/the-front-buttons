package daemon

import (
	"context"
	"image"
	"log/slog"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// The clocks: the dashboard's top row shows the local time as HHMMSS, a figure
// to a key, and the world icon in its corner shows the world clocks, the
// configuration's or Thyme's four zones, a row each, its label and then its HHMMSS, the world icon kept in
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

// loadWorldZones loads the world clocks, leaving out, and logging, any zone
// that will not load, so a wrong time is never shown under a right label. The
// configuration's zones are checked as the file loads, and the zone database
// is embedded with it, so this is only a guard.
func loadWorldZones(clocks []config.WorldClock, logger *slog.Logger) []worldZone {
	zones := make([]worldZone, 0, len(clocks))

	for _, clock := range clocks {
		location, loadError := time.LoadLocation(clock.Zone)
		if loadError != nil {
			logger.Warn("world clock not loaded", slog.String("zone", clock.Label))

			continue
		}

		zones = append(zones, worldZone{label: clock.Label, location: location})
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

// dateParts is how the local date is written down the world clocks' right
// column: YY, MM and DD.
func dateParts() []string {
	return []string{"06", "01", "02"}
}

// dateWords name the date's parts beneath their figures.
func dateWords() []string {
	return []string{"year", "month", "day"}
}

// ticks reports a key the clock draws again as time passes: a figure of a
// time, or a part of the date.
func ticks(tile dashboardTile) bool {
	return tile.role == dashboardDigit || tile.role == dashboardDate
}

// figureOf is what a clock key shows now: one figure of HHMMSS, local or in
// a world clock's zone, or a part of the local date.
func (state *loopState) figureOf(tile dashboardTile) string {
	now := time.Now()
	if tile.role == dashboardDate {
		return now.Format(dateParts()[tile.figure])
	}

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

// clockKeyImage draws a clock key: a figure of a time, a zone's label, the
// world icon, or a part of the date.
func (state *loopState) clockKeyImage(tile dashboardTile) image.Image {
	renderer := state.options.Renderer

	switch tile.role { //nolint:exhaustive // only clock keys are drawn here; a figure is the rest
	case dashboardZoneLabel:
		return renderer.RenderClockLabel(state.zones[tile.zone].label)
	case dashboardWorld:
		return renderer.RenderClockIcon(worldIcon)
	case dashboardDate:
		return renderer.RenderClockDate(state.figureOf(tile), dateWords()[tile.figure])
	default:
		return state.clockImage(tile, state.figureOf(tile))
	}
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
		if !ticks(tile) {
			continue
		}

		figure := state.figureOf(tile)
		if state.dashboard.drawn[keyIndex] == figure {
			continue
		}

		writeError := state.connected.SetKeyImage(keyIndex, state.clockKeyImage(tile))
		if writeError != nil {
			state.disconnect(runContext, "write failed")

			return
		}

		state.dashboard.drawn[keyIndex] = figure
	}
}
