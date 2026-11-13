// Package tablericons is the bundled icon set (TD5): the vendored Tabler icon
// font, drawn through the same font rasteriser as key labels, so icons scale to
// any key size and take whatever colour contrasts with the key's background.
//
// The font and its stylesheet are vendored verbatim from @tabler/icons-webfont
// 3.49.0 (datum 0.13); PROVENANCE.md records their digests. Icon names are read
// from the stylesheet at load, so the name table is never copied by hand.
package tablericons

import (
	"embed"
	"errors"
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed tabler-icons.ttf tabler-icons.css
var vendoredFiles embed.FS

// The vendored file names, as embedded.
const (
	fontFileName       = "tabler-icons.ttf"
	stylesheetFileName = "tabler-icons.css"
)

// The plausible icon sizes. An icon is drawn inside a key, so its size is bounded
// by the largest plausible key.
const (
	minimumIconPixels = 8
	maximumIconPixels = 512
)

// centringDivisor halves the spare space either side of a centred glyph.
const centringDivisor = 2

// typographicDotsPerInch makes a face's size in points equal its size in pixels.
const typographicDotsPerInch = 72

// iconRule matches one rule of the stylesheet: one or more icon selectors and the
// codepoint their glyph sits at. Several selectors can share a glyph.
var iconRule = regexp.MustCompile(`([^{}]+)\{\s*content:\s*"\\([0-9a-fA-F]+)";\s*\}`)

// iconSelector matches one icon's selector within a rule.
var iconSelector = regexp.MustCompile(`\.ti-([a-z0-9-]+):before`)

// ErrVendoredFontUnreadable reports that the embedded font or stylesheet did not
// parse, which means the vendored files themselves are damaged.
var ErrVendoredFontUnreadable = errors.New("vendored icon font unreadable")

// IconSet draws Tabler icons by name.
type IconSet struct {
	parsedFont *opentype.Font
	codepoints map[string]rune

	// Faces are cached per size, and font faces are not safe for concurrent
	// use, so drawing is serialised.
	mutex sync.Mutex
	faces map[int]font.Face
}

// Load parses the embedded font and reads the icon names from the embedded
// stylesheet.
func Load() (*IconSet, error) {
	fontBytes, fontReadError := vendoredFiles.ReadFile(fontFileName)
	stylesheet, stylesheetReadError := vendoredFiles.ReadFile(stylesheetFileName)

	readError := errors.Join(fontReadError, stylesheetReadError)
	if readError != nil {
		return nil, fmt.Errorf("%w: %w", ErrVendoredFontUnreadable, readError)
	}

	parsedFont, parseError := opentype.Parse(fontBytes)
	if parseError != nil {
		return nil, fmt.Errorf("%w: %w", ErrVendoredFontUnreadable, parseError)
	}

	codepoints := make(map[string]rune)

	for _, rule := range iconRule.FindAllStringSubmatch(string(stylesheet), -1) {
		codepoint, codepointError := strconv.ParseInt(rule[2], 16, 32)
		if codepointError != nil {
			return nil, fmt.Errorf("%w: codepoint %q: %w", ErrVendoredFontUnreadable, rule[2], codepointError)
		}

		for _, selector := range iconSelector.FindAllStringSubmatch(rule[1], -1) {
			codepoints[selector[1]] = rune(codepoint)
		}
	}

	if len(codepoints) == 0 {
		return nil, fmt.Errorf("%w: no icon names found in the stylesheet", ErrVendoredFontUnreadable)
	}

	return &IconSet{parsedFont: parsedFont, codepoints: codepoints, mutex: sync.Mutex{}, faces: map[int]font.Face{}}, nil
}

// Count is the number of icon names available.
func (icons *IconSet) Count() int {
	return len(icons.codepoints)
}

// Names returns every icon name, for the editor's palette.
func (icons *IconSet) Names() []string {
	names := make([]string, 0, len(icons.codepoints))
	for iconName := range icons.codepoints {
		names = append(names, iconName)
	}

	return names
}

// IconMask draws the named icon centred in a square of sizePixels, as a mask of
// coverage. It reports false for an unknown name or an implausible size, and the
// renderer then draws its missing icon marker.
func (icons *IconSet) IconMask(iconName string, sizePixels int) (*image.Alpha, bool) {
	codepoint, known := icons.codepoints[strings.ToLower(iconName)]
	if !known || sizePixels < minimumIconPixels || sizePixels > maximumIconPixels {
		return nil, false
	}

	icons.mutex.Lock()
	defer icons.mutex.Unlock()

	face, faceError := icons.faceFor(sizePixels)
	if faceError != nil {
		return nil, false
	}

	glyphBounds, _, hasGlyph := face.GlyphBounds(codepoint)
	if !hasGlyph {
		return nil, false
	}

	// The ink's middle is placed on the square's middle exactly. Rounding the
	// ink's width up and the spare space down each moved the ink left and up
	// by up to half a pixel, and dropped the ink's fractional start, so the
	// off key's power icon sat over a pixel left of centre (evaluation, deck
	// testing).
	middle := fixed.I(sizePixels) / centringDivisor
	mask := image.NewAlpha(image.Rect(0, 0, sizePixels, sizePixels))
	drawer := font.Drawer{
		Dst:  mask,
		Src:  image.Opaque,
		Face: face,
		Dot: fixed.Point26_6{
			X: middle - (glyphBounds.Min.X+glyphBounds.Max.X)/centringDivisor,
			Y: middle - (glyphBounds.Min.Y+glyphBounds.Max.Y)/centringDivisor,
		},
	}
	drawer.DrawString(string(codepoint))

	return mask, true
}

//nolint:ireturn // font.Face is the font library's own abstraction over faces
func (icons *IconSet) faceFor(sizePixels int) (font.Face, error) {
	if face, cached := icons.faces[sizePixels]; cached {
		return face, nil
	}

	face, faceError := opentype.NewFace(icons.parsedFont, &opentype.FaceOptions{
		Size:    float64(sizePixels),
		DPI:     typographicDotsPerInch,
		Hinting: font.HintingNone,
	})
	if faceError != nil {
		return nil, fmt.Errorf("building a %d pixel icon face: %w", sizePixels, faceError)
	}

	icons.faces[sizePixels] = face

	return face, nil
}
