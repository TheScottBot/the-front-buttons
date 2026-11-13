// Package daemon is the main loop: deck connections, key presses, paging,
// configuration reloads and drawing, joined in one place so the composition root
// only wires adapters in (specification section 2).
//
// One goroutine owns all state. A reader goroutine per connected deck and one
// goroutine per key press report back through a channel, so no state is shared
// and no lock is needed around it.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/live"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// pressTimeout bounds one key press, so a brick that hangs cannot pile up
// goroutines; a press that opens a URL takes far less.
const pressTimeout = 30 * time.Second

// eventBuffer lets the reader and press goroutines hand over an event without
// waiting for the loop in the common case.
const eventBuffer = 64

// The connection states logged, each once per change (datum 0.15: say what is
// happening, once).
const (
	stateWaiting           = "waiting"
	stateConnected         = "connected"
	stateDisconnected      = "disconnected"
	stateBusy              = "busy"
	stateEnumerationFailed = "enumeration_failed"
	stateOpenFailed        = "open_failed"
)

// reasonInvalid marks a key whose brick could not be built from its parameters.
const reasonInvalid = "invalid"

// Options are the daemon's adapters and settings, all supplied by the
// composition root.
type Options struct {
	Watcher      deck.DeckWatcher
	Renderer     *render.Renderer
	Registry     *brick.Registry
	Ports        brick.Ports
	Logger       *slog.Logger
	PollInterval time.Duration
	ReadTimeout  time.Duration
	Brightness   int

	Counters        *diagnostics.Counters
	CredentialStore platform.CredentialStore
	// IconFiles reads keys' own icon files; nil reads none, and a key naming
	// one shows the missing icon marker.
	IconFiles IconFileLoader
	// AppIcons reads applications' own icons, for keys that ask to show one;
	// nil reads none, and such a key shows the missing icon marker.
	AppIcons platform.ApplicationIcons
	// Activator brings an app screen's application to the front before a key
	// on the screen sends it key presses; nil sends them to whatever is in
	// front.
	Activator platform.ApplicationActivator
}

// IconFileLoader reads a key's own icon file by its name.
type IconFileLoader interface {
	Load(fileName string) (image.Image, error)
}

// Daemon runs the deck.
type Daemon struct {
	options Options
}

// New returns a daemon with the given options.
func New(options Options) *Daemon {
	return &Daemon{options: options}
}

// Run drives the deck until the context ends, taking each new configuration from
// configurations. With no deck attached it keeps polling, so the deck comes alive
// the moment the dock brings it to this machine. On return the deck is reset and
// closed.
func (daemon *Daemon) Run(runContext context.Context, configurations <-chan config.Configuration) error {
	state := newLoopState(daemon.options)
	state.hub = live.NewHub(state.liveChanges.mark)

	ticker := time.NewTicker(daemon.options.PollInterval)
	defer ticker.Stop()

	state.poll(runContext)

	for {
		select {
		case <-runContext.Done():
			state.shutdown(runContext)

			return nil
		case configuration := <-configurations:
			state.applyConfiguration(runContext, configuration)
		case <-ticker.C:
			state.poll(runContext)
		case received := <-state.events:
			state.handle(runContext, received)
		case <-state.liveChanges.signal:
			state.handleLiveChanges(runContext, state.liveChanges.take())
		}
	}
}

// liveChangeSet collects the identities of live sources that published since the
// loop last looked. Marking never blocks: the hub calls it from source
// goroutines, and the loop may be waiting on the hub to stop a source, so a
// blocking hand over could deadlock the two.
type liveChangeSet struct {
	mutex      sync.Mutex
	identities map[string]bool
	signal     chan struct{}
}

func newLiveChangeSet() *liveChangeSet {
	return &liveChangeSet{mutex: sync.Mutex{}, identities: map[string]bool{}, signal: make(chan struct{}, 1)}
}

func (changes *liveChangeSet) mark(identity string) {
	changes.mutex.Lock()
	changes.identities[identity] = true
	changes.mutex.Unlock()

	select {
	case changes.signal <- struct{}{}:
	default:
	}
}

