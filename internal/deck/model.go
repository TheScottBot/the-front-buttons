package deck

import (
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// Protocol is every fact about how a model's reports are laid out. The protocol
// code reads only these, with no branches on model (specification section 2), so
// supporting another model is another row in Models.
type Protocol struct {
	KeyPixels      int
	FlipHorizontal bool
	FlipVertical   bool
	// JPEGQuality is the quality the reference encodes key images at.
	JPEGQuality int
	// ContentPixels, ContentLeft and ContentTop place a key's content inside
	// a black frame, measured on the deck: the XL draws each 96 pixel image a
	// little up and left of its window (evaluation, deck testing).
	ContentPixels int
	ContentLeft   int
	ContentTop    int

	OutputReportLength  int
	ImageHeaderLength   int
	MaximumImageBytes   int
	FeatureReportLength int
	InputReportLength   int
	InputHeaderLength   int

	ResetKeyStreamReportID byte
	ImageReportID          byte
	ImageCommand           byte
	FeatureReportID        byte
	ResetCommand           byte
	BrightnessCommand      byte
	SerialFeatureID        byte
	SerialTextOffset       int
	FirmwareFeatureID      byte
	FirmwareTextOffset     int
}

// Model is one supported deck: its USB identity, its grid and its protocol.
type Model struct {
	Name      string
	VendorID  uint16
	ProductID uint16
	Rows      int
	Columns   int
	Protocol  Protocol
}

// elgatoVendorID is Elgato's USB vendor ID.
const elgatoVendorID = 0x0FD9

// The XL's protocol facts, read from python-elgato-streamdeck at commit
// 064f6cf7a83146316b76cf84c946952302182091 (src/StreamDeck/Devices/StreamDeckXL.py)
// on 11 November 2026. On 12 November 2026 experiment 1 on a Stream Deck XL V2
// (0x008F) on macOS confirmed the key order, image orientation, output and
// feature lengths, brightness and the serial read; the deck sends 512 byte input
// reports, of which hidapi returns the 36 asked for (evaluation, deck testing).
const (
	xlKeyPixels           = 96
	xlJPEGQuality         = 100
	xlOutputReportLength  = 1024
	xlImageHeaderLength   = 8
	xlFeatureReportLength = 32
	xlKeyCount            = 32
	xlInputHeaderLength   = 4
	xlRows                = 4
	xlColumns             = 8
	// The report IDs, commands and text offsets, in the order StreamDeckXL.py
	// uses them.
	xlResetKeyStreamReportID = 0x02
	xlImageReportID          = 0x02
	xlImageCommand           = 0x07
	xlFeatureReportID        = 0x03
	xlResetCommand           = 0x02
	xlBrightnessCommand      = 0x08
	xlSerialFeatureID        = 0x06
	xlSerialTextOffset       = 2
	xlFirmwareFeatureID      = 0x05
	xlFirmwareTextOffset     = 6
	// The product IDs that share the StreamDeckXL class.
	xlProductID       = 0x006C
	xlV2ProductID     = 0x008F
	xlModuleProductID = 0x00BA
	// xlMaximumImagePages bounds one key image to 64 pages of 1016 bytes; a
	// 96 pixel JPEG at the reference's quality of 100 is a few pages even for a
	// worst case image, and the page number field could otherwise run to 65,535.
	xlMaximumImagePages = 64

	// The content square of a key and its place, from probes on the author's
	// XL V2 on 12 November 2026: about three pixels hidden at the top, two at
	// the left, and two to three of black seen beyond the right edge and two to
	// three times that below, looking straight on. Thicker black at the top
	// and left evens out what is seen; provisional until checked on the deck.
	xlContentPixels = 84
	xlContentLeft   = 8
	xlContentTop    = 10
)

func streamDeckXLProtocol() Protocol {
	return Protocol{
		KeyPixels:              xlKeyPixels,
		FlipHorizontal:         true,
		FlipVertical:           true,
		JPEGQuality:            xlJPEGQuality,
		ContentPixels:          xlContentPixels,
		ContentLeft:            xlContentLeft,
		ContentTop:             xlContentTop,
		OutputReportLength:     xlOutputReportLength,
		ImageHeaderLength:      xlImageHeaderLength,
		MaximumImageBytes:      xlMaximumImagePages * (xlOutputReportLength - xlImageHeaderLength),
		FeatureReportLength:    xlFeatureReportLength,
		InputReportLength:      xlInputHeaderLength + xlKeyCount,
		InputHeaderLength:      xlInputHeaderLength,
		ResetKeyStreamReportID: xlResetKeyStreamReportID,
		ImageReportID:          xlImageReportID,
		ImageCommand:           xlImageCommand,
		FeatureReportID:        xlFeatureReportID,
		ResetCommand:           xlResetCommand,
		BrightnessCommand:      xlBrightnessCommand,
		SerialFeatureID:        xlSerialFeatureID,
		SerialTextOffset:       xlSerialTextOffset,
		FirmwareFeatureID:      xlFirmwareFeatureID,
		FirmwareTextOffset:     xlFirmwareTextOffset,
	}
}

// Models is the model table. The three XL product IDs share the class
// StreamDeckXL in the reference's DeviceManager, so they share every protocol
// fact; the Stream Deck + XL (0x00C6) is a different device and is not listed.
func Models() []Model {
	xlModel := func(name string, productID uint16) Model {
		return Model{
			Name:      name,
			VendorID:  elgatoVendorID,
			ProductID: productID,
			Rows:      xlRows,
			Columns:   xlColumns,
			Protocol:  streamDeckXLProtocol(),
		}
	}

	return []Model{
		xlModel("Stream Deck XL", xlProductID),
		xlModel("Stream Deck XL V2", xlV2ProductID),
		xlModel("Stream Deck XL V2 module", xlModuleProductID),
	}
}

// LookupModel finds the model for a USB vendor and product ID.
func LookupModel(vendorID uint16, productID uint16) (Model, bool) {
	for _, model := range Models() {
		if model.VendorID == vendorID && model.ProductID == productID {
			return model, true
		}
	}

	var absentModel Model

	return absentModel, false
}

// Grid is the model's key grid, from which the layout takes reserved positions
// and key numbers.
func (model Model) Grid() (layout.Grid, error) {
	grid, gridError := layout.NewGrid(model.Rows, model.Columns)
	if gridError != nil {
		return layout.Grid{}, fmt.Errorf("%s: %w", model.Name, gridError)
	}

	return grid, nil
}
