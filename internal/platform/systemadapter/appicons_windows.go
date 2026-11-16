package systemadapter

// NewApplicationIcons returns this machine's ApplicationIcons, which on
// Windows reports that application icons are not available yet; see
// WindowsApplicationIcons. The bridge is ignored.
func NewApplicationIcons(_ Bridge) WindowsApplicationIcons { return NewWindowsApplicationIcons() }
