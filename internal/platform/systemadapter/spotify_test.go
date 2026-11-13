package systemadapter_test

import (
	"context"
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// The artwork address the Spotify tests hand back, on Spotify's image host.
const spotifyArtwork = "https://i.scdn.co/image/ab67616d0000b273abc"

// errNotAuthorised is osascript's refusal when the Automation permission for
// Spotify has not been given.
var errNotAuthorised = errors.New("execution error: Not authorised to send Apple events to Spotify. (-1743)")

// spotifyFake answers scripts with fixed output and counts artwork fetches.
type spotifyFake struct {
	output  string
	failure error
	scripts []string
	fetched []string
}

func (fake *spotifyFake) run(_ context.Context, script string) (string, error) {
	fake.scripts = append(fake.scripts, script)

	return fake.output, fake.failure
}

func (fake *spotifyFake) fetch(_ context.Context, address string) (image.Image, error) {
	fake.fetched = append(fake.fetched, address)

	return image.NewRGBA(image.Rect(0, 0, 4, 4)), nil
}

func newSpotify(fake *spotifyFake) *systemadapter.Spotify {
	return systemadapter.NewSpotify(fake.run, fake.fetch)
}

// TestSpotify_ReadsWhatIsPlaying: the album key shows the current track's
// art and the play key whether it is playing (the author's 2x2, evaluation,
// deck testing). The art is fetched once per track.
func TestSpotify_ReadsWhatIsPlaying(t *testing.T) {
	t.Parallel()

	fake := &spotifyFake{output: "running\nplaying\n" + spotifyArtwork + "\n", failure: nil, scripts: nil, fetched: nil}
	player := newSpotify(fake)

	for range 2 {
		playing, readError := player.NowPlaying(t.Context())
		if readError != nil || !playing.Running || !playing.Playing || playing.Artwork == nil {
			t.Fatalf("want a running, playing player with art, got %+v (%v)", playing, readError)
		}
	}

	if len(fake.fetched) != 1 || fake.fetched[0] != spotifyArtwork {
		t.Errorf("want the art fetched once, got %v", fake.fetched)
	}

	if !strings.Contains(fake.scripts[0], `application "Spotify" is running`) {
		t.Error("want the script to check Spotify is running, so reading it never launches it")
	}
}

// TestSpotify_ReportsAStoppedOrPausedPlayer: a closed Spotify is not
// running, a paused one not playing, and art anywhere but Spotify's image
// host is never fetched.
func TestSpotify_ReportsAStoppedOrPausedPlayer(t *testing.T) {
	t.Parallel()

	closed := &spotifyFake{output: "stopped\n", failure: nil, scripts: nil, fetched: nil}

	stopped, _ := newSpotify(closed).NowPlaying(t.Context())
	if stopped.Running || stopped.Playing || stopped.Artwork != nil {
		t.Errorf("want a stopped player, got %+v", stopped)
	}

	fake := &spotifyFake{
		output: "running\npaused\nhttps://example.test/art.jpg\n", failure: nil, scripts: nil, fetched: nil,
	}

	paused, _ := newSpotify(fake).NowPlaying(t.Context())
	if !paused.Running || paused.Playing || paused.Artwork != nil || len(fake.fetched) != 0 {
		t.Errorf("want a paused player with no art fetched, got %+v, fetched %v", paused, fake.fetched)
	}
}

// TestSpotify_SendsEachCommandAndNamesARefusal: each key's command is a
// script naming it, and macOS refusing the Automation permission is told
// apart from any other failure.
func TestSpotify_SendsEachCommandAndNamesARefusal(t *testing.T) {
	t.Parallel()

	fake := &spotifyFake{output: "", failure: nil, scripts: nil, fetched: nil}
	player := newSpotify(fake)

	commands := map[string]func(context.Context) error{
		"playpause":      player.PlayPause,
		"previous track": player.Previous,
		"next track":     player.Next,
		"activate":       player.Show,
	}

	for word, command := range commands {
		fake.scripts = nil

		commandError := command(t.Context())
		if commandError != nil || len(fake.scripts) != 1 || !strings.Contains(fake.scripts[0], word) {
			t.Errorf("%s: want one script naming it, got %v (%v)", word, fake.scripts, commandError)
		}
	}

	refused := newSpotify(&spotifyFake{output: "", failure: errNotAuthorised, scripts: nil, fetched: nil})

	_, readError := refused.NowPlaying(t.Context())
	nextError := refused.Next(t.Context())

	if !errors.Is(readError, platform.ErrMediaNotAllowed) || !errors.Is(nextError, platform.ErrMediaNotAllowed) {
		t.Errorf("want the Automation refusal named, got %v", readError)
	}
}
