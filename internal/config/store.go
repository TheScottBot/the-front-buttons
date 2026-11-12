package config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ReloadResult is what one reload of the configuration file did.
type ReloadResult string

// The results of a reload.
const (
	// ReloadAccepted means the file was valid and is now running.
	ReloadAccepted ReloadResult = "accepted"
	// ReloadRejected means the file was invalid; the last good configuration
	// keeps running.
	ReloadRejected ReloadResult = "rejected"
	// ReloadUnchanged means the file's content had not changed since the last
	// reload, so nothing was done.
	ReloadUnchanged ReloadResult = "unchanged"
	// ReloadUnreadable means the file could not be read, for example because an
	// editor was midway through replacing it; the last good configuration keeps
	// running.
	ReloadUnreadable ReloadResult = "unreadable"
)

// ReloadOutcome reports one reload.
type ReloadOutcome struct {
	Result    ReloadResult
	Rejection *RejectionError
}

// StoreFileSystem is what the store needs from the filesystem: bounded reads
// and atomic writes.
type StoreFileSystem interface {
	AtomicFileSystem
	ReadBounded(path string, maximumBytes int) ([]byte, error)
}

// Store owns the running configuration. It reloads the file when its content
// changes, keeps the last good configuration when a new one is invalid, and saves
// the editor's changes through the same file so there is one reload path
// (specification section 2).
type Store struct {
	path       string
	decoder    Decoder
	fileSystem StoreFileSystem

	mutex         sync.Mutex
	current       Configuration
	hasCurrent    bool
	lastDigest    [sha256.Size]byte
	hasDigest     bool
	lastRejection *RejectionError
}

// NewStore returns a store for the configuration file at path. Nothing is read
// until the first Reload.
func NewStore(path string, decoder Decoder, fileSystem StoreFileSystem) *Store {
	return &Store{
		path:          path,
		decoder:       decoder,
		fileSystem:    fileSystem,
		mutex:         sync.Mutex{},
		current:       Configuration{SchemaVersion: 0, Pages: nil},
		hasCurrent:    false,
		lastDigest:    [sha256.Size]byte{},
		hasDigest:     false,
		lastRejection: nil,
	}
}

// Reload reads the file and, if its content changed, decodes it. A valid file
// replaces the running configuration; an invalid or unreadable one never does.
func (store *Store) Reload() ReloadOutcome {
	fileContent, readError := store.fileSystem.ReadBounded(store.path, MaximumFileBytes)

	store.mutex.Lock()
	defer store.mutex.Unlock()

	if readError != nil {
		// Forget the digest so the file is decoded again when it reappears,
		// even with the content it had before.
		store.hasDigest = false

		return ReloadOutcome{Result: ReloadUnreadable, Rejection: nil}
	}

	digest := sha256.Sum256(fileContent)
	if store.hasDigest && digest == store.lastDigest {
		return ReloadOutcome{Result: ReloadUnchanged, Rejection: store.lastRejection}
	}

	store.lastDigest = digest
	store.hasDigest = true

	configuration, decodeError := store.decoder.Decode(fileContent)
	if decodeError != nil {
		rejection := asRejection(decodeError)
		store.lastRejection = rejection

		return ReloadOutcome{Result: ReloadRejected, Rejection: rejection}
	}

	store.current = configuration
	store.hasCurrent = true
	store.lastRejection = nil

	return ReloadOutcome{Result: ReloadAccepted, Rejection: nil}
}

func asRejection(decodeError error) *RejectionError {
	// Decode returns the rejection itself, never wrapped.
	if rejection, isRejection := decodeError.(*RejectionError); isRejection { //nolint:errorlint // see above
		return rejection
	}

	return &RejectionError{Problems: []Problem{{Path: "", Line: 0, Reason: ReasonMalformed, Detail: decodeError.Error()}}}
}

// Current returns a copy of the running configuration, and false if no valid file
// has been read yet. It is a deep copy: the editor edits what it is given, and
// those edits must reach the running configuration only through Save and a
// reload, never by sharing memory with it.
func (store *Store) Current() (Configuration, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	return store.current.clone(), store.hasCurrent
}

// LastRejection returns why the most recent file was rejected, or nil if the
// file running is the file on disk. The editor shows it.
func (store *Store) LastRejection() error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if store.lastRejection == nil {
		return nil
	}

	return store.lastRejection
}

// Save validates a configuration by encoding and decoding it, then writes it
// atomically. It does not change what is running: the next reload picks the file
// up exactly as it would a hand edit.
func (store *Store) Save(configuration Configuration) error {
	encoded, encodeError := store.decoder.Encode(configuration)
	if encodeError != nil {
		return encodeError
	}

	_, decodeError := store.decoder.Decode(encoded)
	if decodeError != nil {
		return fmt.Errorf("refusing to save: %w", decodeError)
	}

	return WriteAtomically(store.fileSystem, store.path, encoded)
}

// Watch reloads the file now and then once per interval until the context ends,
// calling onAccepted with each newly accepted configuration. It polls rather than
// subscribing to filesystem events: polling needs no dependency, behaves the same
// on both operating systems, and copes with editors that replace a file rather
// than write to it. Each change of result is logged once, by reason category and
// path, never with a value (datum 0.15).
func (store *Store) Watch(
	watchContext context.Context,
	interval time.Duration,
	onAccepted func(configuration Configuration),
	logger *slog.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var previousResult ReloadResult

	for {
		outcome := store.Reload()
		store.report(watchContext, outcome, previousResult, onAccepted, logger)

		if outcome.Result != ReloadUnchanged {
			previousResult = outcome.Result
		}

		select {
		case <-watchContext.Done():
			return
		case <-ticker.C:
		}
	}
}

// OnDisk decodes the file as it is now, without changing what is running and
// without consuming the change, so the watcher still reloads it. The editor shows
// this, so a person sees a change they saved a moment ago, and a hand edit that
// the store refused, with its problems.
func (store *Store) OnDisk() (Configuration, error) {
	fileContent, readError := store.fileSystem.ReadBounded(store.path, MaximumFileBytes)
	if readError != nil {
		return Configuration{SchemaVersion: 0, Pages: nil}, fmt.Errorf("reading the configuration file: %w", readError)
	}

	return store.decoder.Decode(fileContent)
}

func (store *Store) report(
	watchContext context.Context,
	outcome ReloadOutcome,
	previousResult ReloadResult,
	onAccepted func(configuration Configuration),
	logger *slog.Logger,
) {
	switch outcome.Result {
	case ReloadAccepted:
		logger.InfoContext(watchContext, "configuration reload", slog.String("result", string(outcome.Result)))

		current, _ := store.Current()
		onAccepted(current)
	case ReloadRejected:
		problemSummaries := make([]string, 0, len(outcome.Rejection.Problems))
		for _, problem := range outcome.Rejection.Problems {
			problemSummaries = append(problemSummaries,
				fmt.Sprintf("%s line %d: %s", problem.Path, problem.Line, problem.Reason))
		}

		logger.WarnContext(watchContext, "configuration reload",
			slog.String("result", string(outcome.Result)),
			slog.Int("problems", len(problemSummaries)),
			slog.Any("reasons", problemSummaries),
			slog.String("running", "last good configuration kept"))
	case ReloadUnreadable:
		if previousResult != ReloadUnreadable {
			logger.WarnContext(watchContext, "configuration reload",
				slog.String("result", string(outcome.Result)),
				slog.String("running", "last good configuration kept"))
		}
	case ReloadUnchanged:
	}
}
