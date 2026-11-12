package daemon

import (
	"context"
	"image"
	"log/slog"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// repaintAll draws every key of the current page: the full repaint on connect,
// on paging and on reload. A failed write means the deck has gone, so the deck
// is dropped and the next one to appear gets a full repaint of its own.
func (state *loopState) repaintAll(runContext context.Context) {
	if state.connected == nil {
		return
	}

	state.slots = state.arrangeCurrentPage(runContext)

	for keyIndex, slot := range state.slots {
		writeError := state.connected.SetKeyImage(keyIndex, state.imageFor(slot))
		if writeError != nil {
			state.disconnect(runContext, "write failed")

			return
		}
	}
}

// arrangeCurrentPage lays out the current page. Configuration was validated
// against a grid when decoded; if the attached deck's grid differs, the page is
// shown with navigation only and the mismatch is logged, rather than drawing
// bricks on the wrong keys.
func (state *loopState) arrangeCurrentPage(runContext context.Context) []layout.Slot {
	var positions []layout.Position

	if state.paging.HasPage() {
		for _, configured := range state.pages[state.paging.Current()] {
			positions = append(positions, configured.key.Position)
		}
	}

	slots, arrangeError := state.grid.Arrange(positions)
	if arrangeError == nil {
		return slots
	}

	state.options.Logger.WarnContext(runContext, "page does not fit the attached deck",
		slog.Int("page", state.paging.Current()))

	navigationOnly, _ := state.grid.Arrange(nil)

	return navigationOnly
}

func (state *loopState) repaintKey(runContext context.Context, position layout.Position, configured configuredKey) {
	if state.connected == nil {
		return
	}

	keyIndex, onGrid := state.grid.KeyIndex(position)
	if !onGrid {
		return
	}

	writeError := state.connected.SetKeyImage(keyIndex, state.options.Renderer.Render(state.appearanceFor(configured)))
	if writeError != nil {
		state.disconnect(runContext, "write failed")
	}
}

func (state *loopState) imageFor(slot layout.Slot) image.Image {
	renderer := state.options.Renderer

	switch slot.Kind {
	case layout.SlotBack, layout.SlotForward:
		direction := render.DirectionForward
		if slot.Kind == layout.SlotBack {
			direction = render.DirectionBack
		}

		indication, hasIndication := state.paging.Indication(state.pageNames)

		return renderer.RenderNavigation(direction, indication, hasIndication)
	case layout.SlotBrick:
		return renderer.Render(state.appearanceFor(state.pages[state.paging.Current()][slot.BrickIndex]))
	case layout.SlotEmpty:
		return renderer.RenderBlank()
	default:
		return renderer.RenderBlank()
	}
}

// appearanceFor turns a key's configured appearance, its live state and its
// error state into what the renderer draws. A press failure outranks a live
// error, since it is the newer news. A background the renderer cannot parse
// falls back to the default; the configuration decoder has already refused
// malformed colours.
func (state *loopState) appearanceFor(configured configuredKey) render.Appearance {
	configuredAppearance, _ := brick.AppearanceOf(configured.key.Parameters)

	appearance := render.Appearance{
		Label:       configuredAppearance.Label,
		Icon:        configuredAppearance.Icon,
		ErrorReason: configured.errorReason,
	}

	if source, isLive := configured.instance.(brick.LiveSource); isLive {
		if liveState, published := state.hub.State(source.LiveIdentity()); published {
			appearance.Status = liveState.Status

			if appearance.ErrorReason == "" {
				appearance.ErrorReason = liveState.ErrorReason
			}
		}
	}

	if configuredAppearance.Background != "" {
		background, parseError := render.ParseColour(configuredAppearance.Background)
		if parseError == nil {
			appearance.Background = background
		}
	}

	return appearance
}
