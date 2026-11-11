package repositorychecks_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/repositorychecks"
)

func goSourceUsingRuntimeGOOS() []byte {
	return []byte("package live\n\nimport " + plantedRuntimeImport +
		"\n\nfunc IsMac() bool {\n\treturn runtime.GOOS == \"darwin\"\n}\n")
}

func TestScanOperatingSystemUse_ReportsRuntimeGOOS_WhenInLivePackage(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanOperatingSystemUse("internal/live/hub.go", goSourceUsingRuntimeGOOS())

	if !containsRule(violations, repositorychecks.RuleOperatingSystemCheck) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleOperatingSystemCheck, rulesOf(violations))
	}
}

func TestScanOperatingSystemUse_ReportsRuntimeGOOS_WhenRuntimeImportIsRenamed(t *testing.T) {
	t.Parallel()

	source := "package live\n\nimport goruntime " + plantedRuntimeImport + "\n\nvar operatingSystem = goruntime.GOOS\n"

	violations := repositorychecks.ScanOperatingSystemUse("internal/live/hub.go", []byte(source))

	if !containsRule(violations, repositorychecks.RuleOperatingSystemCheck) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleOperatingSystemCheck, rulesOf(violations))
	}
}

func TestScanOperatingSystemUse_AllowsRuntimeGOOS_WhenInSystemAdapter(t *testing.T) {
	t.Parallel()

	const relativePath = "internal/platform/systemadapter/opener.go"

	violations := repositorychecks.ScanOperatingSystemUse(relativePath, goSourceUsingRuntimeGOOS())

	if len(violations) != 0 {
		t.Fatalf("want no violations in a permitted package, got %v", violations)
	}
}

func TestScanOperatingSystemUse_AllowsRuntimeGOOS_WhenInHIDTransport(t *testing.T) {
	t.Parallel()

	const relativePath = "internal/deck/hidtransport/transport.go"

	violations := repositorychecks.ScanOperatingSystemUse(relativePath, goSourceUsingRuntimeGOOS())

	if len(violations) != 0 {
		t.Fatalf("want no violations in a permitted package, got %v", violations)
	}
}

func TestScanOperatingSystemUse_ReportsRuntimeGOOS_WhenDirectoryOnlySharesPermittedPrefix(t *testing.T) {
	t.Parallel()

	const relativePath = "internal/platform/systemadapterextra/opener.go"

	violations := repositorychecks.ScanOperatingSystemUse(relativePath, goSourceUsingRuntimeGOOS())

	if !containsRule(violations, repositorychecks.RuleOperatingSystemCheck) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleOperatingSystemCheck, rulesOf(violations))
	}
}

func TestScanOperatingSystemUse_AllowsStringThatMentionsRuntimeGOOS(t *testing.T) {
	t.Parallel()

	source := "package live\n\nconst explanation = \"never call runtime.GOOS here\"\n"

	violations := repositorychecks.ScanOperatingSystemUse("internal/live/hub.go", []byte(source))

	if len(violations) != 0 {
		t.Fatalf("want no violations for a string literal, got %v", violations)
	}
}

func TestScanOperatingSystemUse_ReportsImport_WhenOutsidePermittedPackages(t *testing.T) {
	t.Parallel()

	forbiddenImportPaths := []string{"syscall", "C", "golang.org/x/sys/unix", "golang.org/x/sys/windows", "os/exec"}

	for _, forbiddenImportPath := range forbiddenImportPaths {
		t.Run(forbiddenImportPath, func(t *testing.T) {
			t.Parallel()

			source := "package live\n\nimport _ \"" + forbiddenImportPath + "\"\n"

			violations := repositorychecks.ScanOperatingSystemUse("internal/live/hub.go", []byte(source))

			if !containsRule(violations, repositorychecks.RuleOperatingSystemImport) {
				t.Fatalf("want a %s violation for %s, got %v",
					repositorychecks.RuleOperatingSystemImport, forbiddenImportPath, rulesOf(violations))
			}
		})
	}
}

func TestScanOperatingSystemUse_ReportsFileName_WhenSuffixNamesAnOperatingSystem(t *testing.T) {
	t.Parallel()

	operatingSystemFileNames := []string{
		"internal/live/clock_darwin.go",
		"internal/live/clock_windows_test.go",
		"internal/live/clock_linux_arm64.go",
	}

	for _, operatingSystemFileName := range operatingSystemFileNames {
		t.Run(operatingSystemFileName, func(t *testing.T) {
			t.Parallel()

			violations := repositorychecks.ScanOperatingSystemUse(operatingSystemFileName, []byte("package live\n"))

			if !containsRule(violations, repositorychecks.RuleOperatingSystemFile) {
				t.Fatalf("want a %s violation, got %v", repositorychecks.RuleOperatingSystemFile, rulesOf(violations))
			}
		})
	}
}

func TestScanOperatingSystemUse_AllowsFileName_WhenSuffixNamesOnlyAnArchitecture(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanOperatingSystemUse("internal/live/clock_amd64.go", []byte("package live\n"))

	if len(violations) != 0 {
		t.Fatalf("want no violations for an architecture suffix, got %v", violations)
	}
}

func TestScanOperatingSystemUse_ReportsBuildConstraint_WhenItNamesAnOperatingSystem(t *testing.T) {
	t.Parallel()

	constraints := []string{"darwin", "!windows", "unix", "linux && amd64"}

	for _, constraint := range constraints {
		t.Run(constraint, func(t *testing.T) {
			t.Parallel()

			source := "//go:build " + constraint + "\n\npackage live\n"

			violations := repositorychecks.ScanOperatingSystemUse("internal/live/hub.go", []byte(source))

			if !containsRule(violations, repositorychecks.RuleOperatingSystemFile) {
				t.Fatalf("want a %s violation for %q, got %v",
					repositorychecks.RuleOperatingSystemFile, constraint, rulesOf(violations))
			}
		})
	}
}

func TestScanOperatingSystemUse_AllowsBuildConstraint_WhenItNamesNoOperatingSystem(t *testing.T) {
	t.Parallel()

	source := "//go:build integration\n\npackage live\n"

	violations := repositorychecks.ScanOperatingSystemUse("internal/live/hub.go", []byte(source))

	if len(violations) != 0 {
		t.Fatalf("want no violations for a non operating system tag, got %v", violations)
	}
}

func TestScanOperatingSystemUse_ReportsUnparseableFile(t *testing.T) {
	t.Parallel()

	violations := repositorychecks.ScanOperatingSystemUse("internal/live/hub.go", []byte("package live\n\nfunc {\n"))

	if !containsRule(violations, repositorychecks.RuleUnparseableGoFile) {
		t.Fatalf("want a %s violation, got %v", repositorychecks.RuleUnparseableGoFile, rulesOf(violations))
	}
}
