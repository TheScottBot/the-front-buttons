package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

const (
	startLogMessage  = "platform bridge"
	unreadLogMessage = "platform bridge not read from the configuration file; using cgo"
	changeLogMessage = "platform bridge change takes effect at the next start"
)

// jsonLog is a logger writing JSON lines into a buffer, and the buffer.
func jsonLog() (*slog.Logger, *bytes.Buffer) {
	var buffer bytes.Buffer

	return slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		AddSource: false, Level: slog.LevelDebug, ReplaceAttr: nil,
	})), &buffer
}

// linesWithMessage decodes the log's lines whose message is message.
func linesWithMessage(t *testing.T, buffer *bytes.Buffer, message string) []map[string]any {
	t.Helper()

	var found []map[string]any

	for line := range strings.SplitSeq(strings.TrimSpace(buffer.String()), "\n") {
		if line == "" {
			continue
		}

		var fields map[string]any

		decodeError := json.Unmarshal([]byte(line), &fields)
		if decodeError != nil {
			t.Fatalf("log line %q: %v", line, decodeError)
		}

		if fields["msg"] == message {
			found = append(found, fields)
		}
	}

	return found
}

// testStore is a configuration store for a file in a fresh folder holding
// content, or no file at all when content is empty.
func testStore(t *testing.T, content string) *config.Store {
	t.Helper()

	parts, partsError := buildCatalogue()
	if partsError != nil {
		t.Fatal(partsError)
	}

	path := filepath.Join(t.TempDir(), "config.yaml")

	if content != "" {
		writeError := os.WriteFile(path, []byte(content), 0o600)
		if writeError != nil {
			t.Fatal(writeError)
		}
	}

	return config.NewStore(path, parts.decoder(), config.LocalFileSystem{})
}

const (
	pureFile = "schema_version: 1\nplatform_bridge: purego\npages: []\n"
	cgoFile  = "schema_version: 1\nplatform_bridge: cgo\npages: []\n"
	bareFile = "schema_version: 1\npages: []\n"
	// refusedFile names a bridge the decoder refuses.
	refusedFile = "schema_version: 1\nplatform_bridge: neither\npages: []\n"
)

func TestBridgesFor_FollowsTheSetting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		setting  config.PlatformBridge
		platform systemadapter.Bridge
		driver   hidtransport.DriverName
	}{
		{"absent", "", systemadapter.BridgeCgo, hidtransport.DriverHIDAPI},
		{cgoName, config.PlatformBridgeCgo, systemadapter.BridgeCgo, hidtransport.DriverHIDAPI},
		{pureName, config.PlatformBridgePurego, systemadapter.BridgePurego, hidtransport.DriverPurego},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			choice := bridgesFor(config.Configuration{
				SchemaVersion: 1, WorldClocks: nil, PlatformBridge: testCase.setting, Pages: nil,
			})
			if choice.platform != testCase.platform || choice.driver != testCase.driver {
				t.Fatalf("want %s and %s, got %s and %s",
					testCase.platform, testCase.driver, choice.platform, choice.driver)
			}
		})
	}
}

func TestReadBridges_ReadsTheFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		content  string
		platform systemadapter.Bridge
		driver   hidtransport.DriverName
		read     bool
	}{
		{pureName, pureFile, systemadapter.BridgePurego, hidtransport.DriverPurego, true},
		{cgoName, cgoFile, systemadapter.BridgeCgo, hidtransport.DriverHIDAPI, true},
		{"absent", bareFile, systemadapter.BridgeCgo, hidtransport.DriverHIDAPI, true},
		{"missing file", "", systemadapter.BridgeCgo, hidtransport.DriverHIDAPI, true},
		{"refused file", refusedFile, systemadapter.BridgeCgo, hidtransport.DriverHIDAPI, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			choice, read := readBridges(testStore(t, testCase.content))
			if choice.platform != testCase.platform || choice.driver != testCase.driver || read != testCase.read {
				t.Fatalf("want %s, %s, read %t; got %s, %s, read %t", testCase.platform, testCase.driver,
					testCase.read, choice.platform, choice.driver, read)
			}
		})
	}
}

func TestReadBridges_UnreadableFileGivesCgo(t *testing.T) {
	t.Parallel()

	parts, partsError := buildCatalogue()
	if partsError != nil {
		t.Fatal(partsError)
	}

	// A folder where the file should be cannot be read as one.
	store := config.NewStore(t.TempDir(), parts.decoder(), config.LocalFileSystem{})

	choice, read := readBridges(store)
	if choice.platform != systemadapter.BridgeCgo || choice.driver != hidtransport.DriverHIDAPI || read {
		t.Fatalf("want cgo, not read; got %s, %s, read %t", choice.platform, choice.driver, read)
	}
}

