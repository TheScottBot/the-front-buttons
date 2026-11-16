package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/daemon"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/githubactions"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

const leakCheckToken = "github_pat_MUST_NEVER_BE_LOGGED_0123456789" //nolint:gosec // made up, to catch a leak

// lockedLog is a log destination safe to read while the daemon writes.
type lockedLog struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (log *lockedLog) Write(content []byte) (int, error) {
	log.mutex.Lock()
	defer log.mutex.Unlock()

	return log.buffer.Write(content) //nolint:wrapcheck // a bytes.Buffer write cannot fail
}

func (log *lockedLog) String() string {
	log.mutex.Lock()
	defer log.mutex.Unlock()

	return log.buffer.String()
}

// TestComposition_NeverLogsTheToken runs the real composition, the real
// pipeline_status brick and the real GitHub client, against a fake GitHub that
// refuses the token, and checks every log line (specification TE8).
func TestComposition_NeverLogsTheToken(t *testing.T) {
	t.Parallel()

	fakeGitHub := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(fakeGitHub.Close)

	built := testCore(t)

	credentials := testsupport.NewCredentialStoreTestDouble()

	writeError := credentials.WriteCredential("github-work", leakCheckToken)
	if writeError != nil {
		t.Fatal(writeError)
	}

	// The fake GitHub stands in for an Enterprise host the token is bound to.
	bindError := platform.BindCredentialHost(credentials, "github-work", strings.TrimPrefix(fakeGitHub.URL, "https://"))
	if bindError != nil {
		t.Fatal(bindError)
	}

	logs := &lockedLog{mutex: sync.Mutex{}, buffer: bytes.Buffer{}}
	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{
		AddSource: false, Level: slog.LevelDebug, ReplaceAttr: nil,
	}))
	counters := diagnostics.NewCounters()

	configurations := make(chan config.Configuration, 1)
	configurations <- pipelineConfiguration(t, built, fakeGitHub.URL)

	runContext, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() {
		done <- daemon.New(daemonOptions(built, logger, counters, credentials,
			githubactions.NewClient(fakeGitHub.Client(), credentials, counters),
			testsupport.NewDeckWatcherTestDouble(), func() string { return "http://127.0.0.1:1/" },
			systemadapter.NewKeyPresser(runContext, func() {}, systemadapter.BridgeCgo), nil,
			systemadapter.BridgeCgo)).Run(runContext, configurations)
	}()

	waitForLog(logs, `"reason":"unauthorised"`)
	cancel()

	runError := <-done
	if runError != nil {
		t.Fatal(runError)
	}

	// The leak is checked first, so a leak is always reported as one.
	logged := logs.String()
	if strings.Contains(logged, leakCheckToken) || strings.Contains(logged, "MUST_NEVER") {
		t.Fatalf("the token appeared in the log:\n%s", logged)
	}

	if !strings.Contains(logged, `"reason":"unauthorised"`) {
		t.Fatalf("want the refused token reported by reason, got:\n%s", logged)
	}
}

// testCore is the core start builds, through the cgo bridge.
func testCore(t *testing.T) core {
	t.Helper()

	parts, catalogueError := buildCatalogue()
	if catalogueError != nil {
		t.Fatal(catalogueError)
	}

	built, buildError := buildCore(parts, systemadapter.BridgeCgo)
	if buildError != nil {
		t.Fatal(buildError)
	}

	return built
}

// waitForLog waits up to ten seconds for a log to hold some text.
func waitForLog(logs *lockedLog, wanted string) {
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(logs.String(), wanted) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
}

// pipelineConfiguration is one pipeline_status key pointed at the fake GitHub.
func pipelineConfiguration(t *testing.T, built core, apiBaseURL string) config.Configuration {
	t.Helper()

	decoder := config.NewDecoder(built.registry, built.presetTable, built.grid)

	configuration, decodeError := decoder.Decode([]byte(`schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: pipeline_status
        parameters:
          repository: octo-org/hello-world
          workflow: ci.yml
          branch: main
          credential: github-work
          api_base_url: ` + apiBaseURL + `
`))
	if decodeError != nil {
		t.Fatal(decodeError)
	}

	return configuration
}

// brokenStandardError fails every write, as standard error does in a Windows
// build with no console started by its logon task.
type brokenStandardError struct{}

func (brokenStandardError) Write([]byte) (int, error) { return 0, os.ErrInvalid }

// TestLog_ReachesTheFile_WhenStandardErrorFails: the log file is the only log a
// daemon with no console has, so a failing standard error must not stop it.
func TestLog_ReachesTheFile_WhenStandardErrorFails(t *testing.T) {
	t.Parallel()

	var file bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(logDestination(&file, brokenStandardError{}), nil))
	logger.InfoContext(t.Context(), "written")

	if !strings.Contains(file.String(), `"msg":"written"`) {
		t.Fatalf("want the line in the file, got %q", file.String())
	}
}
