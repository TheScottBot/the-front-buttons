package daemon

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// badgeInterval is how often the Dock's badges are read while a key shows
// one: soon enough that a new message shows within a few seconds, and cheap,
// as one read lists every badge at once.
const badgeInterval = 2 * time.Second

// badgesEvent carries one read of the Dock's badges back to the loop.
type badgesEvent struct {
	badges  map[string]string
	failure error
}

// badgeState is what the loop knows of badges: the applications whose badge a
// key shows, by lower case identifier; the badges last read; whether a read is
// under way; and the last failure logged.
type badgeState struct {
	wanted  map[string]bool
	shown   map[string]string
	reading bool
	failure string
}

func newBadgeState() badgeState {
	return badgeState{wanted: map[string]bool{}, shown: map[string]string{}, reading: false, failure: ""}
}

// wantedBadge is the lower case identifier of the application whose badge a
// key shows, if it shows one.
func wantedBadge(parameters any) (string, bool) {
	source, offers := parameters.(brick.ApplicationBadgeSource)
	if !offers {
		return "", false
	}

	identifier, wanted := source.ApplicationBadge()

	return strings.ToLower(identifier), wanted
}

// wantBadges notes which applications' badges a configuration shows. With
// none, the Dock is not read at all.
func (state *loopState) wantBadges(configuration config.Configuration) {
	state.badges.wanted = map[string]bool{}

	for _, page := range configuration.Pages {
		for _, key := range page.Keys {
			if identifier, wanted := wantedBadge(key.Parameters); wanted {
				state.badges.wanted[identifier] = true
			}
		}
	}
}

// readBadges starts a read of the Dock's badges, unless no key shows one or a
// read is already under way. The read runs apart from the loop, so a slow
// Dock never holds up a key press.
func (state *loopState) readBadges(runContext context.Context) {
	reader := state.options.Badges
	if reader == nil || len(state.badges.wanted) == 0 || state.badges.reading {
		return
	}

	state.badges.reading = true

	go func() {
		badges, failure := reader.Badges(runContext)
		state.send(runContext, badgesEvent{badges: badges, failure: failure})
	}()
}

// handleBadges takes a read's badges, logs each wanted badge that changed,
// and redraws the keys showing it. A failure is logged once until it changes,
// and clears the badges rather than leaving old ones showing.
func (state *loopState) handleBadges(runContext context.Context, read badgesEvent) {
	state.badges.reading = false

	if read.failure != nil {
		if read.failure.Error() != state.badges.failure {
			state.badges.failure = read.failure.Error()
			state.options.Logger.WarnContext(runContext, "badges not read", slog.String("error", read.failure.Error()))
		}

		read.badges = nil
	} else {
		state.badges.failure = ""
	}

	changed := map[string]bool{}

	for identifier := range state.badges.wanted {
		if read.badges[identifier] != state.badges.shown[identifier] {
			changed[identifier] = true
			state.options.Logger.InfoContext(runContext, "application badge",
				slog.String("application", identifier), slog.String("badge", read.badges[identifier]))
		}
	}

	state.badges.shown = read.badges
	state.repaintBadged(runContext, changed)
}

// repaintBadged redraws the showing page's keys whose badge changed.
func (state *loopState) repaintBadged(runContext context.Context, changed map[string]bool) {
	pageIndex, showing := state.showingPage()
	if len(changed) == 0 || !showing {
		return
	}

	for _, configured := range state.pages[pageIndex] {
		if identifier, wanted := wantedBadge(configured.key.Parameters); wanted && changed[identifier] {
			state.repaintKey(runContext, configured.key.Position, configured)
		}
	}
}

// badgeFor is the badge a key shows: its application's, if it asks for one.
func (state *loopState) badgeFor(configured configuredKey) string {
	identifier, wanted := wantedBadge(configured.key.Parameters)
	if !wanted {
		return ""
	}

	return state.badges.shown[identifier]
}
