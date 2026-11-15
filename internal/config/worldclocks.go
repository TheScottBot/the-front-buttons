package config

import (
	"fmt"
	"regexp"
	"time"

	// The zone database, embedded so a world clock's zone is checked the
	// same on every machine, whatever its own copy holds.
	_ "time/tzdata"
	"unicode/utf8"

	"go.yaml.in/yaml/v4"
)

// clockLabelPattern is what a world clock's label may hold: letters and
// digits, every one of which the clock's face draws.
var clockLabelPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// localZone is the zone name the time package reads as this machine's own,
// which is not a place a world clock can show.
const localZone = "Local"

// decodeWorldClocks reads the world view's clocks: one to MaximumWorldClocks,
// each with a label that fits its key, none repeated, and a zone the zone
// database knows. An empty list is refused, as leaving the field out already
// means the defaults and no clocks would leave the world view nothing to show.
func decodeWorldClocks(clocksNode *yaml.Node, list *problemList) []WorldClock {
	if clocksNode.Kind != yaml.SequenceNode {
		list.addf(clocksNode, worldClocksName, ReasonWrongType, "expected a list of clocks")

		return nil
	}

	if count := len(clocksNode.Content); count < 1 || count > MaximumWorldClocks {
		reason := ReasonOutOfBounds
		if count > MaximumWorldClocks {
			reason = ReasonTooMany
		}

		list.addf(clocksNode, worldClocksName, reason, "%d clocks; there must be between 1 and %d", count, MaximumWorldClocks)

		return nil
	}

	clocks := make([]WorldClock, 0, len(clocksNode.Content))
	labels := map[string]bool{}

	for clockIndex, clockNode := range clocksNode.Content {
		clockPath := fmt.Sprintf("%s[%d]", worldClocksName, clockIndex)

		entries, isMapping := list.mappingEntries(clockNode, clockPath, clockLabelName, clockZoneName)
		if !isMapping {
			continue
		}

		label := decodeClockLabel(entries, clockNode, clockPath, list)
		if labelNode := entries[clockLabelName]; label != "" && labels[label] {
			list.addf(labelNode, clockPath+"."+clockLabelName, ReasonDuplicateLabel, "another clock has this label")
		}

		labels[label] = true

		clocks = append(clocks, WorldClock{Label: label, Zone: decodeClockZone(entries, clockNode, clockPath, list)})
	}

	return clocks
}

// requiredText reads a required text entry, reporting it missing or not text.
func requiredText(
	entries map[string]*yaml.Node, parent *yaml.Node, parentPath string, entryName string, list *problemList,
) (string, *yaml.Node, bool) {
	node, present := entries[entryName]
	if !present {
		list.addf(parent, parentPath, ReasonMissing, "missing required field %q", entryName)

		return "", nil, false
	}

	text, isText := textOf(node)
	if !isText {
		list.addf(node, parentPath+"."+entryName, ReasonWrongType, "expected text")

		return "", nil, false
	}

	return text, node, true
}

func decodeClockLabel(entries map[string]*yaml.Node, clockNode *yaml.Node, clockPath string, list *problemList) string {
	label, node, found := requiredText(entries, clockNode, clockPath, clockLabelName, list)
	if !found {
		return ""
	}

	path := clockPath + "." + clockLabelName

	if length := utf8.RuneCountInString(label); length < 1 || length > MaximumClockLabelCharacters {
		list.addf(node, path, ReasonOutOfBounds, "must be between 1 and %d characters", MaximumClockLabelCharacters)

		return ""
	}

	if !clockLabelPattern.MatchString(label) {
		list.addf(node, path, ReasonBadFormat, "must be letters and digits only")

		return ""
	}

	return label
}

func decodeClockZone(entries map[string]*yaml.Node, clockNode *yaml.Node, clockPath string, list *problemList) string {
	zone, node, found := requiredText(entries, clockNode, clockPath, clockZoneName, list)
	if !found {
		return ""
	}

	path := clockPath + "." + clockZoneName

	if length := utf8.RuneCountInString(zone); length < 1 || length > MaximumZoneCharacters {
		list.addf(node, path, ReasonOutOfBounds, "must be between 1 and %d characters", MaximumZoneCharacters)

		return ""
	}

	_, loadError := time.LoadLocation(zone)
	if loadError != nil || zone == localZone {
		list.addf(node, path, ReasonUnknownZone, "not a zone in the zone database, such as Europe/Berlin")

		return ""
	}

	return zone
}
