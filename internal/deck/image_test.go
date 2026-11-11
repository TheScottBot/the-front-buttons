package deck_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

func TestEncodeKeyImage_FlipsBothAxes_SoTheDeckShowsItUpright(t *testing.T) {
	t.Parallel()

	model := xlModel(t)
	keyPixels := model.Protocol.KeyPixels

	// A red block in the top left corner of an otherwise black key. The XL draws
	// images flipped on both axes (KEY_FLIP = (True, True)), so the encoded image
	// must carry the block in its bottom right corner.
	upright := image.NewRGBA(image.Rect(0, 0, keyPixels, keyPixels))

	for y := range 16 {
		for x := range 16 {
			upright.Set(x, y, color.RGBA{R: 0xff, G: 0, B: 0, A: 0xff})
		}
	}

	encoded, encodeError := deck.EncodeKeyImage(model, upright)
	if encodeError != nil {
		t.Fatalf("encoding: %v", encodeError)
	}

	decoded, decodeError := jpeg.Decode(bytes.NewReader(encoded))
	if decodeError != nil {
		t.Fatalf("want a JPEG, got %v", decodeError)
	}

	isRed := func(x int, y int) bool {
		red, green, blue, _ := decoded.At(x, y).RGBA()

		return red > 0xC000 && green < 0x4000 && blue < 0x4000
	}

	if !isRed(keyPixels-8, keyPixels-8) || isRed(8, 8) {
		t.Fatal("want the block moved from the top left to the bottom right")
	}
}

func TestEncodeKeyImage_RefusesAnImageOfTheWrongSize(t *testing.T) {
	t.Parallel()

	_, encodeError := deck.EncodeKeyImage(xlModel(t), image.NewRGBA(image.Rect(0, 0, 72, 72)))
	if !errors.Is(encodeError, deck.ErrOutOfRange) {
		t.Fatalf("want ErrOutOfRange, got %v", encodeError)
	}
}

func TestEncodeKeyImage_FitsWithinTheImageBound(t *testing.T) {
	t.Parallel()

	model := xlModel(t)
	noisy := image.NewRGBA(image.Rect(0, 0, model.Protocol.KeyPixels, model.Protocol.KeyPixels))

	for pixelIndex := range noisy.Pix {
		noisy.Pix[pixelIndex] = byte(pixelIndex * 7919 % 251)
	}

	encoded, encodeError := deck.EncodeKeyImage(model, noisy)
	if encodeError != nil || len(encoded) > model.Protocol.MaximumImageBytes {
		t.Fatalf("want a worst case key within %d bytes, got %d (%v)",
			model.Protocol.MaximumImageBytes, len(encoded), encodeError)
	}
}
