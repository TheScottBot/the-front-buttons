package systemadapter_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// The names several tests share, and the reader's failure for a bundle it
// cannot read.
const (
	spotifyName    = "Spotify"
	calculatorName = "Calculator"
)

var errUnreadableInfo = errors.New("unreadable Info.plist")

// TestParseStartApps_NamesEveryStartMenuApplication reads Get-StartApps as
// PowerShell's ConvertTo-Json writes it. Every entry becomes a shell:AppsFolder
// target, which opens a desktop or a packaged application alike through the
// launcher's existing path rule (DV3). Duplicates, blank entries and anything
// the launch target rule refuses are dropped; the rest are sorted by name.
func TestParseStartApps_NamesEveryStartMenuApplication(t *testing.T) {
	t.Parallel()

	output := []byte(`[
		{"Name":"Spotify","AppID":"SpotifyAB.SpotifyMusic_zpdnekdrzrea0!Spotify"},
		{"Name":"Notepad","AppID":"{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\\notepad.exe"},
		{"Name":"Notepad","AppID":"{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\\notepad.exe"},
		{"Name":"","AppID":"Blank.Name"},
		{"Name":"Sneaky","AppID":"-flag"}
	]`)

	applications, parseError := systemadapter.ParseStartApps(output)
	if parseError != nil {
		t.Fatal(parseError)
	}

	want := []platform.Application{
		{Name: "Notepad", Identifier: `shell:AppsFolder\{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\notepad.exe`},
		{Name: spotifyName, Identifier: `shell:AppsFolder\SpotifyAB.SpotifyMusic_zpdnekdrzrea0!Spotify`},
	}
	if !reflect.DeepEqual(applications, want) {
		t.Fatalf("want %+v, got %+v", want, applications)
	}
}

// TestParseStartApps_ReadsASingleApplication: ConvertTo-Json writes one object,
// not an array, when there is only one.
func TestParseStartApps_ReadsASingleApplication(t *testing.T) {
	t.Parallel()

	applications, parseError := systemadapter.ParseStartApps([]byte(`{"Name":"Calculator","AppID":"Calc!App"}`))
	if parseError != nil || len(applications) != 1 || applications[0].Name != calculatorName {
		t.Fatalf("want Calculator, got %+v (%v)", applications, parseError)
	}
}

func TestParseStartApps_RefusesWhatIsNotItsOutput(t *testing.T) {
	t.Parallel()

	_, parseError := systemadapter.ParseStartApps([]byte("Get-StartApps : not recognised"))
	if parseError == nil {
		t.Fatal("want an error for output that is not JSON")
	}
}

// makeApp makes an empty application bundle with an Info.plist placeholder.
func makeApp(t *testing.T, path string) {
	t.Helper()

	makeError := os.MkdirAll(filepath.Join(path, "Contents"), 0o700)
	if makeError != nil {
		t.Fatal(makeError)
	}

	writeError := os.WriteFile(filepath.Join(path, "Contents", "Info.plist"), []byte("plist"), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}
}

// TestScanApplications_FindsBundlesInTheFoldersAndOneLevelDown covers the
// macOS catalogue's search, given a reader for each bundle's Info.plist: the
// applications in each folder and in its subfolders, such as Utilities, never
// the applications inside an application, named by their display name and
// identified by their bundle identifier, or by their name when they have none.
func TestScanApplications_FindsBundlesInTheFoldersAndOneLevelDown(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	makeApp(t, filepath.Join(root, "Spotify.app"))
	makeApp(t, filepath.Join(root, "Utilities", "Terminal.app"))
	makeApp(t, filepath.Join(root, "Spotify.app", "Contents", "Helpers", "Helper.app"))
	makeApp(t, filepath.Join(root, "Plain.app"))
	makeApp(t, filepath.Join(root, "Broken.app"))

	infos := map[string]systemadapter.BundleInfo{
		"Spotify.app":  {Identifier: "com.spotify.client", Name: spotifyName},
		"Terminal.app": {Identifier: "com.apple.Terminal", Name: "Terminal"},
		"Plain.app":    {Identifier: "", Name: ""},
	}

	readInfo := func(bundlePath string) (systemadapter.BundleInfo, error) {
		info, known := infos[filepath.Base(bundlePath)]
		if !known {
			return systemadapter.BundleInfo{Identifier: "", Name: ""}, errUnreadableInfo
		}

		return info, nil
	}

	applications := systemadapter.ScanApplications([]string{root, filepath.Join(root, "missing")}, readInfo)

	want := []platform.Application{
		{Name: "Plain", Identifier: "Plain"},
		{Name: spotifyName, Identifier: "com.spotify.client"},
		{Name: "Terminal", Identifier: "com.apple.Terminal"},
	}
	if !reflect.DeepEqual(applications, want) {
		t.Fatalf("want %+v, got %+v", want, applications)
	}
}

// TestScanApplications_FollowsALinkedBundle: macOS keeps Safari elsewhere and
// links /Applications/Safari.app to it, which the first scan of the author's
// Mac left out (evaluation, deck testing).
func TestScanApplications_FollowsALinkedBundle(t *testing.T) {
	t.Parallel()

	root, elsewhere := t.TempDir(), t.TempDir()
	makeApp(t, filepath.Join(elsewhere, "Safari.app"))

	linkError := os.Symlink(filepath.Join(elsewhere, "Safari.app"), filepath.Join(root, "Safari.app"))
	if linkError != nil {
		t.Skipf("this system will not make a symbolic link here: %v", linkError)
	}

	applications := systemadapter.ScanApplications([]string{root}, func(string) (systemadapter.BundleInfo, error) {
		return systemadapter.BundleInfo{Identifier: "com.apple.Safari", Name: "Safari"}, nil
	})

	if len(applications) != 1 || applications[0].Identifier != "com.apple.Safari" {
		t.Fatalf("want the linked Safari found, got %+v", applications)
	}
}
