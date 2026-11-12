// Command the-front-buttons is the composition root: the only place adapters are
// constructed and injected into the core (specification section 2). It holds no
// behaviour of its own beyond wiring, so everything it starts is tested where it
// is defined.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/allbricks"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/daemon"
	"github.com/TheScottBot/the-front-buttons/internal/deck"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/editor"
	"github.com/TheScottBot/the-front-buttons/internal/githubactions"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/render/tablericons"
)

// The daemon's timings and settings. Each is provisional until the author gates
// on the deck (TE4 for brightness, TE6 for the timings).
const (
	// deckPollInterval is how often attached decks are listed. The dock
	// switching machines takes seconds, so once a second shows the deck promptly
	// at negligible cost.
	deckPollInterval = time.Second
	// keyReadTimeout bounds each wait for a key report. The adapter holds its
	// lock while reading, so a short wait keeps key image writes from queueing.
	keyReadTimeout = 50 * time.Millisecond
	// configurationPollInterval is how often the configuration file is checked.
	configurationPollInterval = time.Second
	// deckBrightness is the backlight percentage.
	deckBrightness = 70
	// githubRequestTimeout bounds one GitHub poll, so a stalled connection
	// shows as a network error on the key instead of a key that never updates.
	githubRequestTimeout = 20 * time.Second
	// The editor server's timeouts: generous for a person on this machine,
	// bounded so a stalled connection cannot hold the server open.
	editorHeaderTimeout   = 10 * time.Second
	editorRequestTimeout  = 30 * time.Second
	editorIdleTimeout     = time.Minute
	editorShutdownTimeout = 5 * time.Second
	// The log file in the state folder (specification section 7): rotated at a
	// megabyte, keeping three older files, so it never fills the disk.
	logFileName               = "the-front-buttons.log"
	logFileBytes              = 1 << 20
	logFilesKept              = 3
	launchdLogName            = "launchd.log"
	stateDirectoryPermissions = 0o700
)

// developmentVersion is reported when the binary was not built from a tagged
// module version, as with go run.
const developmentVersion = "development"

// errNoSupportedModel reports a model table with nothing in it, a build mistake.
var errNoSupportedModel = errors.New("no supported deck model")

func main() {
	os.Exit(run())
}

func run() int {
	configPathOverride := flag.String("config", "",
		"configuration file to use instead of the per machine default")
	printLaunchAgent := flag.Bool("print-launch-agent", false,
		"print the macOS login agent property list for this binary, then exit")
	printScheduledTask := flag.Bool("print-scheduled-task", false,
		"print the Windows logon task definition for this binary, then exit")
	taskUser := flag.String("user", "", "the account the logon task runs as, with -print-scheduled-task")
	printName := flag.Bool("print-name", false, "print the product's display name, the logon task's name, then exit")

	flag.Parse()

	if *printName {
		printOut(platform.ProductDisplayName + "\n")

		return 0
	}

	if *printLaunchAgent || *printScheduledTask {
		return printStartup(*printLaunchAgent, *taskUser)
	}

	stateDirectory, logFile, logger := openLog()
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}

	runContext, stop := signal.NotifyContext(context.Background(), systemadapter.ShutdownSignals()...)
	defer stop()

	logger.InfoContext(runContext, "state", slog.String("directory", stateDirectory))

	startError := start(runContext, logger, *configPathOverride)
	if startError != nil {
		logger.ErrorContext(runContext, "stopped", slog.String("error", startError.Error()))

		return 1
	}

	return 0
}

// openLog sends the log to standard error and to the rotating file in the state
// folder. If the file cannot be opened the daemon still runs, logging to
// standard error, and says so, so the missing file is a fact in the log rather
// than a mystery (datum 0.15).
func openLog() (string, *diagnostics.RotatingFile, *slog.Logger) {
	stateDirectory, locationError := systemadapter.UserConfigLocation{}.StateDirectory()
	if locationError == nil {
		locationError = os.MkdirAll(stateDirectory, stateDirectoryPermissions)
	}

	var logFile *diagnostics.RotatingFile

	if locationError == nil {
		logFile, locationError = diagnostics.OpenRotatingFile(
			filepath.Join(stateDirectory, logFileName), logFileBytes, logFilesKept)
	}

	if locationError != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
		logger.Warn("log file unavailable; logging to standard error only", slog.String("error", locationError.Error()))

		return stateDirectory, nil, logger
	}

	return stateDirectory, logFile, slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stderr, logFile), nil))
}

