package repositorychecks_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/repositorychecks"
)

func TestCheckReadmeFirstLine_Accepts_WhenFirstLineIsTheDescription(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.CheckReadmeFirstLine([]byte(repositorychecks.SettledDescription + "\n\nMore text.\n"))

	if len(violations) != 0 {
		t.Fatalf("want no violations, got %v", violations)
	}
}

func TestCheckReadmeFirstLine_Reports_WhenFirstLineIsChanged(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.CheckReadmeFirstLine([]byte(repositorychecks.SettledDescription + " and more\n"))

	if !containsRule(violations, repositorychecks.RuleReadmeFirstLine) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleReadmeFirstLine, rulesOf(violations))
	}
}

func TestCheckReadmeFirstLine_Reports_WhenFirstLineEndsWithCarriageReturn(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.CheckReadmeFirstLine([]byte(repositorychecks.SettledDescription + "\r\n"))

	if !containsRule(violations, repositorychecks.RuleReadmeFirstLine) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleReadmeFirstLine, rulesOf(violations))
	}
}

func TestCheckReadmeFirstLine_Reports_WhenDescriptionIsOnlyOnSecondLine(t *testing.T) {
	t.Parallel()

	readmeContent := "# The Front Buttons\n" + repositorychecks.SettledDescription + "\n"

	violations := repositorychecks.CheckReadmeFirstLine([]byte(readmeContent))

	if !containsRule(violations, repositorychecks.RuleReadmeFirstLine) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleReadmeFirstLine, rulesOf(violations))
	}
}

func TestCheckReadmeFirstLine_Reports_WhenReadmeIsEmpty(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.CheckReadmeFirstLine(nil)

	if !containsRule(violations, repositorychecks.RuleReadmeFirstLine) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleReadmeFirstLine, rulesOf(violations))
	}
}
