package systemadapter

import "context"

// ApplicationActivator is the ApplicationActivator on Windows, which is
// parked (evaluation, deck testing): it brings nothing forward, so key presses
// go to the window in front, as every key press does there today.
type ApplicationActivator struct{}

// NewApplicationActivator returns this machine's ApplicationActivator.
func NewApplicationActivator() ApplicationActivator { return ApplicationActivator{} }

// ActivateApplication does nothing.
func (ApplicationActivator) ActivateApplication(context.Context, string) error { return nil }
