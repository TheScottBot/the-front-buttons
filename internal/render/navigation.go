package render

import (
	"fmt"
	"image"

	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// RenderNavigation draws a navigation key: a chevron pointing its way, the
// current page as number of count, and the name of the page it leads to. Both
// navigation keys show the current page number, so which page is showing is
// never ambiguous (specification section 2). With no pages configured only the
// chevron is drawn.
func (renderer *Renderer) RenderNavigation(
	direction Direction,
	indication layout.PageIndication,
	hasIndication bool,
) image.Image {
	renderer.mutex.Lock()
	defer renderer.mutex.Unlock()

	key := renderer.newKey(rgb(navigationBackgroundPacked))
	foreground := ForegroundFor(rgb(navigationBackgroundPacked))

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
