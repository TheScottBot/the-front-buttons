package daemon_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/deckcontrol"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/pagemarker"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/daemon"
	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The test timings. Polling and reads run on synctest's fake clock, so these
// cost no real time.
const (
	testPollInterval = time.Second
	testReadTimeout  = 100 * time.Millisecond
	settle           = 3 * time.Second
)

// The names of the two regular pages most tests page between.
const (
	firstPageName  = "First"
	secondPageName = "Second"
)

// The XL's navigation key numbers (TE3).
const (
	backKey    = 24
	forwardKey = 31
)

// lockedBuffer is a log destination safe to read while the daemon writes.
type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (locked *lockedBuffer) Write(content []byte) (int, error) {
	locked.mutex.Lock()
	defer locked.mutex.Unlock()

	written, writeError := locked.buffer.Write(content)
	if writeError != nil {
		return written, fmt.Errorf("buffering a log line: %w", writeError)
	}

	return written, nil
}

func (locked *lockedBuffer) String() string {
	locked.mutex.Lock()
	defer locked.mutex.Unlock()

	return locked.buffer.String()
}

// harness runs a daemon against fake decks and fake bricks.
type harness struct {
	watcher        *testsupport.DeckWatcherTestDouble
	recorder       *testsupport.PressRecorder
	launches       *testsupport.LaunchRecorder
	live           *testsupport.LiveController
	renderer       *render.Renderer
	counters       *diagnostics.Counters
	logs           *lockedBuffer
	configurations chan config.Configuration
	cancel         context.CancelFunc
	done           chan error
}

func xlModel(t *testing.T) deck.Model {
	t.Helper()

	model, found := deck.LookupModel(testsupport.FakeDeckVendorID, testsupport.FakeDeckProductID)
	if !found {
		t.Fatal("want the XL model")
	}

	return model
}

// harnessRegistry holds the fake kinds, and open_application for its
// application icon.
func harnessRegistry(
	t *testing.T,
	recorder *testsupport.PressRecorder,
	liveController *testsupport.LiveController,
) *brick.Registry {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := errors.Join(
		testsupport.RegisterButtonKind(registry, recorder),
		testsupport.RegisterLiveKind(registry, liveController),
		openapplication.Register(registry),
		deckcontrol.Register(registry),
		pagemarker.Register(registry),
	)
	if registerError != nil {
		t.Fatal(registerError)
	}

	return registry
}

func startHarness(t *testing.T) *harness {
	t.Helper()

	recorder := testsupport.NewPressRecorder()
	liveController := testsupport.NewLiveController()
	registry := harnessRegistry(t, recorder, liveController)

	renderer, rendererError := render.NewRenderer(xlModel(t).Protocol.KeyPixels, iconSetTestDouble{})
	if rendererError != nil {
		t.Fatal(rendererError)
	}

	logs := &lockedBuffer{mutex: sync.Mutex{}, buffer: bytes.Buffer{}}
	runContext, cancel := context.WithCancel(t.Context())

	running := &harness{
		watcher:        testsupport.NewDeckWatcherTestDouble(),
		recorder:       recorder,
		launches:       testsupport.NewLaunchRecorder(),
		live:           liveController,
		renderer:       renderer,
		counters:       diagnostics.NewCounters(),
		logs:           logs,
		configurations: make(chan config.Configuration, 1),
		cancel:         cancel,
		done:           make(chan error, 1),
	}

	runner := daemon.New(daemon.Options{
		Watcher:  running.watcher,
		Renderer: renderer,
		Registry: registry,
		Ports: brick.Ports{
			URLOpener:           nil,
			ApplicationLauncher: testsupport.ApplicationLauncherTestDouble{Recorder: running.launches},
			PipelineProvider:    nil,
		},
		Logger:          slog.New(slog.NewJSONHandler(logs, nil)),
		Counters:        running.counters,
		CredentialStore: testsupport.NewCredentialStoreTestDouble(),
		PollInterval:    testPollInterval,
		ReadTimeout:     testReadTimeout,
		Brightness:      70,
		IconFiles:       iconFilesTestDouble{},
		AppIcons:        applicationIconsTestDouble{},
		Activator: testsupport.ActivatorTestDouble{
			Recorder: recorder, Running: []string{knownApplication, slowApplication}, Slow: []string{slowApplication},
		},
	})

	go func() { running.done <- runner.Run(runContext, running.configurations) }()

	return running
}

func (running *harness) stop(t *testing.T) {
	t.Helper()

	running.cancel()

	runError := <-running.done
	if runError != nil {
		t.Fatalf("want a clean shutdown, got %v", runError)
	}
}

// iconSetTestDouble has no icons; these tests are about the loop, not drawing.
type iconSetTestDouble struct{}

func (iconSetTestDouble) IconMask(string, int) (*image.Alpha, bool) { return nil, false }

func button(name string, column int, fails bool) config.Key {
	return config.Key{
		Position: layout.Position{Row: 0, Column: column},
		Kind:     testsupport.ButtonKindName,
		Parameters: &testsupport.ButtonParameters{
			Label: name, Icon: "", Background: "", IconFile: "", Name: name, Fails: fails,
		},
		Preset: nil,
	}
}

func twoPages() config.Configuration {
	return config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion,
		Pages: []config.Page{
			{Name: firstPageName, Application: "", Keys: []config.Key{button("first", 0, false)}},
			{Name: secondPageName, Application: "", Keys: []config.Key{button("second", 0, false), button("broken", 1, true)}},
		},
	}
}

