package repositorychecks

import (
	"fmt"
	"strings"
)

// SettledDescription is the product description the author settled in
// specification section 1. It is the one place the line is written in code, and
// the README's first line must equal it byte for byte.
const SettledDescription = "This is for the lions living in the wiry frames of my peripherals' bodies"

// readmeRelativePath is where the README lives, relative to the repository root.
const readmeRelativePath = "README.md"

// CheckReadmeFirstLine reports a README whose first line is not exactly the
// settled description. A trailing carriage return counts as a difference, so a
// file saved with Windows line endings fails here rather than on someone else's
// machine.
func CheckReadmeFirstLine(readmeContent []byte) []Violation {
	firstLine, _, _ := strings.Cut(string(readmeContent), "\n")

	if firstLine == SettledDescription {
		return nil
	}

	return []Violation{{
		Path:       readmeRelativePath,
		LineNumber: 1,
		Rule:       RuleReadmeFirstLine,
		Detail:     fmt.Sprintf("first line must be exactly %q, found %q", SettledDescription, firstLine),
	}}
}
