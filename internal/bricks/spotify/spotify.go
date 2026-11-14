// Package spotify is the author's first 2x2 (evaluation, deck testing): four
// keys for Spotify. spotify_album shows the current track's art and brings
// Spotify to the front; spotify_play_pause plays or pauses, showing which a
// press will do; spotify_previous and spotify_next change track. The album and
// play keys share one source, so Spotify is read once however many keys show
// it.
package spotify

import (
	"context"
	"fmt"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The kinds' names in configuration.
const (
	AlbumKindName     = "spotify_album"
	PlayPauseKindName = "spotify_play_pause"
	PreviousKindName  = "spotify_previous"
	NextKindName      = "spotify_next"
)

// sourceIdentity is the one source the album and play keys share.
const sourceIdentity = "spotify"

// How often Spotify is read: each second while it is running, so a new track
// or a press of play shows within a moment, and every five seconds while it is
// closed.
const (
	runningInterval = time.Second
	closedInterval  = 5 * time.Second
)

// readTimeout bounds one read of Spotify, so a script stuck waiting, as one
// is while macOS asks for the Automation permission, cannot freeze the keys.
const readTimeout = 10 * time.Second

// The play key's icons: the one a press will do.
const (
	playIcon  = "player-play"
	pauseIcon = "player-pause"
	// albumPlaceholder stands in for the album art where there is none.
	albumPlaceholder = "vinyl"
)

// Parameters is every Spotify kind's one declaration of its parameters: only
// its appearance.
type Parameters struct {
	brick.Appearance
}

// Register adds the four kinds to a registry.
func Register(registry *brick.Registry) error {
	for kindName, build := range map[string]func(platform.MediaPlayer) any{
		AlbumKindName:     func(player platform.MediaPlayer) any { return album{player: player} },
		PlayPauseKindName: func(player platform.MediaPlayer) any { return playPause{player: player} },
		PreviousKindName:  func(player platform.MediaPlayer) any { return command{player: player, send: player.Previous} },
		NextKindName:      func(player platform.MediaPlayer) any { return command{player: player, send: player.Next} },
	} {
		registerError := brick.Register(registry, kindName, func(_ Parameters, ports brick.Ports) (any, error) {
			if ports.MediaPlayer == nil {
				return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, kindName)
			}

			return build(ports.MediaPlayer), nil
		})
		if registerError != nil {
			return registerError //nolint:wrapcheck // Register already names the kind
		}
	}

	// Album art and whether a track plays are live; the editor never sees them.
	for kindName, icon := range map[string]string{AlbumKindName: albumPlaceholder, PlayPauseKindName: playIcon} {
		placeholderError := registry.SetPlaceholderIcon(kindName, icon)
		if placeholderError != nil {
			return placeholderError //nolint:wrapcheck // it names the kind
		}
	}

	return nil
}

// pressed names a failed command by its category on the key.
func pressed(name string, failure error) error {
	if failure == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", name, brick.Categorised(platform.MediaFailureCategory(failure), failure))
}

// watch reads Spotify until the context ends, publishing whenever what it
// shows changes: the art, playing or not, or why it cannot be read.
func watch(watchContext context.Context, player platform.MediaPlayer, publish func(state brick.LiveState)) {
	var last brick.LiveState

	for first := true; ; first = false {
		readContext, cancel := context.WithTimeout(watchContext, readTimeout)
		playing, readError := player.NowPlaying(readContext)

		cancel()

		state := brick.LiveState{
			Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: nil, Playing: false,
		}

		switch {
		case readError != nil:
			state.ErrorReason = platform.MediaFailureCategory(readError)
		case playing.Running:
			state.Picture, state.Playing = playing.Artwork, playing.Playing
		}

		if first || state != last {
			publish(state)
			last = state
		}

		wait := closedInterval
		if playing.Running {
			wait = runningInterval
		}

		select {
		case <-watchContext.Done():
			return
		case <-time.After(wait):
		}
	}
}

// album shows the current track's art, and brings Spotify to the front.
type album struct {
	player platform.MediaPlayer
}

func (key album) Press(pressContext context.Context) error {
	return pressed("showing Spotify", key.player.Show(pressContext))
}

func (album) LiveIdentity() string { return sourceIdentity }

func (key album) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	watch(watchContext, key.player, publish)
}

// DrawLive fills the key with the art; without art the key shows its label
// and icon.
func (album) DrawLive(state brick.LiveState, appearance *render.Appearance) {
	appearance.Picture = state.Picture
}

// playPause plays or pauses, showing the icon of what a press will do.
type playPause struct {
	player platform.MediaPlayer
}

func (key playPause) Press(pressContext context.Context) error {
	return pressed("playing or pausing", key.player.PlayPause(pressContext))
}

func (playPause) LiveIdentity() string { return sourceIdentity }

func (key playPause) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	watch(watchContext, key.player, publish)
}

// DrawLive shows pause while playing and play otherwise.
func (playPause) DrawLive(state brick.LiveState, appearance *render.Appearance) {
	appearance.Icon = playIcon
	if state.Playing {
		appearance.Icon = pauseIcon
	}
}

// command sends one command, previous or next track.
type command struct {
	player platform.MediaPlayer
	send   func(pressContext context.Context) error
}

func (key command) Press(pressContext context.Context) error {
	return pressed("changing track", key.send(pressContext))
}
