package render

import (
	"fmt"
	"image"
	"image/color"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// DefaultNavigationBackground is the navigation keys' usual grey, set apart
// from an unassigned key's black.
func DefaultNavigationBackground() color.RGBA { return rgb(navigationBackgroundPacked) }

// NavigationBackgroundFrom is the navigation keys' background for a colour
// written #RRGGBB, as the configuration names it, or the usual grey when it
// names none. A colour the renderer cannot read also leaves the grey; the
// configuration decoder has already refused malformed colours.
func NavigationBackgroundFrom(written string) color.RGBA {
	background, parseError := ParseColour(written)
	if parseError != nil {
		return DefaultNavigationBackground()
	}

	return background
}

// RenderNavigation draws a navigation key on its background: a chevron
// pointing its way, the current page as number of count, and the name of the
// page it leads to, in whichever of black or white contrasts more with the
// background. Both navigation keys show the current page number, so which page
// is showing is never ambiguous (specification section 2). With no pages
// configured only the chevron is drawn.
func (renderer *Renderer) RenderNavigation(
	direction Direction,
	indication layout.PageIndication,
	hasIndication bool,
	background color.RGBA,
) image.Image {
	if renderer.frame != nil {
		return renderer.frame.place(
			renderer.frame.inner.RenderNavigation(direction, indication, hasIndication, background))
	}

	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	key := renderer.newKey(background)
	foreground := ForegroundFor(background)

	drawChevronGlyph(renderer.glyphBox(key, navigationChevronSize), foreground, direction == DirectionBack)

	if !hasIndication {
		return key
	}

	pageNumber := fmt.Sprintf("%d/%d", indication.Number, indication.Count)
	drawCentredText(key, renderer.faces.pageNumber, pageNumber, foreground, renderer.scaled(pageNumberBaseline))

	destinationName := indication.NextName
	if direction == DirectionBack {
		destinationName = indication.PreviousName
	}

	fitted, face, _ := fitText(destinationName, renderer.faces.labelSizes, renderer.labelWidth())
	drawCentredText(key, face, fitted, foreground, renderer.scaled(navigationNameBaseline))

	return key
}
