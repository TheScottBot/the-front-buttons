// Package jetbrainsmono is the clock's typeface: JetBrains Mono Regular
// v2.304, under the SIL Open Font Licence 1.1 in OFL.txt beside it, the face
// Thyme, the author's clock, draws its time in. Both files are vendored
// verbatim from Thyme's own copy and are never edited; PROVENANCE.md records
// where they came from and their digests, which SHA256SUMS holds for the
// test.
package jetbrainsmono

import _ "embed"

// Regular is the font file's bytes, parsed once by the renderer.
//
//go:embed JetBrainsMono-Regular.ttf
var Regular []byte
