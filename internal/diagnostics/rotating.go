package diagnostics

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
)

// rotatedFilePermissions keeps the log readable by its owner alone: it names
// pages, kinds and positions, which are the person's own business.
const rotatedFilePermissions = 0o600

// RotatingFile is the log file in the state folder (specification section 7):
// appended to, and rotated when the next write would take it past its bound,
// keeping a fixed number of older files, so the log can never fill the disk.
type RotatingFile struct {
	mutex   sync.Mutex
	path    string
	maximum int64
	keep    int
	file    *os.File
	size    int64
}

// OpenRotatingFile opens or creates the log at path, rotating at maximumBytes and
// keeping keepFiles older files beside it as path.1, path.2 and so on.
func OpenRotatingFile(path string, maximumBytes int64, keepFiles int) (*RotatingFile, error) {
	rotating := &RotatingFile{mutex: sync.Mutex{}, path: path, maximum: maximumBytes, keep: keepFiles, file: nil, size: 0}

	openError := rotating.open()
	if openError != nil {
		return nil, openError
	}

	return rotating, nil
}

// Write appends one log record, rotating first if it would pass the bound.
func (rotating *RotatingFile) Write(record []byte) (int, error) {
	rotating.mutex.Lock()
	defer rotating.mutex.Unlock()

	if rotating.size > 0 && rotating.size+int64(len(record)) > rotating.maximum {
		rotateError := rotating.rotate()
		if rotateError != nil {
			return 0, rotateError
		}
	}

	written, writeError := rotating.file.Write(record)
	rotating.size += int64(written)

	if writeError != nil {
		return written, fmt.Errorf("writing the log file: %w", writeError)
	}

	return written, nil
}

// Close closes the log file.
func (rotating *RotatingFile) Close() error {
	rotating.mutex.Lock()
	defer rotating.mutex.Unlock()

	closeError := rotating.file.Close()
	if closeError != nil {
		return fmt.Errorf("closing the log file: %w", closeError)
	}

	return nil
}

func (rotating *RotatingFile) open() error {
	file, openError := os.OpenFile(rotating.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, rotatedFilePermissions)
	if openError != nil {
		return fmt.Errorf("opening the log file: %w", openError)
	}

	info, statError := file.Stat()
	if statError != nil {
		return errors.Join(fmt.Errorf("reading the log file's size: %w", statError), file.Close())
	}

	rotating.file = file
	rotating.size = info.Size()

	return nil
}

// rotate moves path.N to path.N+1 from the oldest down, dropping the oldest,
// then path to path.1, and starts a new path.
func (rotating *RotatingFile) rotate() error {
	closeError := rotating.file.Close()
	if closeError != nil {
		return fmt.Errorf("closing the log file to rotate it: %w", closeError)
	}

	for generation := rotating.keep - 1; generation >= 1; generation-- {
		older := rotating.path + "." + strconv.Itoa(generation)
		newer := rotating.path + "." + strconv.Itoa(generation+1)

		renameError := os.Rename(older, newer)
		if renameError != nil && !errors.Is(renameError, os.ErrNotExist) {
			return fmt.Errorf("rotating the log file: %w", renameError)
		}
	}

	renameError := os.Rename(rotating.path, rotating.path+".1")
	if renameError != nil {
		return fmt.Errorf("rotating the log file: %w", renameError)
	}

	return rotating.open()
}
