package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

var errInjected = errors.New("injected failure")

// failingFileTestDouble is a temporary file that fails at a chosen step, as a
// full disk, a yanked drive or a killed process would.
type failingFileTestDouble struct {
	name            string
	writtenBytes    []byte
	failWriteAfter  int
	failSync        bool
	closeCalled     bool
	writeWasPartial bool
}

func newFailingFile(failWriteAfter int, failSync bool) *failingFileTestDouble {
	return &failingFileTestDouble{
		name:            "",
		writtenBytes:    nil,
		failWriteAfter:  failWriteAfter,
		failSync:        failSync,
		closeCalled:     false,
		writeWasPartial: false,
	}
}

func (file *failingFileTestDouble) Write(content []byte) (int, error) {
	remaining := file.failWriteAfter - len(file.writtenBytes)
	if file.failWriteAfter >= 0 && len(content) > remaining {
		file.writtenBytes = append(file.writtenBytes, content[:remaining]...)
		file.writeWasPartial = true

		return remaining, errInjected
	}

	file.writtenBytes = append(file.writtenBytes, content...)

	return len(content), nil
}

func (file *failingFileTestDouble) Sync() error {
	if file.failSync {
		return errInjected
	}

	return nil
}

func (file *failingFileTestDouble) Close() error {
	file.closeCalled = true

	return nil
}

func (file *failingFileTestDouble) Name() string { return file.name }

// failingFileSystemTestDouble hands out one failingFileTestDouble and records
// what was renamed and removed, never touching the disk.
type failingFileSystemTestDouble struct {
	file        *failingFileTestDouble
	failRename  bool
	renamedFrom string
	removed     []string
}

//nolint:ireturn // implements the port, which returns a file interface
func (fileSystem *failingFileSystemTestDouble) CreateTemp(directory string, _ string) (config.AtomicFile, error) {
	fileSystem.file.name = filepath.Join(directory, "temporary")

	return fileSystem.file, nil
}

func (fileSystem *failingFileSystemTestDouble) Rename(fromPath string, _ string) error {
	if fileSystem.failRename {
		return errInjected
	}

	fileSystem.renamedFrom = fromPath

	return nil
}

func (fileSystem *failingFileSystemTestDouble) Remove(path string) error {
	fileSystem.removed = append(fileSystem.removed, path)

	return nil
}

func TestWriteAtomically_ReplacesContent_AndLeavesNoTemporaryFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.yaml")

	writeError := os.WriteFile(configPath, []byte("old"), 0o600)
	if writeError != nil {
		t.Fatalf("seeding: %v", writeError)
	}

	atomicError := config.WriteAtomically(config.LocalFileSystem{}, configPath, []byte("new content"))
	if atomicError != nil {
		t.Fatalf("writing: %v", atomicError)
	}

	content, readError := os.ReadFile(filepath.Clean(configPath))
	if readError != nil || string(content) != "new content" {
		t.Fatalf("want the new content, got %q (%v)", content, readError)
	}

	entries, listError := os.ReadDir(directory)
	if listError != nil || len(entries) != 1 {
		t.Fatalf("want only the config file left, got %v (%v)", entries, listError)
	}
}

func TestWriteAtomically_NeverRenames_WhenWriteIsInterrupted(t *testing.T) {
	t.Parallel()

	fileSystem := &failingFileSystemTestDouble{
		file:        newFailingFile(4, false),
		failRename:  false,
		renamedFrom: "",
		removed:     nil,
	}

	atomicError := config.WriteAtomically(fileSystem, "/config/config.yaml", []byte("a full configuration"))

	if !errors.Is(atomicError, errInjected) {
		t.Fatalf("want the write failure surfaced, got %v", atomicError)
	}

	if !fileSystem.file.writeWasPartial || fileSystem.renamedFrom != "" {
		t.Fatalf("want a partial write never renamed over the original, renamed from %q", fileSystem.renamedFrom)
	}

	if len(fileSystem.removed) != 1 || !fileSystem.file.closeCalled {
		t.Fatalf("want the temporary file closed and removed, removed %v", fileSystem.removed)
	}
}

func TestWriteAtomically_NeverRenames_WhenSyncFails(t *testing.T) {
	t.Parallel()

	fileSystem := &failingFileSystemTestDouble{
		file:        newFailingFile(-1, true),
		failRename:  false,
		renamedFrom: "",
		removed:     nil,
	}

	atomicError := config.WriteAtomically(fileSystem, "/config/config.yaml", []byte("content"))

	if !errors.Is(atomicError, errInjected) || fileSystem.renamedFrom != "" || len(fileSystem.removed) != 1 {
		t.Fatalf("want no rename and the temporary removed, got %v renamed %q removed %v",
			atomicError, fileSystem.renamedFrom, fileSystem.removed)
	}
}

func TestWriteAtomically_RemovesTemporaryFile_WhenRenameFails(t *testing.T) {
	t.Parallel()

	fileSystem := &failingFileSystemTestDouble{
		file:        newFailingFile(-1, false),
		failRename:  true,
		renamedFrom: "",
		removed:     nil,
	}

	atomicError := config.WriteAtomically(fileSystem, "/config/config.yaml", []byte("content"))

	if !errors.Is(atomicError, errInjected) || len(fileSystem.removed) != 1 {
		t.Fatalf("want the rename failure surfaced and the temporary removed, got %v removed %v",
			atomicError, fileSystem.removed)
	}
}

func TestWriteAtomically_CreatesTemporaryFileBesideTheTarget(t *testing.T) {
	t.Parallel()

	fileSystem := &failingFileSystemTestDouble{
		file:        newFailingFile(-1, false),
		failRename:  false,
		renamedFrom: "",
		removed:     nil,
	}

	atomicError := config.WriteAtomically(fileSystem, filepath.Join("config", "config.yaml"), []byte("content"))
	if atomicError != nil {
		t.Fatalf("writing: %v", atomicError)
	}

	// A rename is only atomic within one filesystem, so the temporary file must
	// be created in the target's own directory.
	if filepath.Dir(fileSystem.renamedFrom) != "config" {
		t.Fatalf("want the temporary file in the target directory, got %q", fileSystem.renamedFrom)
	}
}
