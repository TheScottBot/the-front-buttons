package brick

// ApplicationScreenSource is implemented by a kind's parameters when its key
// can open an application's screen: a page of the deck given to that
// application, shown in place of the regular pages until its return key is
// pressed (evaluation, app screens). The configuration decoder reads it to
// check each such key has its page, and the daemon to switch to it, so neither
// needs to know any kind by name.
type ApplicationScreenSource interface {
	// ApplicationScreen returns the application's identifier, as the
	// ApplicationLauncher takes it, and whether its screen is wanted.
	ApplicationScreen() (identifier string, wanted bool)
}
