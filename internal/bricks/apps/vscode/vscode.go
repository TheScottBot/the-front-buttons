// Package vscode is Visual Studio Code's own keys, offered on its app screen:
// GitHub Copilot's chat and suggestions, and the editor's panels (evaluation,
// app screens).
//
// The shortcuts are VS Code's and GitHub Copilot's defaults, from
// code.visualstudio.com's AI features cheat sheet, default keybindings and
// inline suggestions pages (footers dated 11 November 2026) and GitHub's
// Copilot keyboard shortcuts for VS Code, read on 14 November 2026. Shortcuts
// those pages give only for when the chat has focus, such as a new chat on
// Command N, are left out: in the editor the same keys make a new file. The
// bundle identifier, com.microsoft.VSCode, was read from the installed
// application; none of these has been tried on Windows.
package vscode

import (
	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
)

// SetID is VS Code's app set, the scope its kinds carry.
const SetID = "vscode"

// kindPrefix starts each kind's name, such as vscode_inline_chat.
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

// Actions is every action.
func Actions() []appkeys.Action {
	keys := appkeys.Keys

	return []appkeys.Action{
		// Copilot's chat.
		appkeys.Apart("chat", "Chat", keys("I", control, command), keys("I", control, option)),
		appkeys.Both("inline_chat", "Inline chat", "I", command),
		appkeys.Both("quick_chat", "Quick chat", "L", command, option, shift),
		appkeys.Both("agent", "Agent", "I", command, shift),
		appkeys.Both("model_picker", "Model", "Period", command, option),
		appkeys.Both("changes", "Changes", "B", command, option),
		// Copilot's suggestions.
		appkeys.Apart("previous_suggestion", "Previous", keys("LeftBracket", option), keys("LeftBracket", option)),
		appkeys.Apart("next_suggestion", "Next", keys("RightBracket", option), keys("RightBracket", option)),
		appkeys.Apart("suggest", "Suggest", keys("Backslash", option), keys("Backslash", option)),
		appkeys.Apart("accept", "Accept", keys("Tab"), keys("Tab")),
		appkeys.Both("accept_word", "Accept word", "Right", command),
		appkeys.Apart("dismiss", "Dismiss", keys("Escape"), keys("Escape")),
		appkeys.Apart("more_suggestions", "More", keys("Return", control), keys("Return", control)),
		// The editor.
		appkeys.Both("command_palette", "Commands", "P", command, shift),
		appkeys.Both("quick_open", "Open file", "P", command),
		appkeys.Both("toggle_sidebar", "Sidebar", "B", command),
		appkeys.Apart("terminal", "Terminal", keys("Grave", control), keys("Grave", control)),
	}
}

// Set is VS Code's app set: its bundle identifiers on macOS, the Insiders
// build's among them, and its executables on Windows.
func Set() brick.AppSet {
	return brick.AppSet{
		ID:   SetID,
		Name: "VS Code",
		Applications: []brick.AppIdentity{
			{Name: "VS Code", Identifiers: []string{"com.microsoft.VSCode", "Code.exe"}},
			{Name: "VS Code Insiders", Identifiers: []string{"com.microsoft.VSCodeInsiders", "Code - Insiders.exe"}},
		},
		Groups: []brick.AppGroup{
			{
				Name: "chat", Label: "Chat and inline chat", Width: twoTiles, Height: 1,
				Kinds: []string{kindPrefix + "chat", kindPrefix + "inline_chat"},
			},
			{
				Name: "suggestions", Label: "Suggestions: previous, next, accept and dismiss", Width: fourTiles,
				Height: 1, Kinds: []string{
					kindPrefix + "previous_suggestion", kindPrefix + "next_suggestion", kindPrefix + "accept",
					kindPrefix + "dismiss",
				},
			},
		},
	}
}

// Register adds VS Code's set and its kinds to a registry.
func Register(registry *brick.Registry) error {
	setError := registry.RegisterAppSet(Set())
	if setError != nil {
		return setError //nolint:wrapcheck // RegisterAppSet already names the set
	}

	return appkeys.Register(registry, SetID, kindPrefix, Actions()) //nolint:wrapcheck // it names the kind
}