func (changes *liveChangeSet) take() map[string]bool {
	changes.mutex.Lock()
	defer changes.mutex.Unlock()

	taken := changes.identities
	changes.identities = map[string]bool{}

	return taken
}

// configuredKey is one brick key as configured and built.
type configuredKey struct {
	key         config.Key
	instance    any
	errorReason string
	// pageNumber is the regular page the key is on, counting from one, for a
	// page marker to show; 0 on an app screen.
	pageNumber int
	// pageName is the name of the page the key is on, which a page marker on an
	// app screen shows.
	pageName string
}

// The events the loop receives from its goroutines. Each carries the generation
// it belongs to, so an event from a deck or configuration that has since been
// replaced is ignored rather than acted on.
type keyChangesEvent struct {
	deckGeneration int
	changes        []deck.KeyChange
}

type readFailedEvent struct {
	deckGeneration int
	failure        error
}

type pressResultEvent struct {
	configurationGeneration int
	pageIndex               int
	brickIndex              int
	failure                 error
	// took is how long the press took, logged when it was slow.
	took time.Duration
}

type loopState struct {
	options     Options
	events      chan any
	hub         *live.Hub
	liveChanges *liveChangeSet
	liveReasons map[string]string
	// iconFiles holds each icon file loaded under the current configuration,
	// nil for one that failed, so a file is read once per configuration.
	iconFiles map[string]image.Image
	// appIcons holds each application icon read under the current
	// configuration, nil for one that could not be read, so an icon is read,
	// and a failure logged, once per configuration.
	appIcons map[string]image.Image

	pages     [][]configuredKey
	pageNames []string
	// regular lists the regular pages by their index in pages, in order:
	// paging moves through these and never through an app screen.
	regular []int
	// appScreens finds each application's screen by its index in pages.
	appScreens map[string]int
	// appScreen is the application whose screen is showing, or empty while a
	// regular page is.
	appScreen               string
	paging                  layout.Paging
	hasConfiguration        bool
	configurationGeneration int

	connected      *deck.Deck
	connectionPath string
	grid           layout.Grid
	slots          []layout.Slot
	deckGeneration int

	loggedState string

	controls deckControls
}

func newLoopState(options Options) *loopState {
	return &loopState{
		options:                 options,
		events:                  make(chan any, eventBuffer),
		hub:                     nil,
		liveChanges:             newLiveChangeSet(),
		liveReasons:             map[string]string{},
		iconFiles:               map[string]image.Image{},
		appIcons:                map[string]image.Image{},
		pages:                   nil,
		pageNames:               nil,
		regular:                 nil,
		appScreens:              map[string]int{},
		appScreen:               "",
		paging:                  layout.NewPaging(0),
		hasConfiguration:        false,
		configurationGeneration: 0,
		connected:               nil,
		connectionPath:          "",
		grid:                    layout.Grid{},
		slots:                   nil,
		deckGeneration:          0,
		loggedState:             "",
		controls: deckControls{
			dimmed: false, dimPercent: 0, off: false, offPercent: 0, offPage: 0, offBrick: 0,
		},
	}
}

// logState logs a connection state only when it differs from the last one
// logged, so a deck left busy for an hour is one line, not thousands.
func (state *loopState) logState(runContext context.Context, newState string, attributes ...any) {
	if newState == state.loggedState {
		return
	}

	state.loggedState = newState
	stateAttributes := append([]any{slog.String("state", newState)}, attributes...)
	state.options.Logger.InfoContext(runContext, "deck", stateAttributes...)
}

// send hands an event to the loop unless the daemon is stopping.
func (state *loopState) send(runContext context.Context, received any) {
	select {
	case state.events <- received:
	case <-runContext.Done():
	}
}

func (state *loopState) handle(runContext context.Context, received any) {
	switch typed := received.(type) {
	case keyChangesEvent:
		if typed.deckGeneration == state.deckGeneration {
			state.handleKeyChanges(runContext, typed.changes)
		}
	case readFailedEvent:
		if typed.deckGeneration == state.deckGeneration {
			state.disconnect(runContext, "read failed")
		}
	case pressResultEvent:
		if typed.configurationGeneration == state.configurationGeneration {
			state.handlePressResult(runContext, typed)
		}
	}
}

