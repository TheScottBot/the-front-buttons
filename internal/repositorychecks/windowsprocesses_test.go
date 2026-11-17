package repositorychecks_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/repositorychecks"
)

const (
	plantedExecImport = "\"os/exec\""
	plantedOSImport   = "\"os\""
)

func windowsSourceStarting(call string) []byte {
	return []byte("package systemadapter\n\nimport (\n\t" + plantedExecImport + "\n\t" + plantedOSImport +
		"\n)\n\nvar _ = os.Args\n\nfunc start() {\n\t_ = " + call + "\n}\n")
}

func TestScanWindowsProcessStarts_ReportsEachWayOfStartingAProgram(t *testing.T) {
	t.Parallel()

	for _, call := range []string{
		`exec.Command("powershell.exe")`,
		`exec.CommandContext(nil, "powershell.exe")`,
		`func() error { _, e := os.StartProcess("cmd.exe", nil, nil); return e }()`,
	} {
		violations := repositorychecks.ScanWindowsProcessStarts(
			"internal/platform/systemadapter/quit_windows.go", windowsSourceStarting(call))

		if !containsRule(violations, repositorychecks.RuleWindowsVisibleProcess) {
			t.Errorf("want a %s violation for %s, got %v",
				repositorychecks.RuleWindowsVisibleProcess, call, rulesOf(violations))
		}
	}
}

func TestScanWindowsProcessStarts_ReportsARenamedExecImport(t *testing.T) {
	t.Parallel()

	source := "package systemadapter\n\nimport run " + plantedExecImport +
		"\n\nvar _ = run.Command(\"powershell.exe\")\n"

	violations := repositorychecks.ScanWindowsProcessStarts(
		"internal/platform/systemadapter/system_windows.go", []byte(source))

	if !containsRule(violations, repositorychecks.RuleWindowsVisibleProcess) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleWindowsVisibleProcess, rulesOf(violations))
	}
}

func TestScanWindowsProcessStarts_ReportsAWindowsBuildConstraint(t *testing.T) {
	t.Parallel()

	source := "//go:build windows\n\n" + string(windowsSourceStarting(`exec.Command("powershell.exe")`))

	violations := repositorychecks.ScanWindowsProcessStarts("internal/platform/systemadapter/launcher.go", []byte(source))

	if !containsRule(violations, repositorychecks.RuleWindowsVisibleProcess) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleWindowsVisibleProcess, rulesOf(violations))
	}
}

func TestScanWindowsProcessStarts_AllowsTheHiddenProgramHelper(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanWindowsProcessStarts(
		"internal/platform/systemadapter/hiddenprogram_windows.go", windowsSourceStarting(`exec.Command("x")`))

	if len(violations) != 0 {
		t.Fatalf("want the helper allowed, got %v", violations)
	}
}

func TestScanWindowsProcessStarts_IgnoresTestsAndOtherSystems(t *testing.T) {
	t.Parallel()

	for _, relativePath := range []string{
		"internal/platform/systemadapter/quit_windows_test.go",
		"internal/platform/systemadapter/system_darwin.go",
		"internal/platform/systemadapter/launch.go",
	} {
		violations := repositorychecks.ScanWindowsProcessStarts(relativePath, windowsSourceStarting(`exec.Command("x")`))

		if len(violations) != 0 {
			t.Errorf("want %s ignored, got %v", relativePath, violations)
		}
	}
}
