package testsupport

import (
	"context"
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// launchContractCase builds one launch port for the contract: the function that
// opens or launches a target, and a function making its next request fail at the
// operating system level.
type launchContractCase struct {
	launch   func(launchContext context.Context, target string) error
	failNext func()
}

// RunURLOpenerContract is the one contract suite for the URLOpener port, run by
// every adapter (with a process double, so nothing real opens) and by the test
// double. newOpener returns a fresh opener and its failure switch.
func RunURLOpenerContract(t *testing.T, newOpener func(t *testing.T) (platform.URLOpener, func())) {
	t.Helper()

	runLaunchContract(t, "https://discord.com/channels/1/2", func(t *testing.T) launchContractCase {
		t.Helper()

		opener, failNext := newOpener(t)

		return launchContractCase{launch: opener.OpenURL, failNext: failNext}
	})
}

// RunApplicationLauncherContract is the one contract suite for the
// ApplicationLauncher port.
func RunApplicationLauncherContract(
	t *testing.T,
	identifier string,
	newLauncher func(t *testing.T) (platform.ApplicationLauncher, func()),
) {
	t.Helper()

	runLaunchContract(t, identifier, func(t *testing.T) launchContractCase {
		t.Helper()

		launcher, failNext := newLauncher(t)

		return launchContractCase{launch: launcher.LaunchApplication, failNext: failNext}
	})
}

func runLaunchContract(t *testing.T, validTarget string, newCase func(t *testing.T) launchContractCase) {
	t.Helper()

	t.Run("launches a valid target", func(t *testing.T) {
		launchError := newCase(t).launch(t.Context(), validTarget)
		if launchError != nil {
			t.Fatalf("want %q launched, got %v", validTarget, launchError)
		}
	})

	t.Run("refuses a target a command could misread, before running anything", func(t *testing.T) {
		for _, refused := range []string{"", "-a Calculator", "https://example.test/\nrm"} {
			launchError := newCase(t).launch(t.Context(), refused)
			if !errors.Is(launchError, platform.ErrLaunchRefused) {
				t.Errorf("want %q refused with ErrLaunchRefused, got %v", refused, launchError)
			}
		}
	})

	t.Run("reports an operating system failure as ErrLaunchFailed", func(t *testing.T) {
		contractCase := newCase(t)
		contractCase.failNext()

		launchError := contractCase.launch(t.Context(), validTarget)
		if !errors.Is(launchError, platform.ErrLaunchFailed) {
			t.Fatalf("want ErrLaunchFailed, got %v", launchError)
		}
	})
}
