package deck

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
)

// ErrOutOfRange reports a value the protocol cannot carry, such as a key the
// model does not have. It is refused rather than clamped, so a caller's bug
// shows instead of silently drawing on the wrong key.
var ErrOutOfRange = errors.New("value out of range for the deck protocol")

// The bounds of a brightness percentage.
const (
	minimumBrightness = 0
	maximumBrightness = 100
)

// The byte positions in a key image page header.
const (
	headerReportID = iota
	headerCommand
	headerKeyIndex
	headerIsLastPage
	headerLengthLow
	headerLengthHigh
	headerPageLow
	headerPageHigh
)

// The positions in a feature report.
const (
	featureReportIDPosition = iota
	featureCommandPosition
	featureValuePosition
)

// byteShift and lowByteMask split a 16 bit field into two little endian bytes.
const (
	byteShift   = 8
	lowByteMask = 0xFF
)

// EncodeResetKeyStream is the output report sent once on opening a deck, before
// any key image, to reset the image stream.
func EncodeResetKeyStream(model Model) []byte {
	report := make([]byte, model.Protocol.OutputReportLength)
	report[headerReportID] = model.Protocol.ResetKeyStreamReportID

	return report
}

// EncodeReset is the feature report that resets the deck, clearing every key.
func EncodeReset(model Model) []byte {
	report := make([]byte, model.Protocol.FeatureReportLength)
	report[featureReportIDPosition] = model.Protocol.FeatureReportID
	report[featureCommandPosition] = model.Protocol.ResetCommand

	return report
}

// EncodeBrightness is the feature report that sets the backlight, from 0 to 100
// percent. The reference writes 33 bytes by accident (finding TF6); this writes
// the model's feature report length.
func EncodeBrightness(model Model, percent int) ([]byte, error) {
	if percent < minimumBrightness || percent > maximumBrightness {
		return nil, fmt.Errorf("%w: brightness %d is outside %d to %d percent",
			ErrOutOfRange, percent, minimumBrightness, maximumBrightness)
	}

	report := make([]byte, model.Protocol.FeatureReportLength)
	report[featureReportIDPosition] = model.Protocol.FeatureReportID
	report[featureCommandPosition] = model.Protocol.BrightnessCommand
	report[featureValuePosition] = byte(percent)

	return report, nil
}

// EncodeKeyImagePages splits an encoded key image into the model's output
// reports: each a header naming the key, whether it is the last page, the
// payload length and the page number, then the payload, zero padded.
func EncodeKeyImagePages(model Model, keyIndex int, encodedImage []byte) ([][]byte, error) {
	protocol := model.Protocol
	keyCount := model.Rows * model.Columns

	switch {
	case keyIndex < 0 || keyIndex >= keyCount:
		return nil, fmt.Errorf("%w: key %d on a %d key deck", ErrOutOfRange, keyIndex, keyCount)
	case len(encodedImage) == 0:
		return nil, fmt.Errorf("%w: an empty image", ErrOutOfRange)
	case len(encodedImage) > protocol.MaximumImageBytes:
		return nil, fmt.Errorf("%w: a %d byte image; the limit is %d",
			ErrOutOfRange, len(encodedImage), protocol.MaximumImageBytes)
	}

	payloadPerPage := protocol.OutputReportLength - protocol.ImageHeaderLength
	pages := make([][]byte, 0, (len(encodedImage)+payloadPerPage-1)/payloadPerPage)

	for pageStart := 0; pageStart < len(encodedImage); pageStart += payloadPerPage {
		pageNumber := pageStart / payloadPerPage
		pagePayload := encodedImage[pageStart:min(pageStart+payloadPerPage, len(encodedImage))]
		isLastPage := pageStart+len(pagePayload) == len(encodedImage)

		page := make([]byte, protocol.OutputReportLength)
		page[headerReportID] = protocol.ImageReportID
		page[headerCommand] = protocol.ImageCommand
		// Every conversion to a byte below is bounded by the checks above: the key
		// is below the model's key count, a page's payload is at most 1016 bytes,
		// and the page number is below the maximum page count.
		page[headerKeyIndex] = byte(keyIndex) //nolint:gosec // bounded by the key count check

		if isLastPage {
			page[headerIsLastPage] = 1
		}

		page[headerLengthLow] = byte(len(pagePayload) & lowByteMask)
		page[headerLengthHigh] = byte(len(pagePayload) >> byteShift) //nolint:gosec // bounded by the page size
		page[headerPageLow] = byte(pageNumber & lowByteMask)
		page[headerPageHigh] = byte(pageNumber >> byteShift) //nolint:gosec // bounded by the maximum image size
		copy(page[protocol.ImageHeaderLength:], pagePayload)

		pages = append(pages, page)
	}

	return pages, nil
}

// EncodeKeyImage turns an upright key image, as the renderer draws it, into the
// bytes the model expects: flipped as the model requires, then JPEG encoded. The
// renderer and the editor preview work upright; only this function knows the
// deck draws images flipped.
func EncodeKeyImage(model Model, upright image.Image) ([]byte, error) {
	protocol := model.Protocol
	bounds := upright.Bounds()

	if bounds.Dx() != protocol.KeyPixels || bounds.Dy() != protocol.KeyPixels {
		return nil, fmt.Errorf("%w: a %d by %d image for %d pixel keys",
			ErrOutOfRange, bounds.Dx(), bounds.Dy(), protocol.KeyPixels)
	}

	oriented := image.NewRGBA(image.Rect(0, 0, protocol.KeyPixels, protocol.KeyPixels))

	for pixelRow := range protocol.KeyPixels {
		for pixelColumn := range protocol.KeyPixels {
			sourceColumn, sourceRow := pixelColumn, pixelRow
			if protocol.FlipHorizontal {
				sourceColumn = protocol.KeyPixels - 1 - pixelColumn
			}

			if protocol.FlipVertical {
				sourceRow = protocol.KeyPixels - 1 - pixelRow
			}

			oriented.Set(pixelColumn, pixelRow, upright.At(bounds.Min.X+sourceColumn, bounds.Min.Y+sourceRow))
		}
	}

	var encoded bytes.Buffer

	encodeError := jpeg.Encode(&encoded, oriented, &jpeg.Options{Quality: protocol.JPEGQuality})
	if encodeError != nil {
		return nil, fmt.Errorf("encoding a key image: %w", encodeError)
	}

	if encoded.Len() > protocol.MaximumImageBytes {
		return nil, fmt.Errorf("%w: encoded key image is %d bytes; the limit is %d",
			ErrOutOfRange, encoded.Len(), protocol.MaximumImageBytes)
	}

	return encoded.Bytes(), nil
}