// tap presses and releases a key, then lets the daemon catch up.
func tap(device *testsupport.DeckDeviceTestDouble, keyIndex int) {
	device.PressKeys(keyIndex)
	time.Sleep(testReadTimeout)
	synctest.Wait()
	device.PressKeys()
	time.Sleep(settle)
	synctest.Wait()
}

func TestDaemon_RepaintsEveryKey_AndSetsBrightness_WhenADeckConnects(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)

		time.Sleep(settle)
		synctest.Wait()

		drawn := device.DrawnKeys()
		if len(drawn) != 32 {
			t.Fatalf("want all 32 keys drawn, got %d: %v", len(drawn), drawn)
		}

		if features := device.Features(); len(features) == 0 || features[0][2] != 70 {
			t.Fatalf("want brightness 70 set, got %v", features)
		}

		running.stop(t)
	})
}

func TestDaemon_PagesForwardAndBack_WithWrapAround(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, 0)          // page 1: first
		tap(device, forwardKey) // to page 2
		tap(device, 0)          // second
		tap(device, forwardKey) // wraps to page 1
		tap(device, 0)          // first
		tap(device, backKey)    // wraps back to page 2
		tap(device, 0)          // second

		want := []string{"first", "second", "first", "second"}
		if got := running.recorder.Presses(); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("want presses %v, got %v", want, got)
		}

		running.stop(t)
	})
}

func TestDaemon_SurvivesDisconnectionMidRepaint_AndRepaintsTheReturningDeck(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		vanishing := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		vanishing.DisconnectAfterWrites(6)
		running.watcher.Attach("deck-a", vanishing)
		time.Sleep(settle)
		synctest.Wait()

		if !vanishing.Closed() {
			t.Fatal("want the vanished deck closed")
		}

		running.watcher.Detach("deck-a")

		returning := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-b", returning)
		time.Sleep(settle)
		synctest.Wait()

		if drawn := returning.DrawnKeys(); len(drawn) != 32 {
			t.Fatalf("want a full repaint on the returning deck, got %d keys", len(drawn))
		}

		running.stop(t)
	})
}

func TestDaemon_SurvivesTenDockSwitches_WithAFullRepaintEachTime(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		for switchIndex := range 10 {
			path := "deck-" + string(rune('a'+switchIndex))
			device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
			running.watcher.Attach(path, device)
			time.Sleep(settle)
			synctest.Wait()

			if drawn := device.DrawnKeys(); len(drawn) != 32 {
				t.Fatalf("switch %d: want a full repaint, got %d keys", switchIndex, len(drawn))
			}

			running.watcher.Detach(path)
			time.Sleep(settle)
			synctest.Wait()
		}

		running.stop(t)
	})
}

func TestDaemon_LogsBusyOnce_AndConnectsWhenTheDeckIsFreed(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))

		running.watcher.SetBusy("deck-a", true)
		running.watcher.Attach("deck-a", device)
		time.Sleep(5 * settle)
		synctest.Wait()

		if busyLogs := strings.Count(running.logs.String(), `"state":"busy"`); busyLogs != 1 {
			t.Fatalf("want the busy state logged exactly once, got %d:\n%s", busyLogs, running.logs.String())
		}

		running.watcher.SetBusy("deck-a", false)
		time.Sleep(settle)
		synctest.Wait()

		if drawn := device.DrawnKeys(); len(drawn) != 32 {
			t.Fatalf("want a full repaint once freed, got %d keys", len(drawn))
		}

		running.stop(t)
	})
}

func TestDaemon_RepaintsWhenTheConfigurationReloads(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		// The deck may connect before or after the first configuration arrives,
		// so the reload is measured as one more repaint of every key.
		drawnBefore := device.DrawnKeys()

		running.configurations <- twoPages()

		time.Sleep(settle)
		synctest.Wait()

		drawnAfter := device.DrawnKeys()
		for keyIndex := range 32 {
			if drawnAfter[keyIndex] != drawnBefore[keyIndex]+1 {
				t.Fatalf("key %d: want one more draw after a reload, drawn %d then %d",
					keyIndex, drawnBefore[keyIndex], drawnAfter[keyIndex])
			}
		}

		running.stop(t)
	})
}

func TestDaemon_RedrawsAKeyInError_WhenItsPressFails(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		tap(device, forwardKey)
		drawnBefore := device.DrawnKeys()[1]

		tap(device, 1)

		if drawnAfter := device.DrawnKeys()[1]; drawnAfter != drawnBefore+1 {
			t.Fatalf("want key 1 redrawn once in its error state, drawn %d then %d", drawnBefore, drawnAfter)
		}

		if !strings.Contains(running.logs.String(), `"reason":"failed"`) {
			t.Fatalf("want the press failure logged by reason, got:\n%s", running.logs.String())
		}

		running.stop(t)
	})
}

func TestDaemon_ResetsAndClosesTheDeck_OnShutdown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)
		running.configurations <- twoPages()

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		running.stop(t)

		features := device.Features()
		if lastFeature := features[len(features)-1]; lastFeature[1] != 0x02 || !device.Closed() {
			t.Fatalf("want a reset then close on shutdown, last feature % x, closed %v", lastFeature[:2], device.Closed())
		}
	})
}

func TestDaemon_ShowsTheDeckWithNoConfiguration_AsNavigationOnly(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running := startHarness(t)

		device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
		running.watcher.Attach("deck-a", device)
		time.Sleep(settle)
		synctest.Wait()

		if drawn := device.DrawnKeys(); len(drawn) != 32 {
			t.Fatalf("want every key drawn even before a configuration arrives, got %d", len(drawn))
		}

		running.stop(t)
	})
}
