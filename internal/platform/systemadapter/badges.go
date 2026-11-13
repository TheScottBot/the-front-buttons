package systemadapter

import (
	"strings"
	"unicode"
)

// The separators the Dock reader puts between an identifier and its label,
// and between one application and the next: control characters, which
// neither an identifier nor a label shown on a Dock icon holds.
const (
	badgeFieldSeparator  = "\x1f"
	badgeRecordSeparator = "\x1e"
)

// MaximumBadgeCharacters bounds a badge's label: a Dock badge shows a few
// characters, such as "99+", and nothing longer fits on a key.
const MaximumBadgeCharacters = 8

// ParseBadges reads the Dock reader's records into labels by identifier, in
// lower case. A record without both parts, or whose label is only space and
// control characters, is skipped; a label is cut to MaximumBadgeCharacters.
func ParseBadges(records string) map[string]string {
	badges := map[string]string{}

	for record := range strings.SplitSeq(records, badgeRecordSeparator) {
		identifier, label, found := strings.Cut(record, badgeFieldSeparator)
		label = cleanBadge(label)

		if found && identifier != "" && label != "" {
			badges[strings.ToLower(identifier)] = label
		}
	}

	return badges
}

// cleanBadge drops control characters and surrounding space, and cuts the
// label to its bound.
func cleanBadge(label string) string {
	cleaned := []rune(strings.TrimSpace(strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return -1
		}

		return character
	}, label)))

	if len(cleaned) > MaximumBadgeCharacters {
		cleaned = cleaned[:MaximumBadgeCharacters]
	}

	return string(cleaned)
}
