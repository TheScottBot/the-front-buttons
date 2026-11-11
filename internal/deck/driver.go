package deck

import (
	"errors"
	"fmt"
	"image"
	"sync"
	"time"
)

// ErrUnsupportedModel reports a device that is not in the model table.
var ErrUnsupportedModel = errors.New("unsupported deck model")

// Deck drives one open deck: it composes the pure protocol in this package with
// a DeckDevice that moves the bytes, so no protocol detail reaches the adapter.
type Deck struct {
	model  Model
	device DeckDevice

	// The previous key states, to turn each report into changes. Reads come from
	// one reader goroutine; the mutex keeps a concurrent caller safe anyway.
	mutex          sync.Mutex
	previousStates []bool
}

// OpenDeck opens a listed deck and resets its key stream, as the reference does
// on open, so the first key image starts cleanly.
func OpenDeck(watcher DeckWatcher, connection Connection) (*Deck, error) {
	model, supported := LookupModel(connection.VendorID, connection.ProductID)
	if !supported {
		return nil, fmt.Errorf("%w: %#04x:%#04x", ErrUnsupportedModel, connection.VendorID, connection.ProductID)
	}

	device, openError := watcher.Open(connection)
	if openError != nil {
		return nil, fmt.Errorf("opening %s: %w", model.Name, openError)
	}

	writeError := device.Write(EncodeResetKeyStream(model))
	if writeError != nil {
		return nil, errors.Join(fmt.Errorf("resetting the key stream: %w", writeError), device.Close())
	}

	return &Deck{
		model:          model,
		device:         device,
		mutex:          sync.Mutex{},
		previousStates: make([]bool, model.Rows*model.Columns),
	}, nil
}

// Model is the deck's model.
func (opened *Deck) Model() Model { return opened.model }

// SetKeyImage draws an upright image on a key.
func (opened *Deck) SetKeyImage(keyIndex int, upright image.Image) error {
	encoded, encodeError := EncodeKeyImage(opened.model, upright)
	if encodeError != nil {
		return encodeError
	}

	pages, pagesError := EncodeKeyImagePages(opened.model, keyIndex, encoded)
	if pagesError != nil {
		return pagesError
	}

	for _, page := range pages {
		writeError := opened.device.Write(page)
		if writeError != nil {
			return fmt.Errorf("drawing key %d: %w", keyIndex, writeError)
		}
	}

	return nil
}

// SetBrightness sets the backlight, from 0 to 100 percent.
func (opened *Deck) SetBrightness(percent int) error {
	report, encodeError := EncodeBrightness(opened.model, percent)
	if encodeError != nil {
		return encodeError
	}

	featureError := opened.device.SendFeatureReport(report)
	if featureError != nil {
		return fmt.Errorf("setting brightness: %w", featureError)
	}

	return nil
}

// Reset clears every key.
func (opened *Deck) Reset() error {
	featureError := opened.device.SendFeatureReport(EncodeReset(opened.model))
	if featureError != nil {
		return fmt.Errorf("resetting: %w", featureError)
	}

	return nil
}

// ReadKeyChanges waits up to timeout for one input report and returns the keys
// that changed since the last. No report in time is not an error.
func (opened *Deck) ReadKeyChanges(timeout time.Duration) ([]KeyChange, error) {
	buffer := make([]byte, opened.model.Protocol.InputReportLength)

	readLength, readError := opened.device.ReadInputReport(buffer, timeout)
	if readError != nil {
		return nil, fmt.Errorf("reading keys: %w", readError)
	}

	if readLength == 0 {
		return nil, nil
	}

	states, decodeError := DecodeKeyStates(opened.model, buffer[:readLength])
	if decodeError != nil {
		return nil, decodeError
	}

	opened.mutex.Lock()
	defer opened.mutex.Unlock()

	changes := KeyChanges(opened.previousStates, states)
	opened.previousStates = states

	return changes, nil
}

// SerialNumber reads the deck's serial number.
func (opened *Deck) SerialNumber() (string, error) {
	report, readError := opened.device.GetFeatureReport(opened.model.Protocol.SerialFeatureID,
		opened.model.Protocol.FeatureReportLength)
	if readError != nil {
		return "", fmt.Errorf("reading the serial number: %w", readError)
	}

	return DecodeSerialNumber(opened.model, report)
}

// Close releases the deck.
func (opened *Deck) Close() error {
	closeError := opened.device.Close()
	if closeError != nil {
		return fmt.Errorf("closing %s: %w", opened.model.Name, closeError)
	}

	return nil
}
