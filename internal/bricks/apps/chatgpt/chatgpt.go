// Package chatgpt is the ChatGPT desktop app's own keys, offered on its app
// screen: its chats, tabs and modes, Chat, Work and Codex, and its panels
// (evaluation, app screens).
//
// The shortcuts are the desktop app's default keybindings for each platform,
// read from the command table in ChatGPT 26.1103.60933, installed on the Mac
// as com.openai.codex, on 14 November 2026; shortcuts its website binds and the
// app does not, such as searching chats, are left out. Its Windows program's
// name is unconfirmed.
package chatgpt

import (
	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// SetID is ChatGPT's app set, the scope its kinds carry.
const SetID = "chatgpt"

// kindPrefix starts each kind's name, such as chatgpt_next_chat.
const kindPrefix = SetID + "_"

// The drawer's groups' sides, in tiles.
const (
	twoTiles   = 2
	threeTiles = 3
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

// both and apart build the table's rows.
func both(name string, label string, key string, held ...int) appkeys.Action {
	return appkeys.Both(name, label, key, held...)
}

func apart(name string, label string, mac platform.Shortcut, windows platform.Shortcut) appkeys.Action {
	return appkeys.Apart(name, label, mac, windows)
}

// Actions is every action, grouped as the app's command menu groups them.
func Actions() []appkeys.Action {
	return []appkeys.Action{
		// Chats.
		both("new_chat", "New chat", "N", command),
		both("temporary_chat", "Temporary", "N", command, shift),
		both("previous_chat", "Previous chat", "LeftBracket", command, option),
		both("next_chat", "Next chat", "RightBracket", command, option),
		both("needs_attention", "Needs me", "A", command, option),
		both("pin_chat", "Pin", "P", command, option),
		both("mark_unread", "Unread", "U", command, shift),
		both("archive_chat", "Archive", "A", command, shift),
		both("side_chat", "Side chat", "S", command, option),
		both("copy_last_code", "Copy code", "Semicolon", command, shift),
		both("find", "Find", "F", command),
		// Modes.
		apart("chat_mode", "Chat", keys("1", control), keys("1", option)),
		apart("work_mode", "Work", keys("2", control), keys("2", option)),
		apart("codex_mode", "Codex", keys("3", control), keys("3", option)),
		// Tabs.
		both("new_tab", "New tab", "T", command),
		both("previous_tab", "Previous tab", "LeftBracket", command, shift),
		both("next_tab", "Next tab", "RightBracket", command, shift),
		both("reopen_tab", "Reopen tab", "T", command, shift),
		both("close_other_tabs", "Close others", "W", command, option),
		both("back", "Back", "LeftBracket", command),
		// Panels.
		both("toggle_sidebar", "Sidebar", "B", command),
		both("toggle_bottom_panel", "Bottom panel", "J", command),
		both("toggle_review_panel", "Review panel", "B", command, option),
		both("toggle_file_tree", "Files", "E", command, shift),
		apart("terminal", "Terminal", keys("Grave", control), keys("Grave", control)),
		apart("review", "Review", keys("G", control, shift), keys("G", control, shift)),
		both("open_folder", "Open folder", "O", command),
		both("settings", "Settings", "Comma", command),
		both("keyboard_shortcuts", "Shortcuts", "Slash", command),
	}
}

// Set is ChatGPT's app set: its bundle identifier on macOS, read from the
// installed application, and its likely Windows program names, unconfirmed.
func Set() brick.AppSet {
	return brick.AppSet{
		ID:   SetID,
		Name: "ChatGPT",
		Applications: []brick.AppIdentity{
			{Name: "ChatGPT", Identifiers: []string{"com.openai.codex", "com.openai.chat", "ChatGPT.exe", "Codex.exe"}},
		},
		Groups: []brick.AppGroup{
			{
				Name: "chats", Label: "Previous and next chat", Width: twoTiles, Height: 1,
				Kinds: []string{kindPrefix + "previous_chat", kindPrefix + "next_chat"},
			},
			{
				Name: "modes", Label: "Chat, Work and Codex", Width: threeTiles, Height: 1,
				Kinds: []string{kindPrefix + "chat_mode", kindPrefix + "work_mode", kindPrefix + "codex_mode"},
			},
		},
	}
}

// Register adds ChatGPT's set and its kinds to a registry.
func Register(registry *brick.Registry) error {
	setError := registry.RegisterAppSet(Set())
	if setError != nil {
		return setError //nolint:wrapcheck // RegisterAppSet already names the set
	}

	return appkeys.Register(registry, SetID, kindPrefix, Actions()) //nolint:wrapcheck // it names the kind
}
