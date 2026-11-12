// Package usericons reads the person's own key icons: PNG files in the icons
// folder beside the configuration, named by a key's icon_file (the author's
// request, evaluation, deck testing).
package usericons

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// ErrIconFileRefused reports a name that is not a plain PNG file name, refused
// before the disk is touched.
var ErrIconFileRefused = errors.New("icon file name refused")

// Folder reads icons from one folder and nowhere else.
type Folder struct {
	directory string
}

// NewFolder returns a reader for the icons in directory.
func NewFolder(directory string) Folder {
	return Folder{directory: directory}
}

// Load reads and decodes one icon. The name must pass the png_file format, and
// the file is opened through an os.Root on the folder, so neither a name nor a
// link inside the folder can reach any other file. At most one byte past the
// decoder's bound is read, and the decoder then checks the size and the PNG's
// dimensions before decoding its pixels.
func (folder Folder) Load(fileName string) (image.Image, error) {
	if !brick.CheckFormat(brick.FormatPNGFile, fileName) {
		return nil, fmt.Errorf("%w: %q", ErrIconFileRefused, fileName)
	}

	root, rootError := os.OpenRoot(folder.directory)
	if rootError != nil {
		return nil, fmt.Errorf("opening the icons folder: %w", rootError)
	}

	defer func() { _ = root.Close() }()

	file, openError := root.Open(fileName)
	if openError != nil {
		return nil, fmt.Errorf("opening icon file %q: %w", fileName, openError)
	}

	defer func() { _ = file.Close() }()

	content, readError := io.ReadAll(io.LimitReader(file, render.MaximumUserIconBytes+1))
	if readError != nil {
		return nil, fmt.Errorf("reading icon file %q: %w", fileName, readError)
	}

	icon, decodeError := render.DecodeUserIcon(content)
	if decodeError != nil {
		return nil, fmt.Errorf("icon file %q: %w", fileName, decodeError)
	}

	return icon, nil
}
