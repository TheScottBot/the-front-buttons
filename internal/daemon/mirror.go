package daemon

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/deckmirror"
)

// The mirror is what the editor may see of the deck as it runs: the pictures
// of keys whose look depends on live state, on every page whether showing or
// not, drawn as the deck draws them, and word of each press. A key whose look
// never changes is left to the editor's own preview, which draws it the same.

// mirrored reports a key whose look depends on live state: a live brick, a key
// showing its application's badge, or a key showing a press's error.
func mirrored(configured configuredKey) bool {
	_, isLive := configured.instance.(brick.LiveSource)
	_, wantsBadge := wantedBadge(configured.key.Parameters)

	return isLive || wantsBadge || configured.errorReason != ""
}

// mirrorAll gives the mirror every such key's picture, for a new
// configuration.
func (state *loopState) mirrorAll(runContext context.Context) {
	if state.options.Mirror == nil {
		return
	}

	frames := map[deckmirror.Tile]deckmirror.Frame{}

	for pageIndex, keys := range state.pages {
		for _, configured := range keys {
			if mirrored(configured) {
				frames[tileOf(pageIndex, configured)] = deckmirror.Frame{
					Kind: configured.key.Kind, Picture: state.options.Renderer.Render(state.appearanceFor(runContext, configured)),
				}
			}
		}
	}

	state.options.Mirror.Replace(frames)
}

// mirrorWhere redraws in the mirror every key, on every page, that a change
// touches.
func (state *loopState) mirrorWhere(runContext context.Context, touched func(configured configuredKey) bool) {
	if state.options.Mirror == nil {
		return
	}

	for pageIndex, keys := range state.pages {
		for _, configured := range keys {
			if touched(configured) {
				state.mirrorKey(runContext, pageIndex, configured)
			}
		}
	}
}

// mirrorKey redraws one key in the mirror.
func (state *loopState) mirrorKey(runContext context.Context, pageIndex int, configured configuredKey) {
	if state.options.Mirror == nil {
		return
	}

	state.options.Mirror.Set(tileOf(pageIndex, configured), configured.key.Kind,
		state.options.Renderer.Render(state.appearanceFor(runContext, configured)))
}

// mirrorPress tells the mirror a key on the page showing was pressed, and
// keeps its tile for the release.
func (state *loopState) mirrorPress(keyIndex int) {
	pageIndex, showing := state.showingPage()
	position, onGrid := state.grid.PositionOfKey(keyIndex)

	if showing && onGrid {
		tile := deckmirror.Tile{Page: pageIndex, Position: position}
		state.pressedTiles[keyIndex] = tile
		state.options.Mirror.Press(tile)
	}
}

// mirrorRelease tells the mirror a pressed key was let go of, on the tile it
// was pressed on.
func (state *loopState) mirrorRelease(keyIndex int) {
	tile, pressed := state.pressedTiles[keyIndex]
	if !pressed {
		return
	}

	delete(state.pressedTiles, keyIndex)
	state.options.Mirror.Release(tile)
}

// releaseAllPressed lets go of every key held down when the deck goes, as no
// release will come for them.
func (state *loopState) releaseAllPressed() {
	for keyIndex := range state.pressedTiles {
		state.mirrorRelease(keyIndex)
	}
}

func tileOf(pageIndex int, configured configuredKey) deckmirror.Tile {
	return deckmirror.Tile{Page: pageIndex, Position: configured.key.Position}
}
