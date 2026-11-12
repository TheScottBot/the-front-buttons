package systemadapter

// NewApplicationIcons returns this machine's ApplicationIcons, which on
// Windows reports that application icons are not available yet; see
// WindowsApplicationIcons.
func NewApplicationIcons() WindowsApplicationIcons { return NewWindowsApplicationIcons() }
