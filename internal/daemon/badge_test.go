package daemon_test

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// badgedApplication is written in mixed case, as bundle identifiers often are,
// where the Dock reader gives them in lower case.
const badgedApplication = "com.example.Badged"

// badgeKey is an open_application key with Show notifications ticked or not.
func badgeKey(showNotifications bool) config.Key {
	return config.Key{
		Position: layout.Position{Row: 0, Column: 0},
		Kind:     openapplication.KindName,
		Parameters: &openapplication.Parameters{
			Appearance: openAppAppearance(""), Application: badgedApplication, UseAppIcon: false, AppScreen: false,
			ShowNotifications: showNotifications,
		},
		Preset: nil,
	}
}

func startBadgeHarness(t *testing.T, showNotifications bool) (*harness, *testsupport.DeckDeviceTestDouble) {
	t.Helper()

	running := startHarness(t)
	running.configurations <- config.Configuration{
		SchemaVersion: config.CurrentSchemaVersion,
		Pages:         []config.Page{{Name: firstPageName, Application: "", Keys: []config.Key{badgeKey(showNotifications)}}},
	}

	device := testsupport.NewDeckDeviceTestDouble(xlModel(t))
	running.watcher.Attach("deck-a", device)
	time.Sleep(settle)
	synctest.Wait()

	return running, device
}

// drawnWithBadge reports whether the key was drawn nearer with the badge than
// without it.
func drawnWithBadge(t *testing.T, running *harness, device *testsupport.DeckDeviceTestDouble, badge string) bool {
	t.Helper()

	plain := render.Appearance{Label: appIconTestLabel}
	badged := plain
	badged.Badge = badge
	drawn := drawnImage(t, device, 0)

	return distance(drawn, running.renderer.Render(badged)) < distance(drawn, running.renderer.Render(plain))
}

// TestDaemon_DrawsAnApplicationsBadge_AndFollowsIt: a key with Show
// notifications ticked shows its application's Dock badge within a few
// seconds, matched whatever the identifier's case, and loses it when the
// badge goes.
func TestDaemon_DrawsAnApplicationsBadge_AndFollowsIt(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, device := startBadgeHarness(t, true)

		running.badges.Show(map[string]string{strings.ToLower(badgedApplication): "3"})
		time.Sleep(settle)
		synctest.Wait()

		if !drawnWithBadge(t, running, device, "3") {
			t.Fatal("want the key drawn with its badge")
		}

		running.badges.Show(map[string]string{})
		time.Sleep(settle)
		synctest.Wait()

		if drawnWithBadge(t, running, device, "3") {
			t.Fatal("want the badge gone once the application clears it")
		}

		running.stop(t)
	})
}

// TestDaemon_ReadsNoBadges_UnlessAKeyAsks: the Dock is never read while no
// key shows a badge.
func TestDaemon_ReadsNoBadges_UnlessAKeyAsks(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, _ := startBadgeHarness(t, false)

		running.badges.Show(map[string]string{strings.ToLower(badgedApplication): "3"})
		time.Sleep(3 * settle)
		synctest.Wait()

		if reads := running.badges.Reads(); reads != 0 {
			t.Fatalf("want no reads, got %d", reads)
		}

		running.stop(t)
	})
}

// TestDaemon_LogsABadgeFailure_Once: a reader that keeps failing is logged
// once, not every few seconds.
func TestDaemon_LogsABadgeFailure_Once(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		running, _ := startBadgeHarness(t, true)

		running.badges.Fail(platform.ErrBadgesNotPermitted)
		time.Sleep(3 * settle)
		synctest.Wait()

		if logged := strings.Count(running.logs.String(), "badges not read"); logged != 1 {
			t.Fatalf("want the failure logged once, got %d:\n%s", logged, running.logs.String())
		}

		running.stop(t)
	})
}
