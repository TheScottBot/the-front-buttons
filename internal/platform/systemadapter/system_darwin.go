package systemadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The folders macOS keeps applications in: the shared ones and the user's own.
const (
	macOSApplications       = "/Applications"
	macOSSystemApplications = "/System/Applications"
	macOSPlutilProgram      = "/usr/bin/plutil"
)

// NewURLOpener returns this machine's URLOpener: /usr/bin/open.
func NewURLOpener() MacOSLauncher { return NewMacOSURLOpener(runCommand) }

// NewApplicationLauncher returns this machine's ApplicationLauncher:
// /usr/bin/open.
func NewApplicationLauncher() MacOSLauncher { return NewMacOSApplicationLauncher(runCommand) }

// runCommand runs a program directly, never through a shell. Its output is
// discarded unread, because open's messages can echo the URL, which logs must
// never hold (specification section 7).
func runCommand(runContext context.Context, program string, arguments ...string) error {
	// The program is the fixed open command from launch.go, and the arguments
	// go straight to it with no shell, so no argument is parsed as syntax.
	runError := exec.CommandContext(runContext, program, arguments...).Run() //nolint:gosec // see above
	if runError != nil {
		return fmt.Errorf("running %s: %w", program, runError)
	}

	return nil
}

// NewApplicationCatalogue lists the application bundles in /Applications,
// /System/Applications and the user's own Applications folder, reading each
// one's Info.plist with plutil, which reads both its XML and binary forms.
func NewApplicationCatalogue() ApplicationLister {
	return ApplicationLister{list: func(listContext context.Context) ([]platform.Application, error) {
		roots := []string{macOSApplications, macOSSystemApplications}

		home, homeError := os.UserHomeDir()
		if homeError == nil {
			roots = append(roots, filepath.Join(home, "Applications"))
		}

		return nonEmpty(ScanApplications(roots, func(bundlePath string) (BundleInfo, error) {
			return readBundleInfo(listContext, bundlePath)
		}))
	}}
}

// bundleInfoKeys are the Info.plist entries the picker reads.
//
//nolint:tagliatelle // the names are Apple's, not this project's
type bundleInfoKeys struct {
	Identifier  string `json:"CFBundleIdentifier"`
	DisplayName string `json:"CFBundleDisplayName"`
	Name        string `json:"CFBundleName"`
}

// errNoInfoPlist reports a bundle without an Info.plist, which is not an
// application the launcher can open.
var errNoInfoPlist = errors.New("no Info.plist")

func readBundleInfo(readContext context.Context, bundlePath string) (BundleInfo, error) {
	plistPath := filepath.Join(bundlePath, "Contents", "Info.plist")

	_, statError := os.Stat(plistPath)
	if errors.Is(statError, fs.ErrNotExist) {
		return BundleInfo{Identifier: "", Name: ""}, errNoInfoPlist
	}

	// The program is the fixed plutil path and the plist path is one argument,
	// run with no shell, so no part of a bundle's name is parsed as syntax.
	output, runError := exec.CommandContext(readContext, macOSPlutilProgram, //nolint:gosec // see above
		"-convert", "json", "-o", "-", plistPath).Output()
	if runError != nil {
		return BundleInfo{Identifier: "", Name: ""}, fmt.Errorf("reading %s: %w", plistPath, runError)
	}

	var keys bundleInfoKeys

	decodeError := json.Unmarshal(output, &keys)
	if decodeError != nil {
		return BundleInfo{Identifier: "", Name: ""}, fmt.Errorf("reading %s: %w", plistPath, decodeError)
	}

	name := keys.DisplayName
	if name == "" {
		name = keys.Name
	}

	return BundleInfo{Identifier: keys.Identifier, Name: name}, nil
}