func (state *loopState) applyConfiguration(runContext context.Context, configuration config.Configuration) {
	state.configurationGeneration++
	state.iconFiles = map[string]image.Image{}
	state.appIcons = map[string]image.Image{}
	state.pages = make([][]configuredKey, 0, len(configuration.Pages))
	state.pageNames = nil
	state.regular = nil
	state.appScreens = map[string]int{}

	for pageIndex, page := range configuration.Pages {
		pageNumber := 0

		if page.IsAppScreen() {
			state.appScreens[page.Application] = pageIndex
		} else {
			state.regular = append(state.regular, pageIndex)
			state.pageNames = append(state.pageNames, page.Name)
			pageNumber = len(state.regular)
		}

		keys := make([]configuredKey, 0, len(page.Keys))
		for _, key := range page.Keys {
			built := state.build(runContext, pageIndex, key)
			built.pageNumber, built.pageName = pageNumber, page.Name
			keys = append(keys, built)
		}

		state.pages = append(state.pages, keys)
	}

	// A reload keeps the app screen showing while the application still has
	// one.
	if _, kept := state.appScreens[state.appScreen]; !kept {
		state.appScreen = ""
	}

	if state.hasConfiguration {
		state.paging = state.paging.Resize(len(state.regular))
	} else {
		state.paging = layout.NewPaging(len(state.regular))
		state.hasConfiguration = true
	}

	// The off key may have moved or gone, so a new configuration turns the
	// keys back on.
	if state.controls.off {
		state.controls.off = false
		state.applyBrightness(runContext)
	}

	state.hub.Update(runContext, state.liveSources())
	state.logMissingCredentials(runContext, configuration)
	state.repaintAll(runContext)
}

// liveSources lists every live brick on every page, so the hub runs them all
// whichever page is showing and whether or not a deck is connected.
func (state *loopState) liveSources() []brick.LiveSource {
	var sources []brick.LiveSource

	for _, keys := range state.pages {
		for _, configured := range keys {
			if source, isLive := configured.instance.(brick.LiveSource); isLive {
				sources = append(sources, source)
			}
		}
	}

	return sources
}

// logMissingCredentials names, once per configuration, each credential a key
// refers to that is not in the store (specification section 7), so a key stuck
// on credential_missing is explained in the log by name, never by secret.
func (state *loopState) logMissingCredentials(runContext context.Context, configuration config.Configuration) {
	logged := map[string]bool{}

	for _, page := range configuration.Pages {
		for _, key := range page.Keys {
			kind, kindFound := state.options.Registry.Kind(key.Kind)
			if !kindFound {
				continue
			}

			for _, field := range kind.Schema().Fields() {
				credentialName, isText := kind.Schema().Read(key.Parameters, field).(string)
				if !field.Secret || !isText || credentialName == "" || logged[credentialName] {
					continue
				}

				_, readError := state.options.CredentialStore.ReadCredential(credentialName)
				if errors.Is(readError, platform.ErrCredentialMissing) {
					logged[credentialName] = true
					state.options.Logger.WarnContext(runContext, "credential missing",
						slog.String("credential", credentialName), slog.String("kind", key.Kind))
				}
			}
		}
	}
}

// handleLiveChanges repaints the keys on the showing page whose source
// published, and logs each change of a source's error reason once. A press
// error on such a key is cleared first: the publish is newer news than the
// press, and a press made before any run was known otherwise left "no run" on
// a key whose run had since failed (evaluation, deck testing).
func (state *loopState) handleLiveChanges(runContext context.Context, identities map[string]bool) {
	for identity := range identities {
		liveState, _ := state.hub.State(identity)
		if liveState.ErrorReason != state.liveReasons[identity] {
			state.liveReasons[identity] = liveState.ErrorReason
			state.logLiveReason(runContext, identity, liveState.ErrorReason)
		}
	}

	state.clearPressErrors(identities)

	pageIndex, showing := state.showingPage()
	if !showing {
		return
	}

	for _, configured := range state.pages[pageIndex] {
		source, isLive := configured.instance.(brick.LiveSource)
		if isLive && identities[source.LiveIdentity()] {
			state.repaintKey(runContext, configured.key.Position, configured)
		}
	}
}

