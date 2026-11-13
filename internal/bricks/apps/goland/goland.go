// Package goland is GoLand's own bricks: the JetBrains actions, such as build,
// run, step over and comment, under GoLand's scope, offered on GoLand's app
// screen (evaluation, app screens).
package goland

import (
	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/jetbrains"
)

// SetID is GoLand's app set, the scope its kinds carry.
const SetID = "goland"

// kindPrefix starts each of GoLand's kind names, such as goland_build.
const kindPrefix = SetID + "_"

// The drawer's groups' sides, in tiles.
const (
	twoTiles  = 2
	fourTiles = 4
)

// Set is GoLand's app set: GoLand's bundle identifiers on macOS, its early
// access build's among them, and its executables on Windows.
func Set() brick.AppSet {
	return brick.AppSet{
		ID:          SetID,
		Name:        "GoLand",
		Identifiers: []string{"com.jetbrains.goland", "com.jetbrains.goland-EAP", "goland64.exe", "goland.exe"},
		Groups: []brick.AppGroup{
			{
				Name: "run", Label: "Run, debug, stop and rerun", Width: twoTiles, Height: twoTiles,
				Kinds: []string{kindPrefix + "run", kindPrefix + "debug", kindPrefix + "stop", kindPrefix + "rerun"},
			},
			{
				Name: "stepping", Label: "Step over, into, out and resume", Width: fourTiles, Height: 1,
				Kinds: []string{
					kindPrefix + "step_over", kindPrefix + "step_into", kindPrefix + "step_out", kindPrefix + "resume",
				},
			},
		},
	}
}

// Register adds GoLand's set and its kinds to a registry.
func Register(registry *brick.Registry) error {
	setError := registry.RegisterAppSet(Set())
	if setError != nil {
		return setError //nolint:wrapcheck // RegisterAppSet already names the set
	}

	return jetbrains.RegisterActions(registry, SetID, kindPrefix) //nolint:wrapcheck // it names the kind
}
