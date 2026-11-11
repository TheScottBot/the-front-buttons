package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// temporaryFilePattern names the temporary file beside the configuration while a
// write is in progress. It starts with a dot so a directory listing hides it and
// an editor watching the folder is unlikely to open it.
const temporaryFilePattern = ".config-*.tmp"

// AtomicFile is the part of a temporary file an atomic write uses.
type AtomicFile interface {
	io.Writer
	Sync() error
	Close() error
	Name() string
}

// AtomicFileSystem is what an atomic write needs from the filesystem. It is a
// port so a test can fail any step, as a full disk or a killed process would.
type AtomicFileSystem interface {
	CreateTemp(directory string, pattern string) (AtomicFile, error)
	Rename(fromPath string, toPath string) error
	Remove(path string) error
}

// WriteAtomically replaces the file at targetPath with content so that a reader
// sees either the old file or the new one, never a partial write: the content
// goes to a temporary file in the same directory (a rename is only atomic within
// one filesystem), is flushed to the disk, and only then renamed over the
// target. On any failure the temporary file is removed and the target is left as
// it was.
func WriteAtomically(fileSystem AtomicFileSystem, targetPath string, content []byte) error {
	temporaryFile, createError := fileSystem.CreateTemp(filepath.Dir(targetPath), temporaryFilePattern)
	if createError != nil {
		return fmt.Errorf("creating a temporary file beside %s: %w", targetPath, createError)
	}

	temporaryPath := temporaryFile.Name()

	discardTemporary := func(cause error) error {
		closeError := temporaryFile.Close()
		removeError := fileSystem.Remove(temporaryPath)

		return errors.Join(cause, closeError, removeError)
	}

	_, writeError := temporaryFile.Write(content)
	if writeError != nil {
		return discardTemporary(fmt.Errorf("writing %s: %w", temporaryPath, writeError))
	}

	syncError := temporaryFile.Sync()
	if syncError != nil {
		return discardTemporary(fmt.Errorf("flushing %s to disk: %w", temporaryPath, syncError))
	}

	closeError := temporaryFile.Close()
	if closeError != nil {
		return errors.Join(fmt.Errorf("closing %s: %w", temporaryPath, closeError), fileSystem.Remove(temporaryPath))
	}

	renameError := fileSystem.Rename(temporaryPath, targetPath)
	if renameError != nil {
		return errors.Join(fmt.Errorf("replacing %s: %w", targetPath, renameError), fileSystem.Remove(temporaryPath))
	}

	return nil
}

// LocalFileSystem is the real filesystem. The standard library's file
// operations are the same on both target operating systems, so this needs no
// platform adapter.
type LocalFileSystem struct{}

// CreateTemp creates a temporary file in directory.
//
//nolint:ireturn // the port returns a file interface so a test can substitute one
func (LocalFileSystem) CreateTemp(directory string, pattern string) (AtomicFile, error) {
	return os.CreateTemp(directory, pattern) //nolint:wrapcheck // WriteAtomically adds the context
}

// Rename renames fromPath to toPath, replacing toPath. On Windows Go uses
// MoveFileEx with MOVEFILE_REPLACE_EXISTING, so replacing works there too.
func (LocalFileSystem) Rename(fromPath string, toPath string) error {
	return os.Rename(fromPath, toPath) //nolint:wrapcheck // WriteAtomically adds the context
}

// Remove removes the file at path.
func (LocalFileSystem) Remove(path string) error {
	return os.Remove(path) //nolint:wrapcheck // WriteAtomically adds the context
}

// ReadBounded reads at most maximumBytes plus one byte from the file at path, so
// a file that has grown past the bound is seen to be too large without being
// read in full (datum 0.10).
func (LocalFileSystem) ReadBounded(path string, maximumBytes int) ([]byte, error) {
	file, openError := os.Open(path) //nolint:gosec // the path is the configuration location, not user input
	if openError != nil {
		return nil, fmt.Errorf("opening %s: %w", path, openError)
	}

	content, readError := io.ReadAll(io.LimitReader(file, int64(maximumBytes)+1))
	closeError := file.Close()

	joinedError := errors.Join(readError, closeError)
	if joinedError != nil {
		return nil, fmt.Errorf("reading %s: %w", path, joinedError)
	}

	return content, nil
}
