package usericons_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/usericons"
)

func writePNG(t *testing.T, path string, side int) {
	t.Helper()

	picture := image.NewRGBA(image.Rect(0, 0, side, side))
	picture.Set(0, 0, color.RGBA{R: 0xff, G: 0, B: 0, A: 0xff})

	var encoded bytes.Buffer

	encodeError := png.Encode(&encoded, picture)
	if encodeError != nil {
		t.Fatal(encodeError)
	}

	writeError := os.WriteFile(path, encoded.Bytes(), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}
}

func iconsFolder(t *testing.T) string {
	t.Helper()

	folder := filepath.Join(t.TempDir(), "icons")

	makeError := os.Mkdir(folder, 0o700)
	if makeError != nil {
		t.Fatal(makeError)
	}

	return folder
}

func TestFolder_LoadsAPNGFromTheIconsFolder(t *testing.T) {
	t.Parallel()

	folder := iconsFolder(t)
	writePNG(t, filepath.Join(folder, "yeku.png"), 40)

	icon, loadError := usericons.NewFolder(folder).Load("yeku.png")
	if loadError != nil {
		t.Fatal(loadError)
	}

	if bounds := icon.Bounds(); bounds.Dx() != 40 || bounds.Dy() != 40 {
		t.Fatalf("want the 40 pixel icon, got %v", bounds)
	}
}

// TestFolder_RefusesANameThatIsNotAPlainPNGFileName_BeforeTouchingTheDisk: a
// planted file outside the folder must stay unread whatever the name says.
func TestFolder_RefusesANameThatIsNotAPlainPNGFileName_BeforeTouchingTheDisk(t *testing.T) {
	t.Parallel()

	folder := iconsFolder(t)
	writePNG(t, filepath.Join(filepath.Dir(folder), "outside.png"), 8)

	for _, name := range []string{"../outside.png", "sub/outside.png", "outside.jpg", ""} {
		_, loadError := usericons.NewFolder(folder).Load(name)
		if !errors.Is(loadError, usericons.ErrIconFileRefused) {
			t.Errorf("%q: want refused, got %v", name, loadError)
		}
	}
}

func TestFolder_ReportsAMissingFile(t *testing.T) {
	t.Parallel()

	_, loadError := usericons.NewFolder(iconsFolder(t)).Load("nobody.png")
	if !errors.Is(loadError, fs.ErrNotExist) {
		t.Fatalf("want not found, got %v", loadError)
	}
}

// TestFolder_RefusesAnOversizedFile_WithoutReadingAllOfIt keeps the decoder's
// byte bound: the folder reads at most one byte past it.
func TestFolder_RefusesAnOversizedFile_WithoutReadingAllOfIt(t *testing.T) {
	t.Parallel()

	folder := iconsFolder(t)

	writeError := os.WriteFile(filepath.Join(folder, "huge.png"), make([]byte, render.MaximumUserIconBytes+10), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}

	_, loadError := usericons.NewFolder(folder).Load("huge.png")
	if !errors.Is(loadError, render.ErrUserIconRejected) {
		t.Fatalf("want the oversized icon rejected, got %v", loadError)
	}
}
