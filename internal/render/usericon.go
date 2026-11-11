package render

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
)

// The bounds on a user supplied icon, checked before it is decoded (datum 0.10).
// A key is under a hundred pixels square, so a 512 pixel icon is already far more
// detail than can be shown, and a megabyte is far more than such an icon needs.
const (
	MaximumUserIconPixels = 512
	MaximumUserIconBytes  = 1 << 20
)

// ErrUserIconRejected reports a user icon that is not a PNG within the bounds.
var ErrUserIconRejected = errors.New("user icon rejected")

// DecodeUserIcon decodes a user's PNG icon (TD5: user icons are PNG only). The
// byte bound is checked first, then the dimensions from the PNG header, and only
// then is the image decoded, so a small file claiming enormous dimensions is
// refused before any pixels are allocated.
func DecodeUserIcon(fileContent []byte) (image.Image, error) {
	if len(fileContent) > MaximumUserIconBytes {
		return nil, fmt.Errorf("%w: %d bytes; the limit is %d", ErrUserIconRejected, len(fileContent), MaximumUserIconBytes)
	}

	header, headerError := png.DecodeConfig(bytes.NewReader(fileContent))
	if headerError != nil {
		return nil, fmt.Errorf("%w: not a PNG: %w", ErrUserIconRejected, headerError)
	}

	if header.Width < 1 || header.Height < 1 ||
		header.Width > MaximumUserIconPixels || header.Height > MaximumUserIconPixels {
		return nil, fmt.Errorf("%w: %d by %d pixels; the limit is %d on each side",
			ErrUserIconRejected, header.Width, header.Height, MaximumUserIconPixels)
	}

	icon, decodeError := png.Decode(bytes.NewReader(fileContent))
	if decodeError != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserIconRejected, decodeError)
	}

	return icon, nil
}