// printStartup prints the start at login definition for this binary, which the
// install scripts write into place, so the definition is written once, in Go.
func printStartup(launchAgent bool, taskUser string) int {
	binaryPath, binaryError := executablePath()

	stateDirectory, locationError := systemadapter.UserConfigLocation{}.StateDirectory()

	joined := errors.Join(binaryError, locationError)
	if joined != nil {
		printError(joined.Error())

		return 1
	}

	if launchAgent {
		printOut(platform.LaunchAgentPlist(binaryPath, filepath.Join(stateDirectory, launchdLogName)))

		return 0
	}

	if taskUser == "" {
		printError("-print-scheduled-task needs -user")

		return 1
	}

	printOut(platform.ScheduledTaskXML(binaryPath, taskUser))

	return 0
}

// printOut writes what a script reads. A failed write leaves the script reading
// nothing, which it refuses, so the error has nowhere more useful to go.
func printOut(text string) {
	_, _ = io.WriteString(os.Stdout, text)
}

// printError writes a message for a person at a terminal; the exit status is
// what reports the failure.
func printError(message string) {
	_, _ = io.WriteString(os.Stderr, "error: "+message+"\n")
}

// executablePath is this binary's absolute path with links resolved, so the
// login definition keeps working if the link is moved.
func executablePath() (string, error) {
	binaryPath, executableError := os.Executable()
	if executableError != nil {
		return "", fmt.Errorf("finding this binary: %w", executableError)
	}

	resolved, resolveError := filepath.EvalSymlinks(binaryPath)
	if resolveError != nil {
		return "", fmt.Errorf("resolving this binary's path: %w", resolveError)
	}

	return resolved, nil
}

// core is everything built from the registry: the kinds, the presets and the
// grid configuration is checked against.
type core struct {
	model       deck.Model
	grid        layout.Grid
	registry    *brick.Registry
	presetTable presets.Table
	renderer    *render.Renderer
}

func buildCore() (core, error) {
	models := deck.Models()
	if len(models) == 0 {
		return core{}, errNoSupportedModel
	}

	// Configuration is validated against the XL's grid, the model this product
	// supports (specification section 1); the model table admits others later.
	model := models[0]

	grid, gridError := model.Grid()
	if gridError != nil {
		return core{}, fmt.Errorf("building the grid: %w", gridError)
	}

	registry := brick.NewRegistry()

	registerError := allbricks.Register(registry)
	if registerError != nil {
		return core{}, fmt.Errorf("registering brick kinds: %w", registerError)
	}

	presetTable, presetError := presets.BuiltIn(registry)
	if presetError != nil {
		return core{}, fmt.Errorf("building presets: %w", presetError)
	}

	icons, iconError := tablericons.Load()
	if iconError != nil {
		return core{}, fmt.Errorf("loading icons: %w", iconError)
	}

	renderer, rendererError := render.NewRenderer(model.Protocol.KeyPixels, icons)
	if rendererError != nil {
		return core{}, fmt.Errorf("building the renderer: %w", rendererError)
	}

	return core{model: model, grid: grid, registry: registry, presetTable: presetTable, renderer: renderer}, nil
}

func start(runContext context.Context, logger *slog.Logger, configPathOverride string) error {
	built, buildError := buildCore()
	if buildError != nil {
		return buildError
	}

	configPath, pathError := chooseConfigPath(configPathOverride)
	if pathError != nil {
		return pathError
	}

	logger.InfoContext(runContext, "starting",
		slog.String("product", platform.ProductDisplayName),
		slog.String("version", buildVersion()),
		slog.String("config", configPath),
		slog.Any("kinds", built.registry.KindNames()),
		slog.Any("presets", built.presetTable.Names()))

	decoder := config.NewDecoder(built.registry, built.presetTable, built.grid)
	store := config.NewStore(configPath, decoder, config.LocalFileSystem{})
	configurations := make(chan config.Configuration, 1)

	go store.Watch(runContext, configurationPollInterval, func(configuration config.Configuration) {
		offerLatest(configurations, configuration)
	}, logger)

	counters := diagnostics.NewCounters()
	credentials := systemadapter.NewKeyringCredentialStore()

	editorAddress, stopEditor, editorError := startEditor(runContext, logger, built, store, credentials, counters)
	if editorError != nil {
		return editorError
	}
	defer stopEditor()

	runner := daemon.New(daemonOptions(built, logger, counters, credentials,
		githubactions.NewClient(newGitHubHTTPClient(), credentials, counters),
		hidtransport.NewWatcher(), editorAddress))

	return runner.Run(runContext, configurations) //nolint:wrapcheck // the daemon's errors already say what failed
}

