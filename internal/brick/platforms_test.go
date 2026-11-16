package brick_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// TestRegistry_SaysWhereAKindDoesNotRun: a kind can be marked as not running
// on a platform, with why, so the editor on that platform can say so; it runs
// everywhere else, and an unregistered kind, an unknown platform or no reason
// is refused.
func TestRegistry_SaysWhereAKindDoesNotRun(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := brick.Register(registry, "live_thing", constructSampleTestDouble)
	if registerError != nil {
		t.Fatal(registerError)
	}

	if reason := registry.NotOn("live_thing", brick.PlatformWindows); reason != "" {
		t.Errorf("before marking: want it to run on Windows, got %q", reason)
	}

	setError := registry.SetNotOn("live_thing", brick.PlatformWindows, "Windows has no such thing.")
	if setError != nil {
		t.Fatal(setError)
	}

	if reason := registry.NotOn("live_thing", brick.PlatformWindows); reason != "Windows has no such thing." {
		t.Errorf("want the reason kept for Windows, got %q", reason)
	}

	if reason := registry.NotOn("live_thing", brick.PlatformMac); reason != "" {
		t.Errorf("want it still to run on a Mac, got %q", reason)
	}

	for name, refused := range map[string]error{
		"unregistered kind": registry.SetNotOn("no_such_kind", brick.PlatformWindows, "why"),
		"unknown platform":  registry.SetNotOn("live_thing", "amiga", "why"),
		"no reason":         registry.SetNotOn("live_thing", brick.PlatformMac, ""),
	} {
		if !errors.Is(refused, brick.ErrInvalidKind) {
			t.Errorf("%s: want it refused as an invalid kind, got %v", name, refused)
		}
	}

	if reason := registry.NotOn("live_thing", brick.PlatformMac); reason != "" {
		t.Errorf("a refused mark: want it to run on a Mac still, got %q", reason)
	}
}

// partlyOnMac has a check box that does nothing on Windows.
type partlyOnMac struct {
	Badge bool   `brick:"badge" description:"Show the Dock's badge." not_on:"windows"`
	Label string `brick:"label" description:"The key's text."`
}

// onAnAmiga names a platform there is none of.
type onAnAmiga struct {
	Badge bool `brick:"badge" description:"Show the badge." not_on:"amiga"`
}

// TestDeriveSchema_ReadsWhereAFieldDoesNotRun: a field's not_on tag names the
// platforms it does nothing on; a field without one runs everywhere.
func TestDeriveSchema_ReadsWhereAFieldDoesNotRun(t *testing.T) {
	t.Parallel()

	schema, deriveError := brick.DeriveSchemaOfType(reflect.TypeFor[partlyOnMac]())
	if deriveError != nil {
		t.Fatal(deriveError)
	}

	badge, _ := schema.Field("badge")
	if !slices.Equal(badge.NotOn, []string{brick.PlatformWindows}) {
		t.Errorf("badge: want it marked not on Windows, got %v", badge.NotOn)
	}

	label, _ := schema.Field("label")
	if len(label.NotOn) != 0 {
		t.Errorf("label: want it to run everywhere, got %v", label.NotOn)
	}
}

// TestDeriveSchema_RefusesAnUnknownPlatformInNotOn: a not_on tag naming a
// platform there is none of is a declaration fault, not a field that runs
// everywhere by mistake.
func TestDeriveSchema_RefusesAnUnknownPlatformInNotOn(t *testing.T) {
	t.Parallel()

	_, deriveError := brick.DeriveSchemaOfType(reflect.TypeFor[onAnAmiga]())
	if !errors.Is(deriveError, brick.ErrInvalidParameterDeclaration) {
		t.Fatalf("want an unknown platform refused, got %v", deriveError)
	}
}
