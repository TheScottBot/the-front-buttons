// Package deckcontrol holds the bricks that change the deck itself, at the
// author's request (evaluation, deck testing): brightness, which toggles the
// deck between its usual brightness and a set percentage, and deck_off, which
// turns every other key off and the deck down low until it is pressed again.
package deckcontrol

import (
	"errors"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// The kinds' names in configuration.
const (
	BrightnessKindName = "brightness"
	OffKindName        = "deck_off"
)

// defaultOffPercent is how bright the deck is while off, unless set: low, but
// enough to see the key that brings it back.
const defaultOffPercent = 10

// errNoLevel reports a brightness key built without its level, which the
// configuration decoder refuses first, as the field is required.
var errNoLevel = errors.New("a brightness key needs a level")

// BrightnessParameters is the brightness kind's one declaration of its
// parameters.
type BrightnessParameters struct {
	brick.Appearance

	Level int `brick:"level,required" bounds:"1,100" description:"How bright the deck goes, in percent, when the key is pressed; pressing it again goes back to the usual brightness. The key shows the percentage unless it has a label."`
}

// OffParameters is the deck_off kind's one declaration of its parameters.
type OffParameters struct {
	brick.Appearance

	Level int `brick:"level" bounds:"1,100" description:"How bright the deck is, in percent, while every other key is off; 10 if left out. Pressing this key, or any other, turns the keys back on."`
}

// Register adds both kinds to a registry.
func Register(registry *brick.Registry) error {
	return errors.Join(
		brick.Register(registry, BrightnessKindName, constructBrightness),
		brick.Register(registry, OffKindName, constructOff),
	)
}

func constructBrightness(parameters BrightnessParameters, _ brick.Ports) (any, error) {
	if parameters.Level == 0 {
		return nil, fmt.Errorf("%s: %w", BrightnessKindName, errNoLevel)
	}

	return control{request: brick.DeckControlRequest{Action: brick.DeckControlBrightness, Percent: parameters.Level}}, nil
}

func constructOff(parameters OffParameters, _ brick.Ports) (any, error) {
	percent := parameters.Level
	if percent == 0 {
		percent = defaultOffPercent
	}

	return control{request: brick.DeckControlRequest{Action: brick.DeckControlOff, Percent: percent}}, nil
}

// control is a key whose press the daemon carries out on the deck.
type control struct {
	request brick.DeckControlRequest
}

// DeckControl is the key's request.
func (key control) DeckControl() brick.DeckControlRequest { return key.request }
