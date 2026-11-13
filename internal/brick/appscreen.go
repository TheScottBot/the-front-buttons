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

// ApplicationBadgeSource is implemented by a kind's parameters when its key
// can show its application's badge: the label the application shows on its
// Dock icon, such as a count of unread messages, drawn top right on the key.
// The daemon reads it to know which badges to read and where to draw them.
type ApplicationBadgeSource interface {
	// ApplicationBadge returns the application's identifier, as the
	// ApplicationLauncher takes it, and whether its badge is wanted.
	ApplicationBadge() (identifier string, wanted bool)
}