// startEditor opens the editor on a loopback port with a token minted for this
// launch, prints its address (TD9), and returns the address and a function that
// shuts it down.
func startEditor(
	runContext context.Context,
	logger *slog.Logger,
	built core,
	store *config.Store,
	credentials platform.CredentialStore,
	counters *diagnostics.Counters,
) (string, func(), error) {
	listener, listenError := editor.Listen(runContext)
	if listenError != nil {
		return "", nil, fmt.Errorf("starting the editor: %w", listenError)
	}

	token, tokenError := editor.NewToken()
	if tokenError != nil {
		return "", nil, errors.Join(tokenError, listener.Close())
	}

	address := listener.Addr().String()
	editorURL := "http://" + address + "/"

	server := new(http.Server)
	server.Handler = editor.NewHandler(editor.Options{
		Address: address, Token: token, Store: store, Registry: built.registry, Presets: built.presetTable,
		Grid: built.grid, Renderer: built.renderer, Credentials: credentials, Counters: counters, Logger: logger,
	})
	server.ReadHeaderTimeout = editorHeaderTimeout
	server.ReadTimeout = editorRequestTimeout
	server.WriteTimeout = editorRequestTimeout
	server.IdleTimeout = editorIdleTimeout

	go func() {
		serveError := server.Serve(listener)
		if serveError != nil && !errors.Is(serveError, http.ErrServerClosed) {
			logger.ErrorContext(runContext, "editor stopped", slog.String("error", serveError.Error()))
		}
	}()

	logger.InfoContext(runContext, "editor", slog.String("url", editorURL))
	fmt.Fprintf(os.Stdout, "Editor: %s\n", editorURL) //nolint:errcheck // a closed standard output only loses the hint

	stop := func() {
		shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(runContext), editorShutdownTimeout)
		defer cancel()

		// Shutting down; the daemon is already stopping, so a slow close is moot.
		_ = server.Shutdown(shutdownContext)
	}

	return editorURL, stop, nil
}

// newGitHubHTTPClient is the HTTP client for GitHub: the standard transport with
// a bound on each request.
func newGitHubHTTPClient() *http.Client {
	httpClient := new(http.Client)
	httpClient.Timeout = githubRequestTimeout

	return httpClient
}

// daemonOptions wires the adapters into the daemon. It is separate from start so
// the composition can be tested with fakes in place of the deck and GitHub.
func daemonOptions(
	built core,
	logger *slog.Logger,
	counters *diagnostics.Counters,
	credentials platform.CredentialStore,
	pipelineProvider pipeline.Provider,
	watcher deck.DeckWatcher,
	editorAddress string,
) daemon.Options {
	return daemon.Options{
		Watcher:  watcher,
		Renderer: built.renderer,
		Registry: built.registry,
		Ports: brick.Ports{
			URLOpener:           systemadapter.NewURLOpener(),
			ApplicationLauncher: systemadapter.NewApplicationLauncher(),
			PipelineProvider:    pipelineProvider,
			EditorAddress:       func() string { return editorAddress },
		},
		Logger:          logger,
		PollInterval:    deckPollInterval,
		ReadTimeout:     keyReadTimeout,
		Brightness:      deckBrightness,
		Counters:        counters,
		CredentialStore: credentials,
	}
}

func chooseConfigPath(configPathOverride string) (string, error) {
	if configPathOverride != "" {
		return configPathOverride, nil
	}

	configPath, locationError := systemadapter.UserConfigLocation{}.ConfigFile()
	if locationError != nil {
		return "", fmt.Errorf("locating the configuration file: %w", locationError)
	}

	return configPath, nil
}

// offerLatest replaces any configuration the daemon has not yet taken with the
// newer one, so the daemon always applies the latest file and a slow reload
// never blocks the watcher.
func offerLatest(configurations chan config.Configuration, configuration config.Configuration) {
	for {
		select {
		case configurations <- configuration:
			return
		default:
		}

		select {
		case <-configurations:
		default:
		}
	}
}

func buildVersion() string {
	buildInformation, available := debug.ReadBuildInfo()
	if !available || buildInformation.Main.Version == "" || buildInformation.Main.Version == "(devel)" {
		return developmentVersion
	}

	return buildInformation.Main.Version
}
