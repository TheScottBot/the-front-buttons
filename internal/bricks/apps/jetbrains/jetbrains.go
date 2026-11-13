// Package jetbrains is the table of IDE actions the JetBrains IDEs share, and
// the bricks it makes for an IDE's app set: one kind per action, pressing the
// action's shortcut for the platform the daemon runs on, as the IDE's default
// keymap has it (evaluation, app screens). An IDE's own package, such as
// goland, registers them under its scope.
//
// The shortcuts are the IDEs' predefined macOS and Windows keymaps, from
// JetBrains' GoLand keymap reference (2026.3), read on 13 November 2026;
// those marked unconfirmed come from the IntelliJ platform's long standing
// defaults and are checked on the deck. A changed keymap in the IDE changes
// what a key does.
package jetbrains

import (
	"context"
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// Action is one IDE action: its name in kind names, its label on a key, and
// its shortcut on each platform.
type Action struct {
	Name    string
	Label   string
	Mac     platform.Shortcut
	Windows platform.Shortcut
}

// The modifiers, as shortcuts write them.
const (
	command = iota
	control
	option
	shift
)

// keys builds a shortcut from a key and its modifiers.
func keys(key string, held ...int) platform.Shortcut {
	shortcut := platform.Shortcut{Key: key, Control: false, Alt: false, Shift: false, Command: false}

	for _, modifier := range held {
		switch modifier {
		case command:
			shortcut.Command = true
		case control:
			shortcut.Control = true
		case option:
			shortcut.Alt = true
		case shift:
			shortcut.Shift = true
		}
	}

	return shortcut
}

// action builds a table row.
func action(name string, label string, mac platform.Shortcut, windows platform.Shortcut) Action {
	return Action{Name: name, Label: label, Mac: mac, Windows: windows}
}

// Actions is every action, grouped as the IDE's menus group them. On Windows
// Alt is option's place, and Ctrl command's.
func Actions() []Action {
	return []Action{
		// Build and run.
		action("run", "Run", keys("R", control), keys("F10", shift)),
		action("run_choose", "Run...", keys("R", control, option), keys("F10", option, shift)),
		action("rerun", "Rerun", keys("R", command), keys("F5", control)), // macOS unconfirmed
		action("stop", "Stop", keys("F2", command), keys("F2", control)),
		action("build", "Build", keys("F9", command), keys("F9", control)),
		// Debug.
		action("debug", "Debug", keys("D", control), keys("F9", shift)),
		action("toggle_breakpoint", "Breakpoint", keys("F8", command), keys("F8", control)),
		action("step_over", "Step over", keys("F8"), keys("F8")),
		action("step_into", "Step into", keys("F7"), keys("F7")),
		action("step_out", "Step out", keys("F8", shift), keys("F8", shift)),
		action("resume", "Resume", keys("R", command, option), keys("F9")),
		action("run_to_cursor", "To cursor", keys("F9", option), keys("F9", option)),
		action("evaluate", "Evaluate", keys("F8", option), keys("F8", option)),
		// Testing.
		action("run_tests", "Run tests", keys("R", control, shift), keys("F10", control, shift)), // macOS unconfirmed
		// Navigation.
		action("find_action", "Find action", keys("A", command, shift), keys("A", control, shift)),
		action("go_to_file", "Go to file", keys("O", command, shift), keys("N", control, shift)),
		action("go_to_symbol", "Go to symbol", keys("O", command, option), keys("N", control, option, shift)),
		action("recent_files", "Recent files", keys("E", command), keys("E", control)),
		action("back", "Back", keys("LeftBracket", command), keys("Left", control, option)),
		action("forward", "Forward", keys("RightBracket", command), keys("Right", control, option)),
		action("go_to_declaration", "Declaration", keys("B", command), keys("B", control)),
		action("go_to_implementation", "Implementation", keys("B", command, option), keys("B", control, option)),
		action("find_usages", "Usages", keys("F7", option), keys("F7", option)),
		action("file_structure", "Structure", keys("F12", command), keys("F12", control)),
		action("next_error", "Next error", keys("F2"), keys("F2")),
		action("previous_error", "Previous error", keys("F2", shift), keys("F2", shift)),
		// Editing.
		action("show_intentions", "Intentions", keys("Return", option), keys("Return", option)),
		action("reformat", "Reformat", keys("L", command, option), keys("L", control, option)),
		action("optimise_imports", "Imports", keys("O", control, option), keys("O", control, option)), // unconfirmed
		action("comment_line", "Comment", keys("Slash", command), keys("Slash", control)),
		action("duplicate_line", "Duplicate", keys("D", command), keys("D", control)),
		action("delete_line", "Delete line", keys("Delete", command), keys("Y", control)),
		action("move_line_up", "Line up", keys("Up", command, shift), keys("Up", control, shift)),
		action("move_line_down", "Line down", keys("Down", command, shift), keys("Down", control, shift)),
		// Refactoring.
		action("refactor_this", "Refactor", keys("T", control), keys("T", control, option, shift)),
		action("rename", "Rename", keys("F6", shift), keys("F6", shift)),
		action("extract_method", "Extract method", keys("M", command, option), keys("M", control, option)),
		action("extract_variable", "Extract variable", keys("V", command, option), keys("V", control, option)),
		action("inline", "Inline", keys("N", command, option), keys("N", control, option)),
		// Version control.
		action("commit", "Commit", keys("K", command), keys("K", control)),
		action("push", "Push", keys("K", command, shift), keys("K", control, shift)),
		action("update_project", "Pull", keys("T", command), keys("T", control)),
		action("rollback", "Rollback", keys("Z", command, option), keys("Z", control, option)),
		// Tool windows.
		action("terminal", "Terminal", keys("F12", option), keys("F12", option)),
		action("project_window", "Project", keys("1", command), keys("1", option)),
		action("run_window", "Run window", keys("4", command), keys("4", option)),
		action("debug_window", "Debug window", keys("5", command), keys("5", option)),
		action("problems_window", "Problems", keys("6", command), keys("6", option)),
		action("git_window", "Git", keys("9", command), keys("9", option)),
		action("hide_tool_windows", "Hide windows", keys("F12", command, shift), keys("F12", control, shift)),
	}
}

// Parameters is every action kind's one declaration of its parameters: only
// its appearance.
type Parameters struct {
	brick.Appearance
}

// RegisterActions registers each action as a kind of an IDE's set, scoped to
// it and named with its prefix, such as goland_build.
func RegisterActions(registry *brick.Registry, scope string, prefix string) error {
	for _, row := range Actions() {
		kindName := prefix + row.Name
		pair := platform.ShortcutPair{Mac: row.Mac, Windows: row.Windows}
		label := row.Label

		registerError := brick.RegisterScoped(registry, scope, kindName, func(_ Parameters, ports brick.Ports) (any, error) {
			if ports.ShortcutPresser == nil {
				return nil, fmt.Errorf("%w: %s needs its port", brick.ErrPortMissing, kindName)
			}

			return actionKey{label: label, pair: pair, presser: ports.ShortcutPresser}, nil
		})
		if registerError != nil {
			return registerError //nolint:wrapcheck // RegisterScoped already names the kind
		}
	}

	return nil
}

// actionKey presses one action's shortcut.
type actionKey struct {
	label   string
	pair    platform.ShortcutPair
	presser platform.ShortcutPresser
}

// Press sends this platform's shortcut for the action, naming any failure on
// the key by its category.
func (key actionKey) Press(pressContext context.Context) error {
	pressError := key.presser.PressShortcut(pressContext, key.presser.ForThisPlatform(key.pair))
	if pressError != nil {
		return fmt.Errorf("pressing %s: %w", key.label,
			brick.Categorised(platform.KeyFailureCategory(pressError), pressError))
	}

	return nil
}

// DefaultLabel is the action's label, shown when the key has none of its own.
func (key actionKey) DefaultLabel() string { return key.label }

// SendsKeystrokes marks the key as sending keystrokes.
func (actionKey) SendsKeystrokes() {}
