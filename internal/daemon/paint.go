package daemon

import (
	"context"
	"errors"
	"image"
	"log/slog"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// errNoIconFiles reports a key naming an icon file when the daemon was given no
// way to read icon files.
var errNoIconFiles = errors.New("no icons folder")

// errNoAppIcons reports a key asking for its application's icon when the
// daemon was given no way to read application icons.
var errNoAppIcons = errors.New("no way to read application icons")

// repaintAll draws every key of the current page: the full repaint on connect,
// on paging and on reload. A failed write means the deck has gone, so the deck
// is dropped and the next one to appear gets a full repaint of its own.
func (state *loopState) repaintAll(runContext context.Context) {
	if state.connected == nil {
		return
	}

	state.slots = state.arrangeCurrentPage(runContext)

	for keyIndex, slot := range state.slots {
		writeError := state.connected.SetKeyImage(keyIndex, state.imageFor(runContext, slot))
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
	if !onGrid || state.hiddenByOff(position) {
		return
	}

	keyImage := state.options.Renderer.Render(state.appearanceFor(runContext, configured))

	writeError := state.connected.SetKeyImage(keyIndex, keyImage)
	if writeError != nil {
		state.disconnect(runContext, "write failed")
	}
}

func (state *loopState) imageFor(runContext context.Context, slot layout.Slot) image.Image {
	renderer := state.options.Renderer

	if state.controls.off && (slot.Kind != layout.SlotBrick ||
		state.hiddenByOff(state.pages[state.paging.Current()][slot.BrickIndex].key.Position)) {
		return renderer.RenderBlank()
	}

	switch slot.Kind {
	case layout.SlotBack, layout.SlotForward:
		direction := render.DirectionForward
		if slot.Kind == layout.SlotBack {
			direction = render.DirectionBack
		}

		indication, hasIndication := state.paging.Indication(state.pageNames)

		return renderer.RenderNavigation(direction, indication, hasIndication)
	case layout.SlotBrick:
		return renderer.Render(state.appearanceFor(runContext, state.pages[state.paging.Current()][slot.BrickIndex]))
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
// malformed colours. A key showing its application's own icon takes that icon
// over its icon file, and the colour of the icon's edge unless it sets a
// background.
func (state *loopState) appearanceFor(runContext context.Context, configured configuredKey) render.Appearance {
	configuredAppearance, _ := brick.AppearanceOf(configured.key.Parameters)

	label := configuredAppearance.Label
	if label == "" {
		label = controlLabel(configured.instance)
	}

	appearance := render.Appearance{
		Label:       label,
		Icon:        configuredAppearance.Icon,
		ErrorReason: configured.errorReason,
	}

	if configuredAppearance.IconFile != "" {
		appearance.IconImage = state.iconFile(configuredAppearance.IconFile)
		appearance.IconMissing = appearance.IconImage == nil
	}

	if source, offers := configured.key.Parameters.(brick.ApplicationIconSource); offers {
		if identifier, wanted := source.ApplicationIcon(); wanted {
			appearance.IconImage = state.appIcon(runContext, identifier)
			appearance.IconMissing = appearance.IconImage == nil
			appearance.BackgroundFromIcon = true
		}
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

// iconFile returns a key's own icon, loading it once per configuration, or nil
// if it cannot be loaded, which is logged once with the reason.
func (state *loopState) iconFile(fileName string) image.Image {
	if icon, tried := state.iconFiles[fileName]; tried {
		return icon
	}

	var icon image.Image

	loadError := errNoIconFiles
	if state.options.IconFiles != nil {
		icon, loadError = state.options.IconFiles.Load(fileName)
	}

	if loadError != nil {
		icon = nil

		state.options.Logger.Warn("icon file not loaded",
			slog.String("file", fileName), slog.String("error", loadError.Error()))
	}

	state.iconFiles[fileName] = icon

	return icon
}

// appIcon returns an application's own icon, reading it once per
// configuration, or nil if it cannot be read, which is logged once with the
// reason.
func (state *loopState) appIcon(runContext context.Context, identifier string) image.Image {
	if icon, tried := state.appIcons[identifier]; tried {
		return icon
	}

	var icon image.Image

	readError := errNoAppIcons
	if state.options.AppIcons != nil {
		icon, readError = state.options.AppIcons.Icon(runContext, identifier)
	}

	if readError != nil {
		icon = nil

		state.options.Logger.WarnContext(runContext, "application icon not loaded",
			slog.String("application", identifier), slog.String("error", readError.Error()))
	}

	state.appIcons[identifier] = icon

	return icon
}
