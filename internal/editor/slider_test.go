package editor_test

import (
	"regexp"
	"strings"
	"testing"
)

// brightnessKeyPath is a new brightness key's editor, whose level is a
// bounded number.
const brightnessKeyPath = "/key?page=0&row=1&column=1&type=kind:brightness"

// TestEditor_BoundedNumberHasASlider: the author asked to set a brightness by
// digit and by slider. The number box is what is sent; the slider has no name,
// keeps to the same bounds, and stays hidden until the script that keeps the
// two in step shows it.
func TestEditor_BoundedNumberHasASlider(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	body := harness.get(t, brightnessKeyPath).Body.String()

	for _, want := range []string{
		`<input id="field-level" name="field.level" type="number"`,
		`<div class="slider" hidden>`,
		`<label for="field-level-slider">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %s", want)
		}
	}

	slider := regexp.MustCompile(`<input id="field-level-slider"[^>]*>`).FindString(body)

	for _, want := range []string{`type="range"`, `min="1"`, `max="100"`, `data-mirrors="field-level"`} {
		if !strings.Contains(slider, want) {
			t.Errorf("slider: want %s, got %s", want, slider)
		}
	}

	if strings.Contains(slider, "name=") {
		t.Errorf("want the slider never sent, got %s", slider)
	}

	if strings.Contains(harness.get(t, "/key?page=0&row=0&column=0").Body.String(), `class="slider"`) {
		t.Error("want no slider on a key without a bounded number")
	}

	if !strings.Contains(harness.get(t, "/static/editor.js").Body.String(), "input.slider-input") {
		t.Error("script: want it to keep the sliders and their numbers in step")
	}
}
