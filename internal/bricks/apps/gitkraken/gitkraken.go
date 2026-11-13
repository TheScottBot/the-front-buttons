// Package gitkraken is GitKraken Desktop's own keys, offered on its app
// screen: its tabs, its terminal tabs, and its repository actions, such as
// fetch, stage all and commit (evaluation, app screens).
//
// The shortcuts are GitKraken Desktop's defaults, from its keyboard shortcuts
// and terminal guides (last updated November 2026), read on 13 November
// 2026, and checked against the menus of GitKraken 12.6.2 on macOS, which
// also gave reopen closed tab and the tabs list. Reopening a tab and opening a
// repository are unconfirmed on Windows.
package gitkraken

import (
	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// SetID is GitKraken's app set, the scope its kinds carry.
const SetID = "gitkraken"

// kindPrefix starts each kind's name, such as gitkraken_next_tab.
const kindPrefix = SetID + "_"

// The drawer's groups' sides, in tiles.
const (
	twoTiles  = 2
	fourTiles = 4
)

// The modifiers, as the table writes them.
const (
	command = appkeys.Command
	control = appkeys.Control
	option  = appkeys.Option
	shift   = appkeys.Shift
)

// keys builds a shortcut from a key and its modifiers.
func keys(key string, held ...int) platform.Shortcut { return appkeys.Keys(key, held...) }

// both builds a row whose Windows shortcut is the macOS one with Ctrl for
// Command, as most of GitKraken's are.
func both(name string, label string, key string, held ...int) appkeys.Action {
	windowsHeld := make([]int, 0, len(held))

	for _, modifier := range held {
		if modifier == command {
			modifier = control
		}

		windowsHeld = append(windowsHeld, modifier)
	}

	return appkeys.Action{Name: name, Label: label, Mac: keys(key, held...), Windows: keys(key, windowsHeld...)}
}

// apart builds a row whose shortcuts differ by platform.
func apart(name string, label string, mac platform.Shortcut, windows platform.Shortcut) appkeys.Action {
	return appkeys.Action{Name: name, Label: label, Mac: mac, Windows: windows}
}

// Actions is every action, grouped as GitKraken's shortcuts guide groups
// them.
func Actions() []appkeys.Action {
	return []appkeys.Action{
		// Tabs.
		both("new_tab", "New tab", "T", command),
		both("close_tab", "Close tab", "W", command),
		both("reopen_tab", "Reopen tab", "T", command, shift), // Windows unconfirmed
		apart("next_tab", "Next tab", keys("Tab", control), keys("Tab", control)),
		apart("previous_tab", "Previous tab", keys("Tab", control, shift), keys("Tab", control, shift)),
		both("tabs_list", "Tabs", "A", command, shift),
		// Terminal tabs, in the built in terminal.
		apart("new_terminal_tab", "Terminal", keys("N", command, option), keys("N", control, shift)),
		apart("close_terminal_tab", "Close terminal", keys("W", command, option), keys("W", control, shift)),
		apart("next_terminal_tab", "Next terminal", keys("Right", command, option), keys("PageDown", control)),
		apart("previous_terminal_tab", "Previous terminal", keys("Left", command, option), keys("PageUp", control)),
		// The repository elsewhere.
		apart("open_in_terminal", "Open in terminal", keys("T", option), keys("T", option)),
		apart("open_in_file_manager", "Show files", keys("O", option), keys("O", option)),
		// Finding.
		both("open_repo", "Open repo", "O", command), // Windows unconfirmed
		both("command_palette", "Palette", "P", command),
		both("search", "Search", "F", command),
		both("file_history", "File history", "H", command, shift),
		both("diff_tool", "Diff tool", "D", command),
		// The repository.
		both("fetch_all", "Fetch", "L", command),
		both("create_branch", "Branch", "B", command),
		both("stage_all", "Stage all", "S", command, shift),
		both("unstage_all", "Unstage all", "U", command, shift),
		both("commit", "Commit", "Return", command),
		both("stage_and_commit", "Stage and commit", "Return", command, shift),
		both("commit_message", "Message", "M", command, shift),
		both("undo", "Undo", "Z", command),
		both("redo", "Redo", "Y", command),
		// The view.
		both("toggle_left_panel", "Left panel", "J", command),
		both("toggle_commit_detail", "Details", "K", command),
		apart("first_commit", "First commit", keys("Up", command), keys("Home", control)),
		apart("last_commit", "Last commit", keys("Down", command), keys("End", control)),
		both("zoom_in", "Zoom in", "Equals", command),
		both("zoom_out", "Zoom out", "Minus", command),
		both("zoom_reset", "Zoom reset", "0", command),
		apart("full_screen", "Full screen", keys("F", command, control), keys("F11")),
		both("keyboard_shortcuts", "Shortcuts", "Slash", command),
	}
}

// Set is GitKraken's app set: its bundle identifier on macOS, read from the
// installed application, and its executable on Windows.
func Set() brick.AppSet {
	return brick.AppSet{
		ID:   SetID,
		Name: "GitKraken",
		Applications: []brick.AppIdentity{
			{Name: "GitKraken", Identifiers: []string{"com.axosoft.gitkraken", "gitkraken.exe"}},
		},
		Groups: []brick.AppGroup{
			{
				Name: "tabs", Label: "Previous and next tab", Width: twoTiles, Height: 1,
				Kinds: kinds("previous_tab", "next_tab"),
			},
			{
				Name: "terminal", Label: "Terminal tabs: new, previous, next and close", Width: fourTiles, Height: 1,
				Kinds: kinds("new_terminal_tab", "previous_terminal_tab", "next_terminal_tab", "close_terminal_tab"),
			},
		},
	}
}

// kinds names actions' kinds.
func kinds(names ...string) []string {
	kindNames := make([]string, 0, len(names))
	for _, name := range names {
		kindNames = append(kindNames, kindPrefix+name)
	}

	return kindNames
}

// Register adds GitKraken's set and its kinds to a registry.
func Register(registry *brick.Registry) error {
	setError := registry.RegisterAppSet(Set())
	if setError != nil {
		return setError //nolint:wrapcheck // RegisterAppSet already names the set
	}

	return appkeys.Register(registry, SetID, kindPrefix, Actions()) //nolint:wrapcheck // it names the kind
}
