package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
)

// TestFilePageMemory_KeepsThePageAcrossRestarts: a page remembered by one
// daemon is the page the next one reads, from the state folder. It runs in a
// synctest bubble, whose Wait returns once the writer has written the name it
// was given and is waiting for another.
func TestFilePageMemory_KeepsThePageAcrossRestarts(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		stateDirectory := t.TempDir()
		runContext, cancel := context.WithCancel(t.Context())
		logger := slog.New(slog.DiscardHandler)

		first := newFilePageMemory(runContext, stateDirectory, logger)
		if _, kept := first.Remembered(); kept {
			t.Fatal("want nothing remembered in a new state folder")
		}

		first.Remember("Work")
		first.Remember("Play")
		synctest.Wait()
		cancel()

		second := newFilePageMemory(t.Context(), stateDirectory, logger)
		if name, kept := second.Remembered(); !kept || name != "Play" {
			t.Fatalf("want Play remembered across the restart, got %q", name)
		}
	})
}

// TestFilePageMemory_IgnoresAFileItDidNotWrite: an empty or overlong file is
// taken as nothing remembered.
func TestFilePageMemory_IgnoresAFileItDidNotWrite(t *testing.T) {
	t.Parallel()

	for _, content := range []string{"", "\n", strings.Repeat("x", maximumPageNameBytes+1)} {
		stateDirectory := t.TempDir()

		writeError := os.WriteFile(filepath.Join(stateDirectory, deckPageFileName), []byte(content), 0o600)
		if writeError != nil {
			t.Fatal(writeError)
		}

		if _, kept := newFilePageMemory(t.Context(), stateDirectory, slog.New(slog.DiscardHandler)).Remembered(); kept {
			t.Errorf("want %q taken as nothing remembered", content)
		}
	}
}
