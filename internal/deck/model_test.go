package deck_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck"
)

// Identity values from python-elgato-streamdeck at commit 064f6cf
// (src/StreamDeck/ProductIDs.py), read in the Plan stage.
const (
	elgatoVendorID            = 0x0FD9
	streamDeckXLProductID     = 0x006C
	streamDeckXLV2ProductID   = 0x008F
	streamDeckXLModuleID      = 0x00BA
	streamDeckPlusXLProductID = 0x00C6
	streamDeckOriginalProduct = 0x0060
)

func TestLookupModel_FindsEveryXLVariant_WithTheSameProtocolFacts(t *testing.T) {
	t.Parallel()

	first, firstFound := deck.LookupModel(elgatoVendorID, streamDeckXLProductID)
	if !firstFound {
		t.Fatal("want the XL found")
	}

	for _, productID := range []uint16{streamDeckXLV2ProductID, streamDeckXLModuleID} {
		variant, found := deck.LookupModel(elgatoVendorID, productID)
		if !found {
			t.Fatalf("want product %#04x found", productID)
		}

		if variant.Protocol != first.Protocol || variant.Rows != first.Rows || variant.Columns != first.Columns {
			t.Fatalf("want product %#04x to share the XL's facts, got %+v", productID, variant)
		}
	}
}

func TestLookupModel_CarriesTheXLFactsReadFromSource(t *testing.T) {
	t.Parallel()

	model, _ := deck.LookupModel(elgatoVendorID, streamDeckXLProductID)
	protocol := model.Protocol

	facts := map[string][2]int{
		"rows":                  {model.Rows, 4},
		"columns":               {model.Columns, 8},
		"key pixels":            {protocol.KeyPixels, 96},
		"output report length":  {protocol.OutputReportLength, 1024},
		"image header length":   {protocol.ImageHeaderLength, 8},
		"feature report length": {protocol.FeatureReportLength, 32},
		"input report length":   {protocol.InputReportLength, 36},
		"input header length":   {protocol.InputHeaderLength, 4},
	}

	for factName, gotAndWant := range facts {
		if gotAndWant[0] != gotAndWant[1] {
			t.Errorf("%s: want %d, got %d", factName, gotAndWant[1], gotAndWant[0])
		}
	}

	if !protocol.FlipHorizontal || !protocol.FlipVertical {
		t.Error("want the XL's images flipped on both axes")
	}
}

func TestLookupModel_RefusesOtherDecks_AndOtherVendors(t *testing.T) {
	t.Parallel()

	refused := map[string][2]uint16{
		"Stream Deck + XL, a different device": {elgatoVendorID, streamDeckPlusXLProductID},
		"original Stream Deck, not supported":  {elgatoVendorID, streamDeckOriginalProduct},
		"another vendor with the XL's ID":      {0x1234, streamDeckXLProductID},
	}

	for caseName, identity := range refused {
		if _, found := deck.LookupModel(identity[0], identity[1]); found {
			t.Errorf("%s: want no model", caseName)
		}
	}
}

func TestModels_GridBuildsForEveryRow(t *testing.T) {
	t.Parallel()

	for _, model := range deck.Models() {
		grid, gridError := model.Grid()
		if gridError != nil || grid.KeyCount() != model.Rows*model.Columns {
			t.Errorf("%s: grid wrong: %v (%v)", model.Name, grid, gridError)
		}
	}
}

// TestXL_CarriesTheContentInsetMeasuredOnTheDeck: probes on the author's XL V2
// showed each key's image drawn a little up and left of its window; the inset
// places a key's content so the black seen around it is even (evaluation, deck
// testing). It must fit on the key and leave room to draw.
func TestXL_CarriesTheContentInsetMeasuredOnTheDeck(t *testing.T) {
	t.Parallel()

	model, _ := deck.LookupModel(0x0FD9, 0x008F)
	protocol := model.Protocol

	if protocol.ContentLeft != 8 || protocol.ContentTop != 9 || protocol.ContentPixels != 82 ||
		protocol.ContentBulge != 1.5 || protocol.ContentCornerRadius != 6 {
		t.Fatalf("want content 82 pixels at 8, 9, bulging 1.5 with corners of 6; got %+v", protocol)
	}

	if protocol.ContentLeft+protocol.ContentPixels > protocol.KeyPixels ||
		protocol.ContentTop+protocol.ContentPixels > protocol.KeyPixels {
		t.Fatal("want the content to fit on the key")
	}
}
