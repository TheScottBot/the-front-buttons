package config

import (
	"go.yaml.in/yaml/v4"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// decodeNavigationBackground reads the navigation keys' background: a colour
// written #RRGGBB, as a key's own background is. Anything else, empty text
// included, is refused, since leaving the field out already means the usual
// grey.
func decodeNavigationBackground(backgroundNode *yaml.Node, list *problemList) string {
	text, isText := textOf(backgroundNode)
	if !isText {
		list.addf(backgroundNode, navigationBackgroundName, ReasonWrongType, "expected a colour written #RRGGBB")

		return ""
	}

	if !brick.CheckFormat(brick.FormatColour, text) {
		list.addf(backgroundNode, navigationBackgroundName, ReasonBadFormat, "must be a colour written #RRGGBB")

		return ""
	}

	return text
}
