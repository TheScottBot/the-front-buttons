package editor_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// pressKeyType is the press_key kind in the key form's type list.
const pressKeyType = "kind:press_key"

// TestEditor_OffersAFormatsValuesAsAChoice: the author asked for the key a
// press_key key sends to be chosen, not typed, since a typing slip breaks it
// (evaluation, deck testing). The field is a list of F13 to F24 with the saved
// value chosen, and there is no text box for it.
func TestEditor_OffersAFormatsValuesAsAChoice(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	saved := harness.post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, typeField: {pressKeyType},
		keyField: {"F15"}, labelField: {"Talk"},
	})
	if saved.Code != http.StatusSeeOther {
		t.Fatalf("want the key saved, got %d", saved.Code)
	}

	body := harness.get(t, "/key?page=0&row=1&column=1").Body.String()

	field := regexp.MustCompile(`(?s)<select id="field-key" name="field.key"[^>]*>.*?</select>`).FindString(body)
	if field == "" {
		t.Fatal("want the key field as a list")
	}

	if strings.Count(field, "<option ") != 12 || !strings.Contains(field, `<option value="F15" selected>F15</option>`) ||
		!strings.Contains(field, `<option value="F24">F24</option>`) {
		t.Fatalf("want F13 to F24 with F15 chosen, got %s", field)
	}

	if strings.Contains(body, `<input id="field-key"`) {
		t.Error("want no text box for the key")
	}
}

// TestEditor_StillRefusesAValueOutsideTheChoice: the list is only how the
// value is chosen; the schema still checks what arrives.
func TestEditor_StillRefusesAValueOutsideTheChoice(t *testing.T) {
	t.Parallel()

	recorder := newHarness(t, nil).post(t, "/key", url.Values{
		pageField: {"0"}, rowField: {"1"}, columnField: {"1"}, typeField: {pressKeyType}, keyField: {"F25"},
	})
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want F25 refused, got %d", recorder.Code)
	}
}
