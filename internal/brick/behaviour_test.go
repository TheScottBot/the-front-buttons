package brick_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

func TestAppearanceOf_ReadsTheEmbeddedAppearance(t *testing.T) {
	t.Parallel()

	want := brick.Appearance{Label: docsLabel, Icon: "brand-github", Background: "#102030", IconFile: ""}
	parameters := &sampleParameters{
		Appearance: want, Address: sampleAddress, Count: 1, Enabled: true, Interval: 0, Credential: "",
	}

	appearance, found := brick.AppearanceOf(parameters)
	if !found || appearance != want {
		t.Fatalf("want %+v, got %+v (%v)", want, appearance, found)
	}
}

func TestAppearanceOf_ReportsParametersWithoutAnAppearance(t *testing.T) {
	t.Parallel()

	var valueNotPointer sampleParameters

	for _, parameters := range []any{nil, "text", &missingDescription{Address: ""}, valueNotPointer} {
		if _, found := brick.AppearanceOf(parameters); found {
			t.Errorf("want no appearance found in %T", parameters)
		}
	}
}

// networkTestDoubleError names its own reason category.
type networkTestDoubleError struct{}

func (networkTestDoubleError) Error() string          { return "the network is down" }
func (networkTestDoubleError) ReasonCategory() string { return "network" }

// errUncategorised is a plain error with no category of its own.
var errUncategorised = errors.New("anything")

func TestReasonCategoryOf_UsesTheErrorsOwnCategory_EvenWhenWrapped(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("pressing: %w", networkTestDoubleError{})

	if category := brick.ReasonCategoryOf(wrapped); category != "network" {
		t.Fatalf("want network, got %q", category)
	}
}

func TestReasonCategoryOf_FallsBackToFailed_ForAPlainError(t *testing.T) {
	t.Parallel()

	if category := brick.ReasonCategoryOf(errUncategorised); category != brick.DefaultReasonCategory {
		t.Fatalf("want %q, got %q", brick.DefaultReasonCategory, category)
	}
}
