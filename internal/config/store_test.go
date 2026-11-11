package config_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// newStoreWithFile writes the content to a fresh config file and returns a
// store over it.
func newStoreWithFile(t *testing.T, fileContent string) (*config.Store, string) {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeConfigFile(t, configPath, fileContent)

	return config.NewStore(configPath, newTestDecoder(t), config.LocalFileSystem{}), configPath
}

func writeConfigFile(t *testing.T, configPath string, fileContent string) {
	t.Helper()

	writeError := os.WriteFile(configPath, []byte(fileContent), 0o600)
	if writeError != nil {
		t.Fatalf("writing the config file: %v", writeError)
	}
}

func requireOutcome(t *testing.T, store *config.Store, want config.ReloadResult) config.ReloadOutcome {
	t.Helper()

	outcome := store.Reload()
	if outcome.Result != want {
		t.Fatalf("want reload result %s, got %+v", want, outcome)
	}

	return outcome
}

func TestStore_Reload_AcceptsValidFile(t *testing.T) {
	t.Parallel()

	store, _ := newStoreWithFile(t, validConfiguration)

	requireOutcome(t, store, config.ReloadAccepted)

	current, hasCurrent := store.Current()
	if !hasCurrent || len(current.Pages) != 2 {
		t.Fatalf("want the decoded configuration current, got %+v", current)
	}
}

func TestStore_Reload_KeepsLastGoodConfiguration_WhenFileBecomesInvalid(t *testing.T) {
	t.Parallel()

	store, configPath := newStoreWithFile(t, validConfiguration)
	requireOutcome(t, store, config.ReloadAccepted)

	writeConfigFile(t, configPath, strings.Replace(validConfiguration, "kind: test_link", "kind: no_such_kind", 1))

	outcome := requireOutcome(t, store, config.ReloadRejected)
	if outcome.Rejection == nil || store.LastRejection() == nil {
		t.Fatalf("want the rejection reported and kept, got %+v", outcome)
	}

	current, hasCurrent := store.Current()
	if !hasCurrent || current.Pages[0].Keys[0].Kind != testsupport.LinkKindName {
		t.Fatalf("want the last good configuration still running, got %+v", current)
	}
}

func TestStore_Reload_AcceptsTheFix_AndClearsTheRejection(t *testing.T) {
	t.Parallel()

	store, configPath := newStoreWithFile(t, validConfiguration)
	requireOutcome(t, store, config.ReloadAccepted)

	writeConfigFile(t, configPath, "schema_version: 1\npages: [\n")
	requireOutcome(t, store, config.ReloadRejected)

	writeConfigFile(t, configPath, strings.Replace(validConfiguration, "name: Play", "name: Games", 1))
	requireOutcome(t, store, config.ReloadAccepted)

	current, _ := store.Current()
	if store.LastRejection() != nil || current.Pages[1].Name != "Games" {
		t.Fatalf("want the fix running and the rejection cleared, got %+v, %v", current, store.LastRejection())
	}
}

func TestStore_Reload_ReportsUnchanged_WhenContentIsTheSame(t *testing.T) {
	t.Parallel()

	store, configPath := newStoreWithFile(t, validConfiguration)
	requireOutcome(t, store, config.ReloadAccepted)

	writeConfigFile(t, configPath, validConfiguration)
	requireOutcome(t, store, config.ReloadUnchanged)
}

func TestStore_Reload_KeepsLastGood_WhenFileIsMissingOrTooLarge(t *testing.T) {
	t.Parallel()

	store, configPath := newStoreWithFile(t, validConfiguration)
	requireOutcome(t, store, config.ReloadAccepted)

	writeConfigFile(t, configPath, strings.Repeat("#", config.MaximumFileBytes+1))
	requireOutcome(t, store, config.ReloadRejected)

	removeError := os.Remove(configPath)
	if removeError != nil {
		t.Fatalf("removing: %v", removeError)
	}

	requireOutcome(t, store, config.ReloadUnreadable)

	if _, hasCurrent := store.Current(); !hasCurrent {
		t.Fatal("want the last good configuration kept through a missing file")
	}
}

func TestStore_Save_WritesThroughTheOneReloadPath(t *testing.T) {
	t.Parallel()

	store, _ := newStoreWithFile(t, validConfiguration)
	requireOutcome(t, store, config.ReloadAccepted)

	edited, _ := store.Current()
	edited.Pages[1].Name = "Saved"

	saveError := store.Save(edited)
	if saveError != nil {
		t.Fatalf("saving: %v", saveError)
	}

	// Saving does not change what is running; only a reload does, exactly as
	// for a hand edit (specification section 2: one reload path).
	beforeReload, _ := store.Current()
	if beforeReload.Pages[1].Name != "Play" {
		t.Fatalf("want the running configuration unchanged until reload, got %q", beforeReload.Pages[1].Name)
	}

	requireOutcome(t, store, config.ReloadAccepted)

	afterReload, _ := store.Current()
	if afterReload.Pages[1].Name != "Saved" {
		t.Fatalf("want the saved configuration after reload, got %q", afterReload.Pages[1].Name)
	}
}

