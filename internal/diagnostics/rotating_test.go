package diagnostics_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
)

func TestRotatingFile_RotatesAtItsBound_AndKeepsOnlyTheNewestFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	logPath := filepath.Join(directory, "daemon.log")

	logFile, openError := diagnostics.OpenRotatingFile(logPath, 100, 2)
	if openError != nil {
		t.Fatal(openError)
	}

	line := strings.Repeat("x", 39) + "\n"
	for range 12 {
		_, writeError := logFile.Write([]byte(line))
		if writeError != nil {
			t.Fatal(writeError)
		}
	}

	closeError := logFile.Close()
	if closeError != nil {
		t.Fatal(closeError)
	}

	entries, listError := os.ReadDir(directory)
	if listError != nil {
		t.Fatal(listError)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())

		info, infoError := entry.Info()
		if infoError != nil || info.Size() > 100 {
			t.Errorf("%s is %d bytes, over the bound", entry.Name(), info.Size())
		}
	}

	if strings.Join(names, ",") != "daemon.log,daemon.log.1,daemon.log.2" {
		t.Fatalf("want the live file and two rotations, got %v", names)
	}
}

func TestRotatingFile_AppendsToAnExistingFile(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), "daemon.log")

	writeError := os.WriteFile(logPath, []byte("earlier\n"), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}

	logFile, openError := diagnostics.OpenRotatingFile(logPath, 1024, 2)
	if openError != nil {
		t.Fatal(openError)
	}

	_, _ = logFile.Write([]byte("later\n"))
	_ = logFile.Close()

	content, readError := os.ReadFile(filepath.Clean(logPath))
	if readError != nil || string(content) != "earlier\nlater\n" {
		t.Fatalf("want both lines, got %q (%v)", content, readError)
	}
}
