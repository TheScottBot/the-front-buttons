package systemadapter

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The app picker the author asked for lists the installed applications by
// name, each with the identifier this system's launcher takes (evaluation, deck
// testing). The parts that read and interpret what the system says live here,
// built everywhere so both are tested on any machine; only the commands that
// ask the system live in the per system files.

// applicationBundleSuffix ends a macOS application bundle's folder name.
const applicationBundleSuffix = ".app"

// windowsAppsFolderTarget opens a Start menu entry by its AppID, desktop and
// packaged applications alike; the launcher passes a target holding a
// backslash to ShellExecute unchanged (DV3).
const windowsAppsFolderTarget = `shell:AppsFolder\`

// BundleInfo is what an application bundle's Info.plist says about it.
type BundleInfo struct {
	Identifier string
	Name       string
}

// ScanApplications finds the application bundles in each root and in the
// folders directly inside it (such as /Applications/Utilities), never inside
// a bundle, and names each by its Info.plist: its bundle identifier, or its
// name when it has none, which the macOS launcher then opens by name. A bundle
// whose Info.plist cannot be read is left out; a root that does not exist is
// skipped.
func ScanApplications(roots []string, readInfo func(bundlePath string) (BundleInfo, error)) []platform.Application {
	var applications []platform.Application

	for _, root := range roots {
		for _, bundlePath := range bundlesIn(root) {
			info, readError := readInfo(bundlePath)
			if readError != nil {
				continue
			}

			name := cmp.Or(info.Name, strings.TrimSuffix(filepath.Base(bundlePath), applicationBundleSuffix))
			applications = append(applications, platform.Application{Name: name, Identifier: cmp.Or(info.Identifier, name)})
		}
	}

	return sortedUnique(applications)
}

// bundlesIn lists the bundles in a folder and one level down.
func bundlesIn(root string) []string {
	var bundles []string

	entries, listError := os.ReadDir(root)
	if listError != nil {
		return nil
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())

		switch {
		case !isFolder(entry, path):
		case strings.HasSuffix(entry.Name(), applicationBundleSuffix):
			bundles = append(bundles, path)
		default:
			bundles = append(bundles, bundlesDirectlyIn(path)...)
		}
	}

	return bundles
}

// isFolder reports whether an entry is a folder, following a symbolic link:
// macOS links /Applications/Safari.app to where it really keeps Safari, which
// the first scan of the author's Mac left out (evaluation, deck testing).
func isFolder(entry fs.DirEntry, path string) bool {
	if entry.Type()&fs.ModeSymlink == 0 {
		return entry.IsDir()
	}

	info, statError := os.Stat(path)

	return statError == nil && info.IsDir()
}

func bundlesDirectlyIn(folder string) []string {
	matches, globError := filepath.Glob(filepath.Join(folder, "*"+applicationBundleSuffix))
	if globError != nil {
		return nil
	}

	return slices.DeleteFunc(matches, func(path string) bool {
		info, statError := os.Stat(path)

		return statError != nil || !info.IsDir()
	})
}

// startApp is one entry of Get-StartApps, as ConvertTo-Json writes it.
//
//nolint:tagliatelle // the names are PowerShell's, not this project's
type startApp struct {
	Name  string `json:"Name"`
	AppID string `json:"AppID"`
}

// errNotStartApps reports output that is not Get-StartApps' JSON.
var errNotStartApps = errors.New("not the list of Start menu applications")

// ParseStartApps reads Get-StartApps | ConvertTo-Json, which writes one object
// rather than an array when there is a single entry. Each becomes a
// shell:AppsFolder target; blank entries, duplicates and anything the launch
// target rule refuses are left out.
func ParseStartApps(output []byte) ([]platform.Application, error) {
	trimmed := []byte(strings.TrimSpace(string(output)))

	var entries []startApp

	if len(trimmed) > 0 && trimmed[0] == '{' {
		trimmed = append(append([]byte{'['}, trimmed...), ']')
	}

	decodeError := json.Unmarshal(trimmed, &entries)
	if decodeError != nil {
		return nil, fmt.Errorf("%w: %w", errNotStartApps, decodeError)
	}

	applications := make([]platform.Application, 0, len(entries))

	for _, entry := range entries {
		if entry.Name == "" || entry.AppID == "" || platform.CheckLaunchTarget(entry.AppID) != nil {
			continue
		}

		applications = append(applications, platform.Application{
			Name: entry.Name, Identifier: windowsAppsFolderTarget + entry.AppID,
		})
	}

	return sortedUnique(applications), nil
}

func sortedUnique(applications []platform.Application) []platform.Application {
	slices.SortFunc(applications, func(first platform.Application, second platform.Application) int {
		return cmp.Or(
			cmp.Compare(strings.ToLower(first.Name), strings.ToLower(second.Name)),
			cmp.Compare(first.Identifier, second.Identifier))
	})

	return slices.Compact(applications)
}

// errNoApplications reports a listing that found nothing at all, which on a
// working system means the listing itself failed.
var errNoApplications = errors.New("no applications found")

// nonEmpty turns an empty listing into an error, so the editor keeps any
// earlier list rather than offering none.
func nonEmpty(applications []platform.Application) ([]platform.Application, error) {
	if len(applications) == 0 {
		return nil, errNoApplications
	}

	return applications, nil
}

// ApplicationLister is the ApplicationCatalogue adapter: it asks this system
// for its applications through the function the per system file supplies.
type ApplicationLister struct {
	list func(listContext context.Context) ([]platform.Application, error)
}

// Applications lists the installed applications, or fails if none are found.
func (lister ApplicationLister) Applications(listContext context.Context) ([]platform.Application, error) {
	return lister.list(listContext)
}