// clearPressErrors forgets the press errors of every key, on any page, whose
// source has just published.
func (state *loopState) clearPressErrors(identities map[string]bool) {
	for pageIndex := range state.pages {
		for keyIndex := range state.pages[pageIndex] {
			configured := &state.pages[pageIndex][keyIndex]

			if source, isLive := configured.instance.(brick.LiveSource); isLive && identities[source.LiveIdentity()] {
				configured.errorReason = ""
			}
		}
	}
}

func (state *loopState) logLiveReason(runContext context.Context, identity string, reason string) {
	kindName := ""

	for _, keys := range state.pages {
		for _, configured := range keys {
			if source, isLive := configured.instance.(brick.LiveSource); isLive && source.LiveIdentity() == identity {
				kindName = configured.key.Kind
			}
		}
	}

	if reason == "" {
		state.options.Logger.InfoContext(runContext, "live brick recovered", slog.String("kind", kindName))

		return
	}

	state.options.Counters.Increment("brick_errors." + kindName + "." + reason)
	state.options.Logger.WarnContext(runContext, "live brick error",
		slog.String("kind", kindName), slog.String("reason", reason))
}

// build constructs one key's brick. A brick that cannot be built shows its error
// on its key rather than taking the page down (specification section 2).
func (state *loopState) build(runContext context.Context, pageIndex int, key config.Key) configuredKey {
	built := configuredKey{key: key, instance: nil, errorReason: "", pageNumber: 0, pageName: ""}

	kind, kindFound := state.options.Registry.Kind(key.Kind)
	if !kindFound {
		built.errorReason = reasonInvalid

		return built
	}

	instance, constructError := kind.Construct(key.Parameters, state.options.Ports)
	if constructError != nil {
		built.errorReason = reasonInvalid
		state.options.Logger.WarnContext(runContext, "brick error",
			slog.String("kind", key.Kind), slog.Int("page", pageIndex),
			slog.Int("row", key.Position.Row), slog.Int("column", key.Position.Column),
			slog.String("reason", reasonInvalid))

		return built
	}

	built.instance = instance

	return built
}

func (state *loopState) poll(runContext context.Context) {
	connections, enumerateError := state.options.Watcher.Enumerate()

	if state.connected != nil {
		if enumerateError == nil && !containsPath(connections, state.connectionPath) {
			state.disconnect(runContext, "removed")
		}

		return
	}

	if enumerateError != nil {
		state.logState(runContext, stateEnumerationFailed)

		return
	}

	supported := deck.SupportedConnections(connections)
	if len(supported) == 0 {
		state.logState(runContext, stateWaiting)

		return
	}

	state.connect(runContext, supported[0])
}

func containsPath(connections []deck.Connection, path string) bool {
	for _, connection := range connections {
		if connection.Path == path {
			return true
		}
	}

	return false
}

func (state *loopState) connect(runContext context.Context, connection deck.Connection) {
	opened, openError := deck.OpenDeck(state.options.Watcher, connection)

	switch {
	case errors.Is(openError, deck.ErrDeckBusy):
		if state.loggedState != stateBusy {
			state.options.Counters.Increment("deck.busy")
		}

		state.logState(runContext, stateBusy,
			slog.String("hint", "another program, such as Elgato's own software, may hold the deck"))

		return
	case openError != nil:
		state.logState(runContext, stateOpenFailed)

		return
	}

	grid, gridError := opened.Model().Grid()
	brightnessError := opened.SetBrightness(state.brightness())

	if gridError != nil || brightnessError != nil {
		// Already abandoning a deck that failed to start; its close error adds nothing.
		_ = opened.Close()

		state.logState(runContext, stateOpenFailed)

		return
	}

	state.connected = opened
	state.connectionPath = connection.Path
	state.grid = grid
	state.deckGeneration++
	state.options.Counters.Increment("deck.connects")
	state.logState(runContext, stateConnected, slog.String("model", opened.Model().Name))

	go state.readKeys(runContext, opened, state.deckGeneration)

	state.repaintAll(runContext)
}

