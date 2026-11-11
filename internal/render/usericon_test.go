package render_test

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/render"
)

func encodedPNG(t *testing.T, width int, height int) []byte {
	t.Helper()

	var encoded bytes.Buffer

	encodeError := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, width, height)))
	if encodeError != nil {
		t.Fatalf("encoding: %v", encodeError)
	}

	return encoded.Bytes()
}

func TestDecodeUserIcon_AcceptsASmallPNG(t *testing.T) {
	t.Parallel()

	icon, decodeError := render.DecodeUserIcon(encodedPNG(t, 64, 64))
	if decodeError != nil || icon.Bounds().Dx() != 64 {
		t.Fatalf("want a 64 pixel icon, got %v (%v)", icon, decodeError)
	}
}

func TestDecodeUserIcon_RefusesBeforeDecoding_WhenDimensionsExceedTheBound(t *testing.T) {
	t.Parallel()

	_, decodeError := render.DecodeUserIcon(encodedPNG(t, render.MaximumUserIconPixels+1, 1))

	if !errors.Is(decodeError, render.ErrUserIconRejected) {
		t.Fatalf("want ErrUserIconRejected, got %v", decodeError)
	}
}

func TestDecodeUserIcon_RefusesFilesOverTheByteBound(t *testing.T) {
	t.Parallel()

	oversized := append(encodedPNG(t, 1, 1), make([]byte, render.MaximumUserIconBytes)...)

	_, decodeError := render.DecodeUserIcon(oversized)
	if !errors.Is(decodeError, render.ErrUserIconRejected) {
		t.Fatalf("want ErrUserIconRejected, got %v", decodeError)
	}
}

func TestDecodeUserIcon_RefusesAnythingButPNG(t *testing.T) {
	t.Parallel()

	for _, notPNG := range [][]byte{nil, []byte("GIF89a"), []byte("not an image at all")} {
		_, decodeError := render.DecodeUserIcon(notPNG)
		if !errors.Is(decodeError, render.ErrUserIconRejected) {
			t.Errorf("want ErrUserIconRejected for %q, got %v", notPNG, decodeError)
		}
	}
}
