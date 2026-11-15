// Package pagemarker is the page_marker brick: a key in a page's corner that
// says which page is showing, and does nothing when pressed. The author marked
// pages with keys that opened a placeholder address, and asked for a brick of
// its own that copying a page leaves out (evaluation, deck testing).
package pagemarker

import "github.com/TheScottBot/the-front-buttons/internal/brick"

// KindName is the kind's name in configuration.
const KindName = brick.PageMarkerKindName

// Parameters is the kind's one declaration of its parameters: its
// appearance, and whether its page's number stands in for it. Left without a
// label or icon, the key shows its page's number either way.
type Parameters struct {
	brick.Appearance

	PageNumber bool `brick:"page_number" default:"true" replaces:"label,icon,icon_file" description:"Show the page's number, Page 2 and the icon of a 2, which follows the page wherever it moves; on an app screen, the screen's name. Untick it to give the key a label and icon of your own."`
}

// ShowsPageNumber reports whether the page's number stands in for the
// marker's own label and icons.
func (parameters Parameters) ShowsPageNumber() bool {
	return parameters.PageNumber
}

// Register adds the kind to a registry.
func Register(registry *brick.Registry) error {
	return brick.Register(registry, KindName, construct) //nolint:wrapcheck // Register already names the kind
}

func construct(Parameters, brick.Ports) (any, error) {
	return marker{}, nil
}

// marker marks its page.
type marker struct{}

// MarksPage makes it a page marker.
func (marker) MarksPage() {}
