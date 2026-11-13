package systemadapter

import (
	"context"
	"fmt"
	"image"
	"strings"
	"sync"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// spotifyArtworkPrefix is where Spotify serves album art. Art anywhere else is
// never fetched, so a script's output cannot send the daemon to another host.
const spotifyArtworkPrefix = "https://i.scdn.co/image/"

// The lines the now playing script answers with: running or stopped, then the
// player's state and the current track's art address.
const (
	spotifyStateLine   = 1
	spotifyArtworkLine = 2
)

// spotifyNotAuthorised is AppleScript's error number when the Automation
// permission to control an application has not been given.
const spotifyNotAuthorised = "-1743"

// The scripts the Spotify adapter runs. Reading what is playing checks first
// that Spotify is running, because telling an application anything launches
// it; the commands may launch it, as pressing play on a closed player should.
// Inside the tell block a name Spotify defines is Spotify's own property, so
// the script's own names must be none of Spotify's: "artwork" was Spotify's
// read only property, and setting it failed every read (evaluation, deck
// testing).
const (
	spotifyNowPlayingScript = `if application "Spotify" is running then
	tell application "Spotify"
		set coverAddress to ""
		try
			set coverAddress to artwork url of current track
		end try
		return "running" & linefeed & (player state as string) & linefeed & coverAddress
	end tell
else
	return "stopped"
end if`
	spotifyPlayPauseScript = `tell application "Spotify" to playpause`
	spotifyPreviousScript  = `tell application "Spotify" to previous track`
	spotifyNextScript      = `tell application "Spotify" to next track`
	spotifyShowScript      = `tell application "Spotify" to activate`
)

// ScriptRunner runs one AppleScript and returns its output.
type ScriptRunner func(runContext context.Context, script string) (string, error)

// ArtworkFetcher reads one album art image from its address.
type ArtworkFetcher func(fetchContext context.Context, address string) (image.Image, error)

// Spotify is the MediaPlayer for Spotify, through AppleScript, for the 2x2 the
// author asked for (evaluation, deck testing). It keeps the last track's art,
// so each track's art is fetched once however often it is read.
type Spotify struct {
	run   ScriptRunner
	fetch ArtworkFetcher

	mutex      sync.Mutex
	artworkURL string
	artwork    image.Image
}

// NewSpotify returns a Spotify player running scripts and fetching art with
// the functions given.
func NewSpotify(run ScriptRunner, fetch ArtworkFetcher) *Spotify {
	return &Spotify{run: run, fetch: fetch, mutex: sync.Mutex{}, artworkURL: "", artwork: nil}
}

// NowPlaying reads whether Spotify is running and playing, and the current
// track's art.
func (player *Spotify) NowPlaying(nowContext context.Context) (platform.NowPlaying, error) {
	output, runError := player.script(nowContext, spotifyNowPlayingScript)
	if runError != nil {
		return platform.NowPlaying{Running: false, Playing: false, Artwork: nil}, runError
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if lines[0] != "running" {
		return platform.NowPlaying{Running: false, Playing: false, Artwork: nil}, nil
	}

	playing := len(lines) > spotifyStateLine && strings.TrimSpace(lines[spotifyStateLine]) == "playing"

	artworkURL := ""
	if len(lines) > spotifyArtworkLine {
		artworkURL = strings.TrimSpace(lines[spotifyArtworkLine])
	}

	return platform.NowPlaying{Running: true, Playing: playing, Artwork: player.artworkFor(nowContext, artworkURL)}, nil
}

// PlayPause plays or pauses.
func (player *Spotify) PlayPause(pressContext context.Context) error {
	return player.command(pressContext, spotifyPlayPauseScript)
}

// Previous goes to the previous track.
func (player *Spotify) Previous(pressContext context.Context) error {
	return player.command(pressContext, spotifyPreviousScript)
}

// Next goes to the next track.
func (player *Spotify) Next(pressContext context.Context) error {
	return player.command(pressContext, spotifyNextScript)
}

// Show brings Spotify to the front.
func (player *Spotify) Show(pressContext context.Context) error {
	return player.command(pressContext, spotifyShowScript)
}

// artworkFor is the art at an address on Spotify's image host, fetched once
// per address; nil for any other address, or art that could not be read.
func (player *Spotify) artworkFor(fetchContext context.Context, address string) image.Image {
	if !strings.HasPrefix(address, spotifyArtworkPrefix) {
		return nil
	}

	player.mutex.Lock()
	defer player.mutex.Unlock()

	if address == player.artworkURL {
		return player.artwork
	}

	artwork, fetchError := player.fetch(fetchContext, address)
	if fetchError != nil {
		return nil
	}

	player.artworkURL, player.artwork = address, artwork

	return artwork
}

func (player *Spotify) command(pressContext context.Context, script string) error {
	_, runError := player.script(pressContext, script)

	return runError
}

// script runs a script, naming macOS's refusal of the Automation permission.
func (player *Spotify) script(runContext context.Context, script string) (string, error) {
	output, runError := player.run(runContext, script)
	if runError == nil {
		return output, nil
	}

	if strings.Contains(runError.Error(), spotifyNotAuthorised) {
		return "", fmt.Errorf("%w: %w", platform.ErrMediaNotAllowed, runError)
	}

	return "", fmt.Errorf("controlling Spotify: %w", runError)
}
