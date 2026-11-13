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

// ShadeTolerance is how far, in shades of 255, a pixel's channel may differ
// from the golden's and still count as the same. The goldens are made on the
// author's arm64 Mac and checked on CI's amd64 Windows runner, whose rounding
// in the icon rasteriser shades a curve's edge a little differently; the
// brand-github golden failed there on four pixels (evaluation, CI). A change
// to what is drawn moves an edge pixel by far more than this.
const ShadeTolerance = 8

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

	if differing, largest := DifferingPixels(golden, rendered); differing > 0 {
		t.Fatalf("%s: %d pixels differ from the golden image by more than %d shades, the largest by %d",
			goldenName, differing, ShadeTolerance, largest)
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

// DifferingPixels counts the pixels in which any channel differs from the
// golden's by more than ShadeTolerance, and reports the largest difference in
// any channel, so a failure says how far off it is.
func DifferingPixels(golden image.Image, rendered image.Image) (int, int) {
	differing, largest := 0, 0

	for y := golden.Bounds().Min.Y; y < golden.Bounds().Max.Y; y++ {
		for x := golden.Bounds().Min.X; x < golden.Bounds().Max.X; x++ {
			want, _ := color.RGBAModel.Convert(golden.At(x, y)).(color.RGBA)
			got, _ := color.RGBAModel.Convert(rendered.At(x, y)).(color.RGBA)

			difference := max(channelDifference(want.R, got.R), channelDifference(want.G, got.G),
				channelDifference(want.B, got.B), channelDifference(want.A, got.A))
			largest = max(largest, difference)

			if difference > ShadeTolerance {
				differing++
			}
		}
	}

	return differing, largest
}

func channelDifference(want uint8, got uint8) int {
	if want > got {
		return int(want - got)
	}

	return int(got - want)
}

// SameImage reports whether two images match pixel for pixel.
func SameImage(first image.Image, second image.Image) bool {
	if first.Bounds() != second.Bounds() {
		return false
	}

	for y := first.Bounds().Min.Y; y < first.Bounds().Max.Y; y++ {
		for x := first.Bounds().Min.X; x < first.Bounds().Max.X; x++ {
			if color.RGBAModel.Convert(first.At(x, y)) != color.RGBAModel.Convert(second.At(x, y)) {
				return false
			}
		}
	}

	return true
}
