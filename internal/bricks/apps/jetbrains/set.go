package jetbrains

import (
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/apps/appkeys"
)

// SetID is the JetBrains IDEs' app set, the scope their kinds carry.
const SetID = "jetbrains"

// kindPrefix starts each kind's name, such as jetbrains_build.
const kindPrefix = SetID + "_"

// The drawer's groups' sides, in tiles.
const (
	twoTiles  = 2
	fourTiles = 4
)

// ide is one IDE the set is for: its name, its bundle identifiers on macOS,
// and its executable's name on Windows.
type ide struct {
	name       string
	bundles    []string
	executable string
}

// ides are the IDEs built on the IntelliJ platform, which share its actions
// and default keymaps. Each macOS bundle identifier is joined by its early
// access build's, the same with "-EAP" after it, and each Windows executable
// by its 32 bit name, without the 64. GoLand's, IntelliJ IDEA's, PyCharm's and
// Rider's bundle identifiers were read from their installed applications on
// 13 November 2026; the rest are as JetBrains publishes them, and matching
// ignores case. Android Studio is Google's, built on the same platform.
func ides() []ide {
	return []ide{
		{
			name: "IntelliJ IDEA", bundles: []string{"com.jetbrains.intellij", "com.jetbrains.intellij.ce"},
			executable: "idea64.exe",
		},
		{
			name: "PyCharm", bundles: []string{"com.jetbrains.pycharm", "com.jetbrains.pycharm.ce"},
			executable: "pycharm64.exe",
		},
		{name: "GoLand", bundles: []string{"com.jetbrains.goland"}, executable: "goland64.exe"},
		{name: "WebStorm", bundles: []string{"com.jetbrains.WebStorm"}, executable: "webstorm64.exe"},
		{name: "PhpStorm", bundles: []string{"com.jetbrains.PhpStorm"}, executable: "phpstorm64.exe"},
		{name: "Rider", bundles: []string{"com.jetbrains.rider"}, executable: "rider64.exe"},
		{name: "CLion", bundles: []string{"com.jetbrains.CLion"}, executable: "clion64.exe"},
		{name: "RubyMine", bundles: []string{"com.jetbrains.rubymine"}, executable: "rubymine64.exe"},
		{name: "DataGrip", bundles: []string{"com.jetbrains.datagrip"}, executable: "datagrip64.exe"},
		{name: "DataSpell", bundles: []string{"com.jetbrains.dataspell"}, executable: "dataspell64.exe"},
		{name: "RustRover", bundles: []string{"com.jetbrains.rustrover"}, executable: "rustrover64.exe"},
		{name: "Aqua", bundles: []string{"com.jetbrains.aqua"}, executable: "aqua64.exe"},
		{name: "Writerside", bundles: []string{"com.jetbrains.writerside"}, executable: "writerside64.exe"},
		{name: "Android Studio", bundles: []string{"com.google.android.studio"}, executable: "studio64.exe"},
	}
}

// Set is the JetBrains IDEs' app set: every IDE above, each with its own
// screen and name, offered the same keys and groups.
func Set() brick.AppSet {
	applications := make([]brick.AppIdentity, 0, len(ides()))

	for _, row := range ides() {
		identifiers := make([]string, 0, len(row.bundles)*twoTiles+twoTiles)
		for _, bundle := range row.bundles {
			identifiers = append(identifiers, bundle, bundle+"-EAP")
		}

		identifiers = append(identifiers, row.executable, strings.Replace(row.executable, "64.exe", ".exe", 1))
		applications = append(applications, brick.AppIdentity{Name: row.name, Identifiers: identifiers})
	}

	return brick.AppSet{
		ID:           SetID,
		Name:         "JetBrains IDEs",
		Applications: applications,
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

// Register adds the JetBrains IDEs' set and its kinds to a registry.
func Register(registry *brick.Registry) error {
	setError := registry.RegisterAppSet(Set())
	if setError != nil {
		return setError //nolint:wrapcheck // RegisterAppSet already names the set
	}

	return appkeys.Register(registry, SetID, kindPrefix, Actions()) //nolint:wrapcheck // it names the kind
}
