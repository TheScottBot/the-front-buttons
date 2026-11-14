// Package claude is the Claude desktop app's own keys, offered on its app
// screen: its sessions, command palette, search and panes (evaluation, app
// screens).
//
// The shortcuts are read from Claude 5.11492.5, installed on the Mac as
// com.anthropic.claudefordesktop, on 14 November 2026: its menus, with
// CmdOrCtrl binding Command on macOS and Ctrl on Windows, and the keys its
// window handles itself, with Command on macOS and Ctrl elsewhere. The Code
// tab's come from its documented table (code.claude.com/docs/en/desktop,
// read the same day), which maps Command to Ctrl and Command Option to Alt on
// Windows. Back and forward are macOS menu items, so their Windows shortcuts
// are unconfirmed, as is its Windows program's name. The switch between chat
// and Code has no default shortcut, so it has no key.
package claude

import (
	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
)

// SetID is Claude's app set, the scope its kinds carry.
const SetID = "claude"

// kindPrefix starts each kind's name, such as claude_next_session.
const kindPrefix = SetID + "_"

// twoTiles is the session group's width.
const twoTiles = 2

// The modifiers, as the table writes them.
const (
	command = appkeys.Command
	control = appkeys.Control
	shift   = appkeys.Shift
)

// Actions is every action.
func Actions() []appkeys.Action {
	return []appkeys.Action{
		// Sessions.
		appkeys.Both("new_session", "New session", "N", command),
		appkeys.Both("previous_session", "Previous", "LeftBracket", command, shift),
		appkeys.Both("next_session", "Next", "RightBracket", command, shift),
		appkeys.Both("close_session", "Close", "W", command),
		appkeys.Both("reopen_session", "Reopen", "T", command, shift),
		appkeys.Both("back", "Back", "LeftBracket", command),        // Windows unconfirmed
		appkeys.Both("forward", "Forward", "RightBracket", command), // Windows unconfirmed
		appkeys.Apart("stop", "Stop", appkeys.Keys("Escape"), appkeys.Keys("Escape")),
		// The Code tab's panes and menus.
		appkeys.Apart("terminal", "Terminal", appkeys.Keys("Grave", control), appkeys.Keys("Grave", control)),
		appkeys.Both("diff", "Diff", "D", command, shift),
		appkeys.Both("browser", "Browser", "B", command, shift),
		appkeys.Both("side_chat", "Side chat", "Semicolon", command),
		appkeys.Apart("view_mode", "View", appkeys.Keys("O", control), appkeys.Keys("O", control)),
		appkeys.Both("permission_mode", "Permissions", "M", command, shift),
		appkeys.Both("model", "Model", "I", command, shift),
		appkeys.Both("effort", "Effort", "E", command, shift),
		// Finding.
		appkeys.Both("command_palette", "Commands", "K", command),
		appkeys.Both("search", "Search", "K", command, shift),
		// The window.
		appkeys.Both("toggle_sidebar", "Sidebar", "B", command),
		appkeys.Both("close_pane", "Close pane", "Backslash", command),
		appkeys.Both("open_folder", "Open folder", "O", command, shift),
		appkeys.Both("settings", "Settings", "Comma", command),
		appkeys.Both("keyboard_shortcuts", "Shortcuts", "Slash", command),
	}
}

// Set is Claude's app set: its bundle identifier on macOS, read from the
// installed application, and its likely Windows program name, unconfirmed.
func Set() brick.AppSet {
	return brick.AppSet{
		ID:   SetID,
		Name: "Claude",
		Applications: []brick.AppIdentity{
			{Name: "Claude", Identifiers: []string{"com.anthropic.claudefordesktop", "claude.exe"}},
		},
		Groups: []brick.AppGroup{
			{
				Name: "sessions", Label: "Previous and next session", Width: twoTiles, Height: 1,
				Kinds: []string{kindPrefix + "previous_session", kindPrefix + "next_session"},
			},
		},
	}
}

// Register adds Claude's set and its kinds to a registry.
func Register(registry *brick.Registry) error {
	setError := registry.RegisterAppSet(Set())
	if setError != nil {
		return setError //nolint:wrapcheck // RegisterAppSet already names the set
	}

	return appkeys.Register(registry, SetID, kindPrefix, Actions()) //nolint:wrapcheck // it names the kind
}
