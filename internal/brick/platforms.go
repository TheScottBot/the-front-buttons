package brick

import "slices"

// The platforms a kind or a field can be marked as not running on, named as
// the editor names the machine it runs on.
const (
	PlatformMac     = "mac"
	PlatformWindows = "windows"
)

// isPlatform reports a platform there is one of.
func isPlatform(platform string) bool {
	return slices.Contains([]string{PlatformMac, PlatformWindows}, platform)
}