func TestStartBridges_PassesTheSettingToTheWatcherAndLogsOnce(t *testing.T) {
	t.Parallel()

	logger, buffer := jsonLog()

	choice, watcher := startBridges(t.Context(), logger, testStore(t, pureFile))

	if choice.platform != systemadapter.BridgePurego {
		t.Fatalf("want the purego platform bridge, got %s", choice.platform)
	}

	if asked := watcher.DriverChoice().Asked; asked != hidtransport.DriverPurego {
		t.Fatalf("want the watcher asked for the purego driver, got %s", asked)
	}

	lines := linesWithMessage(t, buffer, startLogMessage)
	if len(lines) != 1 {
		t.Fatalf("want one start line, got %d in:\n%s", len(lines), buffer.String())
	}

	if lines[0]["asked"] != pureName {
		t.Fatalf("want purego asked for, got %v", lines[0])
	}

	if unread := linesWithMessage(t, buffer, unreadLogMessage); len(unread) != 0 {
		t.Fatalf("want no unread line for a readable file, got %v", unread)
	}
}

func TestStartBridges_RefusedFileLogsCgoAndWhy(t *testing.T) {
	t.Parallel()

	logger, buffer := jsonLog()

	choice, watcher := startBridges(t.Context(), logger,
		testStore(t, "schema_version: 1\nplatform_bridge: 7\npages: []\n"))

	if choice.platform != systemadapter.BridgeCgo || watcher.DriverChoice().Asked != hidtransport.DriverHIDAPI {
		t.Fatalf("want cgo for a refused file, got %s and %s", choice.platform, watcher.DriverChoice().Asked)
	}

	if unread := linesWithMessage(t, buffer, unreadLogMessage); len(unread) != 1 {
		t.Fatalf("want one unread line, got %d in:\n%s", len(unread), buffer.String())
	}

	if lines := linesWithMessage(t, buffer, startLogMessage); len(lines) != 1 || lines[0]["asked"] != cgoName {
		t.Fatalf("want one start line asking for cgo, got %v", lines)
	}
}

// The names the start line and the tests use.
const (
	cgoName    = "cgo"
	pureName   = "purego"
	macSystem  = "mac"
	levelInfo  = "INFO"
	levelWarn  = "WARN"
	noDriverAt = "no driver"
)

// errNoDriverBuilt stands in for hidtransport's error when no driver is built.
var errNoDriverBuilt = errors.New(noDriverAt)

// startLine is the start line's fields: its level, the bridge asked for, the
// platform bridge and HID driver in use, and any more given in pairs.
func startLine(level, asked, platform, hid string, more ...string) map[string]any {
	fields := map[string]any{"level": level, "asked": asked, "platform": platform, "hid": hid}
	for index := 0; index+1 < len(more); index += 2 {
		fields[more[index]] = more[index+1]
	}

	return fields
}

type logBridgesCase struct {
	name         string
	choice       bridgeChoice
	driver       hidtransport.DriverChoice
	cgoAvailable bool
	system       string
	want         map[string]any
}

func logBridgesCases() []logBridgesCase {
	cgoChoice := defaultBridges()
	pureChoice := bridgesFor(config.Configuration{
		SchemaVersion: 1, WorldClocks: nil, PlatformBridge: config.PlatformBridgePurego, Pages: nil,
	})
	driver := func(asked, using hidtransport.DriverName, unavailable error) hidtransport.DriverChoice {
		return hidtransport.DriverChoice{Asked: asked, Using: using, Unavailable: unavailable}
	}

	return []logBridgesCase{
		{
			name: "cgo as asked", choice: cgoChoice, cgoAvailable: true, system: macSystem,
			driver: driver(cgoName, cgoName, nil), want: startLine(levelInfo, cgoName, cgoName, cgoName),
		},
		{
			name: "purego as asked", choice: pureChoice, cgoAvailable: true, system: macSystem,
			driver: driver(pureName, pureName, nil), want: startLine(levelInfo, pureName, pureName, pureName),
		},
		{
			name: "cgo asked without cgo", choice: cgoChoice, cgoAvailable: false, system: macSystem,
			driver: driver(cgoName, pureName, nil),
			want: startLine(levelWarn, cgoName, pureName, pureName,
				"platform_fell_back", "cgo is not in this build; using purego",
				"hid_fell_back", "the cgo driver is not in this build; using purego"),
		},
		{
			name: "no driver built", choice: pureChoice, cgoAvailable: true, system: macSystem,
			driver: driver(pureName, "", errNoDriverBuilt),
			want:   startLine(levelWarn, pureName, pureName, "none", "hid_unavailable", noDriverAt),
		},
		{
			name: "windows", choice: pureChoice, cgoAvailable: false, system: "windows",
			driver: driver(pureName, pureName, nil),
			want:   startLine(levelInfo, pureName, "unused on this system", pureName),
		},
	}
}

