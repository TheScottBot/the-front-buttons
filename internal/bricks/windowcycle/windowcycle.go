// Package windowcycle holds the author's JetBrains window keys (evaluation,
// deck testing): jetbrains_next_window and jetbrains_previous_window bring
// forward the next or previous window across every JetBrains IDE running,
// GoLand, PyCharm and Rider among them, in a stable order, wrapping round. The
// IDEs' own next window shortcut only moves within one IDE.
package windowcycle

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The kinds' names in configuration.
const (
	NextKindName     = "jetbrains_next_window"
	PreviousKindName = "jetbrains_previous_window"
)

// toolboxIdentifier is JetBrains Toolbox, which shares the IDEs' prefix but
// is not an IDE.
const toolboxIdentifier = "com.jetbrains.toolbox"

// jetbrainsIDEs is every JetBrains application but Toolbox.
func jetbrainsIDEs() platform.WindowGroup {
	return platform.WindowGroup{Prefix: "com.jetbrains.", Excluded: []string{toolboxIdentifier}}
}

// Parameters is both kinds' one declaration of their parameters: only their
// appearance.
type Parameters struct {
	brick.Appearance
}

// Register adds both kinds to a registry.
func Register(registry *brick.Registry) error {
	for kindName, step := range map[string]int{NextKindName: 1, PreviousKindName: -1} {
		registerError := brick.Register(registry, kindName, func(_ Parameters, ports brick.Ports) (any, error) {
			if ports.WindowCycler == nil {
				return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, kindName)
			}

			return cycleKey{cycler: ports.WindowCycler, step: step}, nil
		})
		if registerError != nil {
			return registerError //nolint:wrapcheck // Register already names the kind
		}
	}

	return nil
}

// cycleKey moves a step through the JetBrains IDEs' windows.
type cycleKey struct {
	cycler platform.WindowCycler
	step   int
}

// Press brings the window a step away forward, naming any failure on the key
// by its category.
func (key cycleKey) Press(pressContext context.Context) error {
	cycleError := key.cycler.CycleWindows(pressContext, jetbrainsIDEs(), key.step)
	if cycleError != nil {
		return fmt.Errorf("moving between windows: %w",
			brick.Categorised(platform.WindowFailureCategory(cycleError), cycleError))
	}

	return nil
}
