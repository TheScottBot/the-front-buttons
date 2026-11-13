package spotify_test

import (
	"context"
	"image"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/spotify"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// playerFake is a media player that records the commands sent to it and
// answers what is playing from a fixed state.
type playerFake struct {
	mutex    sync.Mutex
	playing  platform.NowPlaying
	failure  error
	commands []string
}

func (fake *playerFake) NowPlaying(context.Context) (platform.NowPlaying, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return fake.playing, fake.failure
}

func (fake *playerFake) PlayPause(context.Context) error { return fake.record("playpause") }
func (fake *playerFake) Previous(context.Context) error  { return fake.record("previous") }
func (fake *playerFake) Next(context.Context) error      { return fake.record("next") }
func (fake *playerFake) Show(context.Context) error      { return fake.record("show") }

func (fake *playerFake) record(command string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.commands = append(fake.commands, command)

	return fake.failure
}

// newPlayerFake is a player with nothing playing.
func newPlayerFake() *playerFake {
	return &playerFake{
		mutex: sync.Mutex{}, playing: platform.NowPlaying{Running: false, Playing: false, Artwork: nil},
		failure: nil, commands: nil,
	}
}

func construct(t *testing.T, kindName string, player platform.MediaPlayer) any {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := spotify.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	kind, found := registry.Kind(kindName)
	if !found {
		t.Fatalf("want the %s kind", kindName)
	}

	constructed, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
		MediaPlayer: player,
	})
	if constructError != nil {
		t.Fatal(constructError)
	}

	return constructed
}

// TestSpotify_EachKeyPressSendsItsCommand: the author's 2x2 is the album,
// play or pause, previous track and next track; pressing the album brings
// Spotify to the front.
func TestSpotify_EachKeyPressSendsItsCommand(t *testing.T) {
	t.Parallel()

	for kindName, want := range map[string]string{
		spotify.AlbumKindName:     "show",
		spotify.PlayPauseKindName: "playpause",
		spotify.PreviousKindName:  "previous",
		spotify.NextKindName:      "next",
	} {
		player := newPlayerFake()

		pressable, isPressable := construct(t, kindName, player).(brick.Pressable)
		if !isPressable {
			t.Fatalf("%s: want a pressable key", kindName)
		}

		pressError := pressable.Press(t.Context())
		if pressError != nil || len(player.commands) != 1 || player.commands[0] != want {
			t.Errorf("%s: want %s sent, got %v (%v)", kindName, want, player.commands, pressError)
		}
	}

	refused := &playerFake{
		mutex: sync.Mutex{}, playing: platform.NowPlaying{Running: false, Playing: false, Artwork: nil},
		failure: platform.ErrMediaNotAllowed, commands: nil,
	}
	pressable, _ := construct(t, spotify.NextKindName, refused).(brick.Pressable)

	if reason := brick.ReasonCategoryOf(pressable.Press(t.Context())); reason != "permission" {
		t.Errorf("want a refused press shown as permission, got %q", reason)
	}
}

// TestSpotify_AlbumAndPlayKeysDrawWhatIsPlaying: the album key fills with
// the art; the play key shows pause while playing and play otherwise.
func TestSpotify_AlbumAndPlayKeysDrawWhatIsPlaying(t *testing.T) {
	t.Parallel()

	art := image.NewRGBA(image.Rect(0, 0, 4, 4))
	playing := brick.LiveState{Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: art, Playing: true}
	paused := brick.LiveState{Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: art, Playing: false}

	album, _ := construct(t, spotify.AlbumKindName, newPlayerFake()).(brick.LiveDrawn)
	play, _ := construct(t, spotify.PlayPauseKindName, newPlayerFake()).(brick.LiveDrawn)

	if album == nil || play == nil {
		t.Fatal("want the album and play keys drawn from their state")
	}

	var appearance render.Appearance

	album.DrawLive(playing, &appearance)

	if appearance.Picture != art {
		t.Error("want the album key filled with the art")
	}

	for state, wantIcon := range map[bool]string{true: "player-pause", false: "player-play"} {
		appearance = render.Appearance{}

		if state {
			play.DrawLive(playing, &appearance)
		} else {
			play.DrawLive(paused, &appearance)
		}

		if appearance.Icon != wantIcon || appearance.Picture != nil {
			t.Errorf("playing %v: want the %s icon and no art, got %+v", state, wantIcon, appearance)
		}
	}
}

// TestSpotify_AlbumWatchesThePlayer: the album and play keys share one
// source, which publishes the art and whether it is playing, and a reason
// when the player cannot be read.
func TestSpotify_AlbumWatchesThePlayer(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		art := image.NewRGBA(image.Rect(0, 0, 4, 4))
		player := &playerFake{
			mutex:    sync.Mutex{},
			playing:  platform.NowPlaying{Running: true, Playing: true, Artwork: art},
			failure:  nil,
			commands: nil,
		}

		album, _ := construct(t, spotify.AlbumKindName, player).(brick.LiveSource)
		play, _ := construct(t, spotify.PlayPauseKindName, player).(brick.LiveSource)

		if album == nil || play == nil || album.LiveIdentity() != play.LiveIdentity() {
			t.Fatal("want the album and play keys watching one shared source")
		}

		var (
			mutex     sync.Mutex
			published []brick.LiveState
		)

		watchContext, cancel := context.WithCancel(t.Context())
		go album.Watch(watchContext, func(state brick.LiveState) {
			mutex.Lock()
			defer mutex.Unlock()

			published = append(published, state)
		})

		synctest.Wait()

		player.mutex.Lock()
		player.failure = platform.ErrMediaNotAllowed
		player.mutex.Unlock()

		time.Sleep(5 * time.Second)
		synctest.Wait()
		cancel()

		mutex.Lock()
		defer mutex.Unlock()

		if len(published) < 2 || published[0].Picture != art || !published[0].Playing {
			t.Fatalf("want the art and playing published first, got %+v", published)
		}

		if last := published[len(published)-1]; last.ErrorReason != "permission" || last.Picture != nil {
			t.Errorf("want a refused read shown as permission without art, got %+v", last)
		}
	})
}
