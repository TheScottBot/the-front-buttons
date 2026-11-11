package repositorychecks_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/repositorychecks"
)

func TestScanProse_ReportsEnDash_WhenMarkdownContainsOne(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanProse("docs/notes.md", []byte("one "+plantedEnDash+" two\n"))

	if !containsRule(violations, repositorychecks.RuleDashCharacter) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleDashCharacter, rulesOf(violations))
	}
}

func TestScanProse_ReportsEmDash_WhenMarkdownContainsOne(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanProse("docs/notes.md", []byte("one"+plantedEmDash+"two\n"))

	if !containsRule(violations, repositorychecks.RuleDashCharacter) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleDashCharacter, rulesOf(violations))
	}
}

func TestScanProse_ReportsHyphenRun_WhenProseContainsSpacedDoubleHyphen(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanProse("README.md", []byte("one "+plantedDoubleHyphen+" two\n"))

	if !containsRule(violations, repositorychecks.RuleHyphenRun) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleHyphenRun, rulesOf(violations))
	}
}

func TestScanProse_ReportsHyphenRun_WhenProseContainsUnspacedDoubleHyphen(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanProse("README.md", []byte("one"+plantedDoubleHyphen+"two\n"))

	if !containsRule(violations, repositorychecks.RuleHyphenRun) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleHyphenRun, rulesOf(violations))
	}
}

func TestScanProse_ReportsHyphenRun_WhenLineIsTripleHyphenRule(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanProse("README.md", []byte("above\n"+plantedTripleHyphen+"\nbelow\n"))

	if !containsRule(violations, repositorychecks.RuleHyphenRun) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleHyphenRun, rulesOf(violations))
	}
}

func TestScanProse_AllowsMarkdownTableSeparatorRow(t *testing.T) {
	t.Parallel()

	table := "| Name | Value |\n|" + plantedTripleHyphen + "|:" + plantedTripleHyphen + ":|\n| a | b |\n"

	violations := repositorychecks.ScanProse("README.md", []byte(table))

	if len(violations) != 0 {
		t.Fatalf("want no violations for a table separator row, got %v", violations)
	}
}

func TestScanProse_AllowsCommandLineFlagToken(t *testing.T) {
	t.Parallel()

	command := "Run `go mod download " + plantedDoubleHyphen + "json` and " + plantedDoubleHyphen + "depth=1.\n"

	violations := repositorychecks.ScanProse("TESTING.md", []byte(command))

	if len(violations) != 0 {
		t.Fatalf("want no violations for command line flag tokens, got %v", violations)
	}
}

func TestScanProse_ReportsLineNumber_WhenViolationIsOnLaterLine(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanProse("docs/notes.md", []byte("first\nsecond\nthird "+plantedEnDash+"\n"))

	if len(violations) != 1 || violations[0].LineNumber != 3 {
		t.Fatalf("want one violation on line 3, got %v", violations)
	}
}

func TestScanProse_ReportsHyphenRun_WhenGoCommentContainsOne(t *testing.T) {
	t.Parallel()

	source := "package sample\n\n// one " + plantedDoubleHyphen + " two\nfunc Sample() {}\n"

	violations := repositorychecks.ScanProse("internal/sample/sample.go", []byte(source))

	if !containsRule(violations, repositorychecks.RuleHyphenRun) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleHyphenRun, rulesOf(violations))
	}
}

func TestScanProse_ReportsHyphenRun_WhenGoStringLiteralContainsOne(t *testing.T) {
	t.Parallel()

	source := "package sample\n\nconst label = \"one " + plantedDoubleHyphen + " two\"\n"

	violations := repositorychecks.ScanProse("internal/sample/sample.go", []byte(source))

	if !containsRule(violations, repositorychecks.RuleHyphenRun) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleHyphenRun, rulesOf(violations))
	}
}

func TestScanProse_AllowsDecrementOperator_WhenGoCodeUsesOne(t *testing.T) {
	t.Parallel()

	source := "package sample\n\nfunc Sample(remaining int) int {\n\tremaining" + plantedDoubleHyphen +
		"\n\n\treturn remaining\n}\n"

	violations := repositorychecks.ScanProse("internal/sample/sample.go", []byte(source))

	if len(violations) != 0 {
		t.Fatalf("want no violations for a decrement operator, got %v", violations)
	}
}

func TestScanProse_ReportsDash_WhenGoCodeContainsOneOutsideComments(t *testing.T) {
	t.Parallel()

	source := "package sample\n\nconst label = \"a" + plantedEmDash + "b\"\n"

	violations := repositorychecks.ScanProse("internal/sample/sample.go", []byte(source))

	if !containsRule(violations, repositorychecks.RuleDashCharacter) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleDashCharacter, rulesOf(violations))
	}
}

func TestScanProse_AllowsCleanProse(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanProse("README.md", []byte("A well behaved sentence, with a hyphenated-word.\n"))

	if len(violations) != 0 {
		t.Fatalf("want no violations, got %v", violations)
	}
}
