package jetbrainsmono_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render/jetbrainsmono"
)

// fontFileName is the vendored font, as SHA256SUMS names it.
const fontFileName = "JetBrainsMono-Regular.ttf"

// TestVendoredFiles_MatchTheirRecordedDigests: the vendored files on disk, and
// the font compiled into the binary, are the ones recorded at the import, so
// neither can be edited unnoticed.
func TestVendoredFiles_MatchTheirRecordedDigests(t *testing.T) {
	t.Parallel()

	sums, openError := os.Open("SHA256SUMS")
	if openError != nil {
		t.Fatalf("opening SHA256SUMS: %v", openError)
	}

	defer func() { _ = sums.Close() }()

	checked := 0
	scanner := bufio.NewScanner(sums)

	for scanner.Scan() {
		recorded, fileName, found := strings.Cut(scanner.Text(), "  ")
		if !found {
			t.Fatalf("a malformed SHA256SUMS line: %q", scanner.Text())
		}

		contents, readError := os.ReadFile(filepath.Clean(fileName))
		if readError != nil {
			t.Fatalf("reading %s: %v", fileName, readError)
		}

		if digest := sha256.Sum256(contents); hex.EncodeToString(digest[:]) != recorded {
			t.Errorf("%s does not match its recorded digest", fileName)
		}

		if fileName == fontFileName {
			if digest := sha256.Sum256(jetbrainsmono.Regular); hex.EncodeToString(digest[:]) != recorded {
				t.Error("the embedded font does not match its recorded digest")
			}
		}

		checked++
	}

	if checked != 2 {
		t.Errorf("want two recorded files, checked %d", checked)
	}
}
