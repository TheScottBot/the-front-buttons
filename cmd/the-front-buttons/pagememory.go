package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// The regular page the deck last showed, by its name, in the state folder, so
// a restart or an install opens the deck where it was rather than on the
// first page. It holds a page's name and nothing else.
const (
	deckPageFileName        = "deck-page"
	deckPageFilePermissions = 0o600
	// maximumPageNameBytes bounds what is read back: a page's name is at
	// most 24 characters, of up to four bytes each.
	maximumPageNameBytes = 96
)

// filePageMemory keeps the page's name in the state folder. The daemon tells
// it from its loop, which must never wait, so the file is written on a
// goroutine of its own, the newest name replacing one not yet written.
type filePageMemory struct {
	path       string
	remembered string
	names      chan string
}

// newFilePageMemory reads the page last remembered and writes each new one
// until the context ends.
func newFilePageMemory(runContext context.Context, stateDirectory string, logger *slog.Logger) *filePageMemory {
	memory := &filePageMemory{
		path:       filepath.Join(stateDirectory, deckPageFileName),
		remembered: "",
		names:      make(chan string, 1),
	}

	content, readError := os.ReadFile(filepath.Clean(memory.path))
	if name := strings.TrimSpace(string(content)); readError == nil && name != "" &&
		len(content) <= maximumPageNameBytes && utf8.ValidString(name) {
		memory.remembered = name
	}

	go memory.write(runContext, logger)

	return memory
}

// Remembered is the page last remembered, if any.
func (memory *filePageMemory) Remembered() (string, bool) {
	return memory.remembered, memory.remembered != ""
}

// Remember keeps a page's name, without waiting: a name not yet written gives
// way to the newer.
func (memory *filePageMemory) Remember(name string) {
	select {
	case memory.names <- name:
		return
	default:
	}

	select {
	case <-memory.names:
	default:
	}

	select {
	case memory.names <- name:
	default:
	}
}

// write saves each name as it comes, skipping one already saved, and logs a
// failure by its category only.
func (memory *filePageMemory) write(runContext context.Context, logger *slog.Logger) {
	written := memory.remembered

	for {
		select {
		case <-runContext.Done():
			return
		case name := <-memory.names:
			if name == written {
				continue
			}

			writeError := os.WriteFile(memory.path, []byte(name+"\n"), deckPageFilePermissions)
			if writeError != nil {
				logger.WarnContext(runContext, "deck page not remembered", slog.String("reason", "write failed"))

				continue
			}

			written = name
		}
	}
}
