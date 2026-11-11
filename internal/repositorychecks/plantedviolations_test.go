package repositorychecks_test

import "github.com/TheScottBot/the-front-buttons/internal/repositorychecks"

// The planted violations are built from code points and concatenation rather
// than written literally, because a literal dash or hyphen run in this file would
// itself break the prose scan these tests prove.
const (
	plantedEnDash        = string(rune(0x2013))
	plantedEmDash        = string(rune(0x2014))
	plantedDoubleHyphen  = "-" + "-"
	plantedTripleHyphen  = "-" + "-" + "-"
	plantedRuntimeImport = "\"runtime\""
)

// rulesOf lists the rules of the given violations in order, so a test failure
// shows which rules fired rather than only how many.
func rulesOf(violations []repositorychecks.Violation) []repositorychecks.RuleName {
	ruleNames := make([]repositorychecks.RuleName, 0, len(violations))
	for _, violation := range violations {
		ruleNames = append(ruleNames, violation.Rule)
	}

	return ruleNames
}

// containsRule reports whether any violation breaks the given rule.
func containsRule(violations []repositorychecks.Violation, wantedRule repositorychecks.RuleName) bool {
	for _, violation := range violations {
		if violation.Rule == wantedRule {
			return true
		}
	}

	return false
}
