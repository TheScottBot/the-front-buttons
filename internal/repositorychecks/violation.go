package repositorychecks

import "fmt"

// RuleName identifies which repository rule a violation breaks. The values are
// stable so continuous integration output can be searched by rule.
type RuleName string

// The rules this package enforces, each traceable to the document that requires
// it.
const (
	// RuleDashCharacter is engineering datum 0.8: no em dash or en dash.
	RuleDashCharacter RuleName = "dash-character"
	// RuleHyphenRun is engineering datum 0.8: no converter mangled double or
	// triple hyphen in prose.
	RuleHyphenRun RuleName = "hyphen-run"
	// RuleOperatingSystemImport is specification section 2: operating system
	// imports only in the permitted packages.
	RuleOperatingSystemImport RuleName = "operating-system-import"
	// RuleOperatingSystemCheck is specification section 2: no runtime.GOOS outside
	// the permitted packages.
	RuleOperatingSystemCheck RuleName = "operating-system-check"
	// RuleOperatingSystemFile is specification section 2: build tagged operating
	// system files only in the permitted packages.
	RuleOperatingSystemFile RuleName = "operating-system-file"
	// RuleReadmeFirstLine is specification section 1: the README's first line is
	// the settled description.
	RuleReadmeFirstLine RuleName = "readme-first-line"
	// RuleUnsupportedIgnorePattern reports a .gitignore line the scan does not
	// understand. Refusing it is safer than guessing, because a guessed pattern
	// would scan files git ignores or skip files git tracks.
	RuleUnsupportedIgnorePattern RuleName = "unsupported-ignore-pattern"
	// RuleUnparseableGoFile reports a Go file the operating system check could
	// not parse, so a syntax error can never hide a violation.
	RuleUnparseableGoFile RuleName = "unparseable-go-file"
)

// Violation is one breach of a repository rule at a place in the tree.
type Violation struct {
	Path       string
	LineNumber int
	Rule       RuleName
	Detail     string
}

// String renders the violation in the path:line form editors and continuous
// integration logs make clickable.
func (violation Violation) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", violation.Path, violation.LineNumber, violation.Rule, violation.Detail)
}
