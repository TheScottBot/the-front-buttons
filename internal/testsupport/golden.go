package testsupport

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// UpdateGoldensVariable rewrites golden images instead of comparing against them
// when set to 1. It is an environment variable rather than a test flag so it
// needs no package level state.
const UpdateGoldensVariable = "UPDATE_GOLDENS"

// Golden files are private to the developer writing them.
const (
	goldenDirectoryPermissions = 0o750
	goldenFilePermissions      = 0o600
)

// CompareWithGolden checks an image pixel for pixel against the golden image
// testdata/golden/NAME.png in the calling package, or rewrites it when
// UPDATE_GOLDENS=1. Comparing pixels rather than file bytes keeps the comparison
// independent of PNG encoder settings.
func CompareWithGolden(t *testing.T, goldenName string, rendered image.Image) {
	t.Helper()

	goldenPath := filepath.Join("testdata", "golden", goldenName+".png")

	if os.Getenv(UpdateGoldensVariable) == "1" {
		writeGolden(t, goldenPath, rendered)

		return
	}

	goldenBytes, readError := os.ReadFile(filepath.Clean(goldenPath))
	if readError != nil {
		t.Fatalf("reading %s (run with %s=1 to create it): %v", goldenPath, UpdateGoldensVariable, readError)
	}

	golden, decodeError := png.Decode(bytes.NewReader(goldenBytes))
	if decodeError != nil {
		t.Fatalf("decoding %s: %v", goldenPath, decodeError)
	}

	if golden.Bounds() != rendered.Bounds() {
		t.Fatalf("%s: want bounds %v, got %v", goldenName, golden.Bounds(), rendered.Bounds())
	}

	if differing := countDifferingPixels(golden, rendered); differing > 0 {
		t.Fatalf("%s: %d pixels differ from the golden image", goldenName, differing)
	}
}

func writeGolden(t *testing.T, goldenPath string, rendered image.Image) {
	t.Helper()

	makeDirectoryError := os.MkdirAll(filepath.Dir(goldenPath), goldenDirectoryPermissions)
	if makeDirectoryError != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(goldenPath), makeDirectoryError)
	}

	var encoded bytes.Buffer

	encodeError := png.Encode(&encoded, rendered)
	if encodeError != nil {
		t.Fatalf("encoding %s: %v", goldenPath, encodeError)
	}

	writeError := os.WriteFile(goldenPath, encoded.Bytes(), goldenFilePermissions)
	if writeError != nil {
		t.Fatalf("writing %s: %v", goldenPath, writeError)
	}
}

func countDifferingPixels(golden image.Image, rendered image.Image) int {
	differing := 0

	for y := golden.Bounds().Min.Y; y < golden.Bounds().Max.Y; y++ {
		for x := golden.Bounds().Min.X; x < golden.Bounds().Max.X; x++ {
			if color.RGBAModel.Convert(golden.At(x, y)) != color.RGBAModel.Convert(rendered.At(x, y)) {
				differing++
			}
		}
	}

	return differing
}