// readKeys runs for the life of one connected deck, handing key changes to the
// loop. It ends when a read fails, which closing the deck guarantees.
func (state *loopState) readKeys(runContext context.Context, opened *deck.Deck, deckGeneration int) {
	for {
		changes, readError := opened.ReadKeyChanges(state.options.ReadTimeout)
		if readError != nil {
			state.send(runContext, readFailedEvent{deckGeneration: deckGeneration, failure: readError})

			return
		}

		if len(changes) > 0 {
			state.send(runContext, keyChangesEvent{deckGeneration: deckGeneration, changes: changes})
		}
	}
}

// disconnect forgets the deck. The dock moving the deck away is normal, so this
// logs once and waits for the watcher to see a deck again.
func (state *loopState) disconnect(runContext context.Context, reason string) {
	if state.connected == nil {
		return
	}

	// The deck is already gone; closing only frees the handle.
	_ = state.connected.Close()

	state.connected = nil
	state.connectionPath = ""
	state.slots = nil
	state.deckGeneration++
	state.options.Counters.Increment("deck.disconnects")
	state.logState(runContext, stateDisconnected, slog.String("reason", reason))
}

func (state *loopState) shutdown(runContext context.Context) {
	state.hub.Stop()

	if state.connected == nil {
		return
	}

	// Leave the deck blank rather than showing keys that no longer work.
	// A deck that is already gone needs no reset.
	_ = state.connected.Reset()

	state.disconnect(runContext, "shutting down")
}

func (state *loopState) handleKeyChanges(runContext context.Context, changes []deck.KeyChange) {
	for _, change := range changes {
		if !change.Pressed {
			continue
		}

		// While off, a press only wakes the deck: what it would do cannot be
		// seen. The rest of the report is dropped with it.
		if state.controls.off {
			state.wake(runContext)

			return
		}

		slot, slotFound := layout.SlotAt(state.slots, change.KeyIndex)
		if !slotFound {
			continue
		}

		state.pressSlot(runContext, slot)
	}
}

// pressSlot does what a key does: back and forward page through the regular
// pages, leaving an app screen; the return tile leaves it for the regular page
// it was opened over; a brick is pressed.
func (state *loopState) pressSlot(runContext context.Context, slot layout.Slot) {
	switch slot.Kind {
	case layout.SlotBack:
		state.appScreen = ""
		state.paging = state.paging.Back()
		state.repaintAll(runContext)
	case layout.SlotForward:
		state.appScreen = ""
		state.paging = state.paging.Forward()
		state.repaintAll(runContext)
	case layout.SlotApplication:
		state.appScreen = ""
		state.repaintAll(runContext)
	case layout.SlotBrick:
		if pageIndex, showing := state.showingPage(); showing {
			state.press(runContext, pageIndex, slot.BrickIndex)
		}
	case layout.SlotEmpty:
	}
}

// showingPage is the index in pages of the page on the deck: the app screen
// while one is showing, otherwise the current regular page, if there is one.
func (state *loopState) showingPage() (int, bool) {
	if pageIndex, showing := state.appScreens[state.appScreen]; showing {
		return pageIndex, true
	}

	if !state.paging.HasPage() {
		return 0, false
	}

	return state.regular[state.paging.Current()], true
}

func (state *loopState) press(runContext context.Context, pageIndex int, brickIndex int) {
	configured := state.pages[pageIndex][brickIndex]

	state.options.Logger.InfoContext(runContext, "key press",
		slog.String("kind", configured.key.Kind), slog.Int("page", pageIndex),
		slog.Int("row", configured.key.Position.Row), slog.Int("column", configured.key.Position.Column))

	state.options.Counters.Increment("key_presses." + configured.key.Kind)
	state.openAppScreen(runContext, configured)

	if control, isControl := configured.instance.(brick.DeckControl); isControl {
		state.pressControl(runContext, pageIndex, brickIndex, control.DeckControl())

		return
	}

	pressFunction := state.bringingForward(pageIndex, configured, state.pressFunction(configured))
	if pressFunction == nil {
		return
	}

	configurationGeneration := state.configurationGeneration

	go func() {
		pressContext, cancel := context.WithTimeout(runContext, pressTimeout)
		defer cancel()

		started := time.Now()
		failure := pressFunction(pressContext)

		state.send(runContext, pressResultEvent{
			configurationGeneration: configurationGeneration,
			pageIndex:               pageIndex,
			brickIndex:              brickIndex,
			failure:                 failure,
			took:                    time.Since(started),
		})
	}()
}

