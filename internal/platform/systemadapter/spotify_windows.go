package systemadapter

import (
	"context"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// UnavailableMediaPlayer is the MediaPlayer on Windows, where Spotify offers
// no scripting: every command reports media control unavailable, and the
// keys say so. Windows is parked (evaluation, deck testing).
type UnavailableMediaPlayer struct{}

// NewMediaPlayer returns this machine's MediaPlayer.
func NewMediaPlayer() UnavailableMediaPlayer { return UnavailableMediaPlayer{} }

// NowPlaying reports media control unavailable.
func (UnavailableMediaPlayer) NowPlaying(context.Context) (platform.NowPlaying, error) {
	return platform.NowPlaying{Running: false, Playing: false, Artwork: nil}, platform.ErrMediaUnavailable
}

// PlayPause reports media control unavailable.
func (UnavailableMediaPlayer) PlayPause(context.Context) error { return platform.ErrMediaUnavailable }

// Previous reports media control unavailable.
func (UnavailableMediaPlayer) Previous(context.Context) error { return platform.ErrMediaUnavailable }

// Next reports media control unavailable.
func (UnavailableMediaPlayer) Next(context.Context) error { return platform.ErrMediaUnavailable }

// Show reports media control unavailable.
func (UnavailableMediaPlayer) Show(context.Context) error { return platform.ErrMediaUnavailable }
