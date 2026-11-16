package editor

import "github.com/TheScottBot/the-front-buttons/internal/brick"

// What does not run on the machine the editor runs on: a kind its brick marks
// as not running on this platform is marked in the drawer and the type list,
// with why first in what the drawer says of it, and its key form says so above
// its fields; a field marked so is marked beside it. Nothing is hidden, so a
// key already in the file can still be seen and changed.

// platformName names a platform as a sentence does.
func platformName(platform string) string {
	if platform == brick.PlatformWindows {
		return "Windows"
	}

	return "a Mac"
}

// notHere is why a kind does not run on this machine, or nothing where it
// runs.
func (pages *pageHandlers) notHere(kindName string) string {
	return pages.options.Registry.NotOn(kindName, pages.options.Platform)
}

// markNotHere marks a type choice for a kind that does not run on this
// machine: after its label, and with why before what it does.
func (pages *pageHandlers) markNotHere(option typeOption, kindName string) typeOption {
	reason := pages.notHere(kindName)
	if reason == "" {
		return option
	}

	option.Label += " (not on " + platformName(pages.options.Platform) + ")"
	option.About = "Not available on " + platformName(pages.options.Platform) + ": " + reason + " " + option.About

	return option
}

// notHereNote is what a key form says above its fields when its kind does not
// run on this machine, or nothing.
func (pages *pageHandlers) notHereNote(kindName string) string {
	reason := pages.notHere(kindName)
	if reason == "" {
		return ""
	}

	return "Not available on " + platformName(pages.options.Platform) + ": " + reason
}

// markFieldsNotHere marks each field that does nothing on this machine.
func markFieldsNotHere(views []fieldView, platform string) {
	for index := range views {
		for _, notOn := range views[index].NotOn {
			if notOn == platform {
				views[index].NotHere = "Not available on " + platformName(platform) + "."
			}
		}
	}
}