func TestLogBridges_SaysAskedAndUsed(t *testing.T) {
	t.Parallel()

	for _, testCase := range logBridgesCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			logger, buffer := jsonLog()
			logBridges(t.Context(), logger, testCase.choice, testCase.driver, testCase.cgoAvailable, testCase.system)

			lines := linesWithMessage(t, buffer, startLogMessage)
			if len(lines) != 1 {
				t.Fatalf("want one line, got %d in:\n%s", len(lines), buffer.String())
			}

			for key, value := range testCase.want {
				if lines[0][key] != value {
					t.Errorf("%s: want %v, got %v", key, value, lines[0][key])
				}
			}

			for _, key := range []string{"platform_fell_back", "hid_fell_back", "hid_unavailable"} {
				if _, wanted := testCase.want[key]; !wanted && lines[0][key] != nil {
					t.Errorf("%s: want absent, got %v", key, lines[0][key])
				}
			}
		})
	}
}

func TestAcceptConfigurations_LogsAChangedBridgeOnceAndPassesEveryFileOn(t *testing.T) {
	t.Parallel()

	logger, buffer := jsonLog()
	configurations := make(chan config.Configuration, 1)
	accept := acceptConfigurations(t.Context(), configurations, logger, config.PlatformBridgeCgo)

	withBridge := func(bridge config.PlatformBridge, clocks int) config.Configuration {
		return config.Configuration{
			SchemaVersion: 1, WorldClocks: make([]config.WorldClock, clocks), PlatformBridge: bridge, Pages: nil,
		}
	}

	steps := []struct {
		configuration config.Configuration
		changeLines   int
	}{
		{withBridge("", 0), 0},
		{withBridge(config.PlatformBridgeCgo, 1), 0},
		{withBridge(config.PlatformBridgePurego, 0), 1},
		// Another edit with purego still set says nothing more.
		{withBridge(config.PlatformBridgePurego, 2), 1},
		{withBridge(config.PlatformBridgePurego, 3), 1},
		// Back to the running bridge, then away again, is a new change.
		{withBridge("", 0), 1},
		{withBridge(config.PlatformBridgePurego, 0), 2},
	}

	for index, step := range steps {
		accept(step.configuration)

		var passed config.Configuration

		select {
		case passed = <-configurations:
		default:
			t.Fatalf("step %d: the configuration was not passed on", index)
		}

		if len(passed.WorldClocks) != len(step.configuration.WorldClocks) ||
			passed.PlatformBridge != step.configuration.PlatformBridge {
			t.Fatalf("step %d: want the configuration passed on, got %+v", index, passed)
		}

		lines := linesWithMessage(t, buffer, changeLogMessage)
		if len(lines) != step.changeLines {
			t.Fatalf("step %d: want %d change lines, got %d in:\n%s", index, step.changeLines, len(lines),
				buffer.String())
		}
	}

	last := linesWithMessage(t, buffer, changeLogMessage)[1]
	if last["running"] != cgoName || last["file"] != pureName {
		t.Fatalf("want running cgo and file purego, got %v", last)
	}
}

func TestOneShotBridge_ReadsTheFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")

	writeError := os.WriteFile(path, []byte(pureFile), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}

	if bridge := oneShotBridge(path); bridge != systemadapter.BridgePurego {
		t.Fatalf("want purego, got %s", bridge)
	}

	if bridge := oneShotBridge(filepath.Join(t.TempDir(), "missing.yaml")); bridge != systemadapter.BridgeCgo {
		t.Fatalf("want cgo for a missing file, got %s", bridge)
	}
}

// TestComposition_NamesNoBridgeOutsideTheChoice: the bridge and driver
// constants appear only in bridgesFor, which makes the choice, and logBridges,
// which reports it, so every constructor and NewWatcher in this command takes
// the choice read from the file.
func TestComposition_NamesNoBridgeOutsideTheChoice(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()

	sources, globError := filepath.Glob("*.go")
	if globError != nil {
		t.Fatal(globError)
	}

	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}

		file, parseError := parser.ParseFile(fileSet, source, nil, parser.SkipObjectResolution)
		if parseError != nil {
			t.Fatal(parseError)
		}

		for _, position := range bridgeNamesOutsideTheChoice(file) {
			t.Errorf("%s: %s named outside bridgesFor and logBridges",
				fileSet.Position(position.Pos()), position.Sel.Name)
		}
	}
}

// bridgeNamesOutsideTheChoice finds the bridge and driver constants named in
// a file outside bridgesFor and logBridges.
func bridgeNamesOutsideTheChoice(file *ast.File) []*ast.SelectorExpr {
	named := map[string]bool{
		"BridgeCgo": true, "BridgePurego": true, "DriverHIDAPI": true, "DriverPurego": true,
	}

	var found []*ast.SelectorExpr

	for _, declaration := range file.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		if isFunction && (function.Name.Name == "bridgesFor" || function.Name.Name == "logBridges") {
			continue
		}

		ast.Inspect(declaration, func(node ast.Node) bool {
			selector, isSelector := node.(*ast.SelectorExpr)
			if isSelector && named[selector.Sel.Name] {
				found = append(found, selector)
			}

			return true
		})
	}

	return found
}
