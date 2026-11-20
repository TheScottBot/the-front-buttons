package main

import (
	"context"
	"log/slog"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// unusedPlatformBridge is logged as the platform bridge on a system whose
// adapters take the bridge and ignore it.
const unusedPlatformBridge = "unused on this system"

// bridgeChoice is the platform_bridge setting read at start, and what it
// asks the macOS adapters and the HID watcher for.
type bridgeChoice struct {
	asked    config.PlatformBridge
	platform systemadapter.Bridge
	driver   hidtransport.DriverName
}

// bridgesFor maps a configuration's platform bridge, cgo when it names none,
// to the adapters' bridge and the HID driver.
func bridgesFor(configuration config.Configuration) bridgeChoice {
	asked := configuration.PlatformBridgeOrDefault()
	if asked == config.PlatformBridgePurego {
		return bridgeChoice{asked: asked, platform: systemadapter.BridgePurego, driver: hidtransport.DriverPurego}
	}

	return bridgeChoice{
		asked: config.PlatformBridgeCgo, platform: systemadapter.BridgeCgo, driver: hidtransport.DriverHIDAPI,
	}
}

// defaultBridges is the choice for a file that names no bridge or cannot be
// read: cgo.
func defaultBridges() bridgeChoice {
	return bridgesFor(config.Configuration{
		SchemaVersion: 0, WorldClocks: nil, PlatformBridge: "", NavigationBackground: "", Pages: nil,
	})
}

// readBridges reads the configuration file as it is now, without consuming
// the change the store's watcher reloads, and gives its bridges. A missing
// file is a first run and gives cgo; a file that cannot be read or is refused
// gives cgo too, and false, so the daemon still starts as it would without
// the setting.
func readBridges(store *config.Store) (bridgeChoice, bool) {
	configuration, readError := store.OnDisk()
	if readError != nil {
		return defaultBridges(), false
	}

	return bridgesFor(configuration), true
}

// startBridges reads the bridges once at start, makes the HID watcher with
// the driver chosen, and logs which bridge was asked for and which is in use.
func startBridges(
	runContext context.Context,
	logger *slog.Logger,
	store *config.Store,
) (bridgeChoice, *hidtransport.Watcher) {
	choice, read := readBridges(store)
	if !read {
		// The reason is left out: a refusal can quote the file.
		logger.WarnContext(runContext, "platform bridge not read from the configuration file; using cgo")
	}

	watcher := hidtransport.NewWatcher(choice.driver)
	logBridges(runContext, logger, choice, watcher.DriverChoice(), systemadapter.CgoAvailable,
		systemadapter.ShortcutPlatform())

	return choice, watcher
}

// logBridges logs one line: the bridge asked for, the platform bridge and HID
// driver in use, and any fall back, at warning level when there is one. The
// macOS constructors give purego when cgo is asked for in a build without it;
// other systems' adapters ignore the bridge. system is the name
// systemadapter.ShortcutPlatform gives, "mac" on macOS.
func logBridges(
	runContext context.Context,
	logger *slog.Logger,
	choice bridgeChoice,
	driver hidtransport.DriverChoice,
	cgoAvailable bool,
	system string,
) {
	level := slog.LevelInfo
	attributes := []slog.Attr{slog.String("asked", string(choice.asked))}

	switch {
	case system != "mac":
		attributes = append(attributes, slog.String("platform", unusedPlatformBridge))
	case choice.platform == systemadapter.BridgePurego || cgoAvailable:
		attributes = append(attributes, slog.String("platform", string(choice.platform)))
	default:
		level = slog.LevelWarn

		attributes = append(attributes, slog.String("platform", string(systemadapter.BridgePurego)),
			slog.String("platform_fell_back", "cgo is not in this build; using purego"))
	}

	switch {
	case driver.Unavailable != nil:
		level = slog.LevelWarn

		attributes = append(attributes, slog.String("hid", "none"),
			slog.String("hid_unavailable", driver.Unavailable.Error()))
	case driver.FellBack():
		level = slog.LevelWarn

		attributes = append(attributes, slog.String("hid", string(driver.Using)),
			slog.String("hid_fell_back",
				"the "+string(driver.Asked)+" driver is not in this build; using "+string(driver.Using)))
	default:
		attributes = append(attributes, slog.String("hid", string(driver.Using)))
	}

	logger.LogAttrs(runContext, level, "platform bridge", attributes...)
}

// acceptConfigurations is the store watcher's callback: it offers each
// accepted configuration to the daemon, and logs once, when a file's
// platform bridge first differs from the one the daemon started with, that
// the change takes effect at the next start. A file returning to the running
// bridge says nothing, and a later change away from it is logged again.
// The store's watcher calls it from one goroutine.
func acceptConfigurations(
	runContext context.Context,
	configurations chan config.Configuration,
	logger *slog.Logger,
	started config.PlatformBridge,
) func(config.Configuration) {
	lastSeen := started

	return func(configuration config.Configuration) {
		bridge := configuration.PlatformBridgeOrDefault()
		if bridge != lastSeen {
			lastSeen = bridge

			if bridge != started {
				logger.InfoContext(runContext, "platform bridge change takes effect at the next start",
					slog.String("running", string(started)), slog.String("file", string(bridge)))
			}
		}

		offerLatest(configurations, configuration)
	}
}

// oneShotBridge is the platform bridge for a flag that answers one question
// and exits: the configuration file's, or cgo when it cannot be read.
func oneShotBridge(configPathOverride string) systemadapter.Bridge {
	configPath, pathError := chooseConfigPath(configPathOverride)
	if pathError != nil {
		return defaultBridges().platform
	}

	parts, partsError := buildCatalogue()
	if partsError != nil {
		return defaultBridges().platform
	}

	choice, _ := readBridges(config.NewStore(configPath, parts.decoder(), config.LocalFileSystem{}))

	return choice.platform
}
