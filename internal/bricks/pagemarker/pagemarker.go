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

// description says what the marker does, for the editor's drawer.
const description = "Marks its page and does nothing when tapped. Held for half a second, it opens the page " +
	"dashboard on the deck: the time across its top row, with a world icon for the world clocks, the " +
	"regular pages on its second row and the app screens on its third, each shown when pressed, with the " +
	"marker in the top left to come back."

// Register adds the kind to a registry, with what it does.
func Register(registry *brick.Registry) error {
	registerError := brick.Register(registry, KindName, construct)
	if registerError != nil {
		return registerError //nolint:wrapcheck // Register already names the kind
	}

	return registry.SetDescription(KindName, description) //nolint:wrapcheck // the kind was registered just now
}

func construct(Parameters, brick.Ports) (any, error) {
	return marker{}, nil
}

// marker marks its page.
type marker struct{}

// MarksPage makes it a page marker.
func (marker) MarksPage() {}