// openAppScreen shows the application's screen when a key with its app screen
// ticked is pressed, before the key launches the application.
func (state *loopState) openAppScreen(runContext context.Context, configured configuredKey) {
	source, offers := configured.key.Parameters.(brick.ApplicationScreenSource)
	if !offers {
		return
	}

	application, wanted := source.ApplicationScreen()
	if _, hasScreen := state.appScreens[application]; !wanted || !hasScreen || application == state.appScreen {
		return
	}

	state.appScreen = application
	state.repaintAll(runContext)
}

// slowPress is how long a press takes before it is logged as slow, enough to
// be felt between pressing a key and seeing what it does.
const slowPress = 250 * time.Millisecond

// What a key on an app screen shows when its application could not be brought
// forward for its key presses: it is not open, or it would not come forward.
const (
	reasonNotOpen    = "not open"
	reasonNotInFront = "not in front"
)

// bringingForward has a key that sends key presses on an app screen bring the
// application to the front first, so the presses reach it; if it cannot, they
// are not sent to whatever is in front instead.
func (state *loopState) bringingForward(
	pageIndex int,
	configured configuredKey,
	press func(pressContext context.Context) error,
) func(pressContext context.Context) error {
	_, sendsKeystrokes := configured.instance.(brick.KeystrokeSender)
	application, onScreen := state.applicationOf(pageIndex)
	activator := state.options.Activator

	if press == nil || !sendsKeystrokes || !onScreen || activator == nil {
		return press
	}

	return func(pressContext context.Context) error {
		activateError := activator.ActivateApplication(pressContext, application)
		if activateError == nil {
			return press(pressContext)
		}

		reason := reasonNotInFront
		if errors.Is(activateError, platform.ErrApplicationNotRunning) {
			reason = reasonNotOpen
		}

		return brick.Categorised(reason, fmt.Errorf("bringing %s forward: %w", application, activateError))
	}
}

// applicationOf is the application whose screen a page is, if it is one.
func (state *loopState) applicationOf(pageIndex int) (string, bool) {
	for application, screenIndex := range state.appScreens {
		if screenIndex == pageIndex {
			return application, true
		}
	}

	return "", false
}

// pressFunction returns what a press of the key does, or nil if the brick does
// nothing when pressed. A live brick is pressed with its latest cached state.
func (state *loopState) pressFunction(configured configuredKey) func(pressContext context.Context) error {
	if pressable, isPressable := configured.instance.(brick.Pressable); isPressable {
		return pressable.Press
	}

	livePressable, isLivePressable := configured.instance.(brick.LivePressable)
	source, isLive := configured.instance.(brick.LiveSource)

	if !isLivePressable || !isLive {
		return nil
	}

	liveState, _ := state.hub.State(source.LiveIdentity())

	return func(pressContext context.Context) error {
		return livePressable.PressWithState(pressContext, liveState)
	}
}

func (state *loopState) handlePressResult(runContext context.Context, result pressResultEvent) {
	configured := &state.pages[result.pageIndex][result.brickIndex]
	previousReason := configured.errorReason

	if result.took >= slowPress {
		state.options.Logger.InfoContext(runContext, "slow key press",
			slog.String("kind", configured.key.Kind), slog.Duration("took", result.took))
	}

	configured.errorReason = ""
	if result.failure != nil {
		configured.errorReason = brick.ReasonCategoryOf(result.failure)
		state.options.Counters.Increment("brick_errors." + configured.key.Kind + "." + configured.errorReason)
		state.options.Logger.WarnContext(runContext, "brick error",
			slog.String("kind", configured.key.Kind), slog.Int("page", result.pageIndex),
			slog.Int("row", configured.key.Position.Row), slog.Int("column", configured.key.Position.Column),
			slog.String("reason", configured.errorReason))
	}

	showingIndex, showing := state.showingPage()
	if configured.errorReason != previousReason && showing && result.pageIndex == showingIndex {
		state.repaintKey(runContext, configured.key.Position, *configured)
	}
}