func TestStore_Save_RefusesInvalidConfiguration_AndLeavesTheFileAlone(t *testing.T) {
	t.Parallel()

	store, configPath := newStoreWithFile(t, validConfiguration)
	requireOutcome(t, store, config.ReloadAccepted)

	invalid, _ := store.Current()
	invalid.Pages[0].Keys[0].Position = layout.Position{Row: 3, Column: 0}

	saveError := store.Save(invalid)
	if saveError == nil {
		t.Fatal("want an error saving a key on a navigation position")
	}

	content, readError := os.ReadFile(filepath.Clean(configPath))
	if readError != nil || string(content) != validConfiguration {
		t.Fatalf("want the file untouched, got %q (%v)", content, readError)
	}
}

// acceptedRecorder collects configurations the watcher delivers.
type acceptedRecorder struct {
	mutex    sync.Mutex
	accepted []config.Configuration
}

func (recorder *acceptedRecorder) record(configuration config.Configuration) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	recorder.accepted = append(recorder.accepted, configuration)
}

func (recorder *acceptedRecorder) count() int {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	return len(recorder.accepted)
}

func TestStore_Watch_DeliversEachAcceptedChange_AndStopsWhenCancelled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		store, configPath := newStoreWithFile(t, validConfiguration)

		var logBuffer bytes.Buffer

		logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
		recorder := &acceptedRecorder{mutex: sync.Mutex{}, accepted: nil}
		watchContext, cancelWatch := context.WithCancel(t.Context())

		watchDone := make(chan struct{})

		go func() {
			store.Watch(watchContext, time.Second, recorder.record, logger)
			close(watchDone)
		}()

		time.Sleep(time.Second)
		synctest.Wait()

		if recorder.count() != 1 {
			t.Fatalf("want the first load delivered, got %d deliveries", recorder.count())
		}

		writeConfigFile(t, configPath, strings.Replace(validConfiguration, "name: Play", "name: Later", 1))
		time.Sleep(time.Second)
		synctest.Wait()

		writeConfigFile(t, configPath, "not: [valid\n")
		time.Sleep(time.Second)
		synctest.Wait()

		if recorder.count() != 2 {
			t.Fatalf("want two accepted deliveries and none for the rejection, got %d", recorder.count())
		}

		cancelWatch()
		<-watchDone

		if !strings.Contains(logBuffer.String(), `"result":"rejected"`) {
			t.Fatalf("want the rejection logged, got:\n%s", logBuffer.String())
		}
	})
}

func TestStore_Watch_LogsReasonsButNeverValues(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		colleagueURL := "http://teams.example.test/l/chat/0/0?users=colleague@example.test"
		store, _ := newStoreWithFile(t, strings.Replace(validConfiguration, "https://example.test/docs", colleagueURL, 1))

		var logBuffer bytes.Buffer

		logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
		watchContext, cancelWatch := context.WithCancel(t.Context())

		watchDone := make(chan struct{})

		go func() {
			store.Watch(watchContext, time.Second, func(config.Configuration) {}, logger)
			close(watchDone)
		}()

		time.Sleep(time.Second)
		synctest.Wait()
		cancelWatch()
		<-watchDone

		logged := logBuffer.String()
		if !strings.Contains(logged, string(config.ReasonBadFormat)) || strings.Contains(logged, "colleague") {
			t.Fatalf("want the reason logged and the value never, got:\n%s", logged)
		}
	})
}

func TestStore_Current_ReturnsACopy_SoParameterEditsCannotLeakIn(t *testing.T) {
	t.Parallel()

	store, _ := newStoreWithFile(t, validConfiguration)
	requireOutcome(t, store, config.ReloadAccepted)

	edited, _ := store.Current()

	link, _ := edited.Pages[0].Keys[0].Parameters.(*testsupport.LinkParameters)
	link.Label = "Changed"

	running, _ := store.Current()

	runningLink, _ := running.Pages[0].Keys[0].Parameters.(*testsupport.LinkParameters)
	if runningLink.Label != "Docs" {
		t.Fatalf("want the running parameters untouched, got label %q", runningLink.Label)
	}
}
