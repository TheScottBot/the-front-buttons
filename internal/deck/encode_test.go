package deck_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

func xlModel(t *testing.T) deck.Model {
	t.Helper()

	model, found := deck.LookupModel(elgatoVendorID, streamDeckXLProductID)
	if !found {
		t.Fatal("want the XL model")
	}

	return model
}

// The byte fixtures below are derived by hand from StreamDeckXL.py at commit
// 064f6cf: reset() writes feature 0x03 0x02 in 32 bytes; set_brightness writes
// 0x03 0x08 percent; _reset_key_stream writes 0x02 in a 1024 byte output
// report; set_key_image writes pages of 0x02 0x07 key last length_low
// length_high page_low page_high, then up to 1016 bytes, zero padded to 1024.

func TestEncodeResetKeyStream_MatchesTheReference(t *testing.T) {
	t.Parallel()

	want := make([]byte, 1024)
	want[0] = 0x02

	if got := deck.EncodeResetKeyStream(xlModel(t)); !bytes.Equal(got, want) {
		t.Fatalf("want 0x02 then 1023 zero bytes, got % x...", got[:8])
	}
}

func TestEncodeReset_MatchesTheReference(t *testing.T) {
	t.Parallel()

	want := make([]byte, 32)
	want[0], want[1] = 0x03, 0x02

	if got := deck.EncodeReset(xlModel(t)); !bytes.Equal(got, want) {
		t.Fatalf("want % x, got % x", want, got)
	}
}

func TestEncodeBrightness_MatchesTheReference_InThirtyTwoBytes(t *testing.T) {
	t.Parallel()

	want := make([]byte, 32)
	want[0], want[1], want[2] = 0x03, 0x08, 80

	got, encodeError := deck.EncodeBrightness(xlModel(t), 80)
	if encodeError != nil || !bytes.Equal(got, want) {
		t.Fatalf("want % x, got % x (%v)", want, got, encodeError)
	}
}

func TestEncodeBrightness_RefusesRatherThanClampsOutOfRangeValues(t *testing.T) {
	t.Parallel()

	for _, percent := range []int{-1, 101, 255} {
		_, encodeError := deck.EncodeBrightness(xlModel(t), percent)
		if !errors.Is(encodeError, deck.ErrOutOfRange) {
			t.Errorf("want %d refused with ErrOutOfRange, got %v", percent, encodeError)
		}
	}
}

func TestEncodeKeyImagePages_WritesExactReferenceHeaders(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte{0xAB}, 2000)

	pages, encodeError := deck.EncodeKeyImagePages(xlModel(t), 5, payload)
	if encodeError != nil {
		t.Fatalf("encoding: %v", encodeError)
	}

	wantHeaders := [][]byte{
		{0x02, 0x07, 5, 0, 0xF8, 0x03, 0x00, 0x00},
		{0x02, 0x07, 5, 1, 0xD8, 0x03, 0x01, 0x00},
	}

	if len(pages) != len(wantHeaders) {
		t.Fatalf("want %d pages, got %d", len(wantHeaders), len(pages))
	}

	var reassembled []byte

	for pageIndex, page := range pages {
		if len(page) != 1024 {
			t.Fatalf("page %d: want 1024 bytes, got %d", pageIndex, len(page))
		}

		if !bytes.Equal(page[:8], wantHeaders[pageIndex]) {
			t.Fatalf("page %d: want header % x, got % x", pageIndex, wantHeaders[pageIndex], page[:8])
		}

		payloadLength := int(page[4]) | int(page[5])<<8
		reassembled = append(reassembled, page[8:8+payloadLength]...)

		if trailing := page[8+payloadLength:]; !bytes.Equal(trailing, make([]byte, len(trailing))) {
			t.Fatalf("page %d: want zero padding after the payload", pageIndex)
		}
	}

	if !bytes.Equal(reassembled, payload) {
		t.Fatal("want the pages to reassemble into the payload")
	}
}

func TestEncodeKeyImagePages_FitsAnExactPageInOnePage(t *testing.T) {
	t.Parallel()

	pages, encodeError := deck.EncodeKeyImagePages(xlModel(t), 0, bytes.Repeat([]byte{1}, 1016))
	if encodeError != nil || len(pages) != 1 || pages[0][3] != 1 {
		t.Fatalf("want one last page, got %d pages (%v)", len(pages), encodeError)
	}

	pages, encodeError = deck.EncodeKeyImagePages(xlModel(t), 0, bytes.Repeat([]byte{1}, 1017))
	if encodeError != nil || len(pages) != 2 || pages[0][3] != 0 || pages[1][3] != 1 {
		t.Fatalf("want two pages with only the second last, got %d (%v)", len(pages), encodeError)
	}
}

func TestEncodeKeyImagePages_RefusesKeysThatDoNotExist_AndEmptyOrOversizedImages(t *testing.T) {
	t.Parallel()

	model := xlModel(t)

	refusals := map[string]struct {
		keyIndex int
		payload  []byte
	}{
		"key below zero":     {-1, []byte{1}},
		"key 32 on a 32 key": {32, []byte{1}},
		"empty image":        {0, nil},
		"oversized image":    {0, make([]byte, model.Protocol.MaximumImageBytes+1)},
	}

	for caseName, refusal := range refusals {
		_, encodeError := deck.EncodeKeyImagePages(model, refusal.keyIndex, refusal.payload)
		if !errors.Is(encodeError, deck.ErrOutOfRange) {
			t.Errorf("%s: want ErrOutOfRange, got %v", caseName, encodeError)
		}
	}
}
