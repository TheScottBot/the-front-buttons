package platform_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The property list's two boolean elements, which carry their value in their
// name.
const (
	plistTrue  = "true"
	plistFalse = "false"
)

// The depth of the top level dictionary's own keys and values: plist, dict, then
// the element.
const topLevelDepth = 3

// plistReader reads a property list's top level dictionary into each key's
// value, flattened: a nested dictionary's keys and values run together, so
// KeepAlive with SuccessfulExit false reads "SuccessfulExitfalse".
type plistReader struct {
	entries    map[string]string
	depth      int
	currentKey string
	inKey      bool
}

func (reader *plistReader) read(token xml.Token) {
	switch typed := token.(type) {
	case xml.StartElement:
		reader.depth++
		reader.inKey = reader.depth == topLevelDepth && typed.Name.Local == "key"

		if typed.Name.Local == plistTrue || typed.Name.Local == plistFalse {
			reader.add(typed.Name.Local)
		}
	case xml.EndElement:
		reader.depth--
		reader.inKey = false
	case xml.CharData:
		reader.text(strings.TrimSpace(string(typed)))
	}
}

func (reader *plistReader) text(text string) {
	if text == "" {
		return
	}

	if reader.inKey {
		reader.currentKey = text

		return
	}

	reader.add(text)
}

func (reader *plistReader) add(text string) {
	if reader.depth >= topLevelDepth && reader.currentKey != "" {
		reader.entries[reader.currentKey] += text
	}
}

func plistEntries(t *testing.T, plist string) map[string]string {
	t.Helper()

	decoder := xml.NewDecoder(strings.NewReader(plist))
	decoder.Strict = true

	reader := &plistReader{entries: map[string]string{}, depth: 0, currentKey: "", inKey: false}

	for {
		token, tokenError := decoder.Token()
		if tokenError != nil {
			return reader.entries
		}

		reader.read(token)
	}
}

func TestLaunchAgentPlist_StartsTheDaemonAtLoginAndRestartsItOnlyAfterACrash(t *testing.T) {
	t.Parallel()

	plist := platform.LaunchAgentPlist("/Applications/The Front Buttons/the-front-buttons", "/tmp/state/launchd.log")
	entries := plistEntries(t, plist)

	want := map[string]string{
		"Label":             platform.LaunchAgentLabel,
		"ProgramArguments":  "/Applications/The Front Buttons/the-front-buttons",
		"RunAtLoad":         plistTrue,
		"KeepAlive":         "SuccessfulExitfalse",
		"ProcessType":       "Interactive",
		"StandardOutPath":   "/tmp/state/launchd.log",
		"StandardErrorPath": "/tmp/state/launchd.log",
	}

	for key, wantValue := range want {
		if entries[key] != wantValue {
			t.Errorf("%s: want %q, got %q", key, wantValue, entries[key])
		}
	}

	if !strings.HasPrefix(plist, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("want an XML declaration first")
	}
}

func TestLaunchAgentPlist_EscapesAPathThatLooksLikeMarkup(t *testing.T) {
	t.Parallel()

	plist := platform.LaunchAgentPlist("/Users/a&b/<bin>/the-front-buttons", "/tmp/log")

	if got := plistEntries(t, plist)["ProgramArguments"]; got != "/Users/a&b/<bin>/the-front-buttons" {
		t.Fatalf("want the path back intact, got %q", got)
	}
}

// taskDefinition is the part of a Task Scheduler definition the test reads.
type taskDefinition struct {
	XMLName  xml.Name `xml:"http://schemas.microsoft.com/windows/2004/02/mit/task Task"`
	Version  string   `xml:"version,attr"`
	Triggers struct {
		Logon struct {
			Enabled string `xml:"Enabled"`
			UserID  string `xml:"UserId"`
		} `xml:"LogonTrigger"`
	} `xml:"Triggers"`
	Principal struct {
		ID        string `xml:"id,attr"`
		UserID    string `xml:"UserId"`
		LogonType string `xml:"LogonType"`
		RunLevel  string `xml:"RunLevel"`
	} `xml:"Principals>Principal"`
	Settings struct {
		MultipleInstancesPolicy    string `xml:"MultipleInstancesPolicy"`
		DisallowStartIfOnBatteries string `xml:"DisallowStartIfOnBatteries"`
		StopIfGoingOnBatteries     string `xml:"StopIfGoingOnBatteries"`
		ExecutionTimeLimit         string `xml:"ExecutionTimeLimit"`
	} `xml:"Settings"`
	Actions struct {
		Context string `xml:"Context,attr"`
		Command string `xml:"Exec>Command"`
	} `xml:"Actions"`
}

func TestScheduledTaskXML_StartsTheDaemonAtThisUsersLogon_WithoutElevation(t *testing.T) {
	t.Parallel()

	definition := platform.ScheduledTaskXML(`C:\Users\a&b\the-front-buttons.exe`, `WORK\person`)

	var decoded taskDefinition

	decodeError := xml.Unmarshal([]byte(definition), &decoded)
	if decodeError != nil {
		t.Fatalf("want well formed XML, got %v", decodeError)
	}

	checks := map[string][2]string{
		"version":            {decoded.Version, "1.3"},
		"trigger enabled":    {decoded.Triggers.Logon.Enabled, "true"},
		"trigger user":       {decoded.Triggers.Logon.UserID, `WORK\person`},
		"principal user":     {decoded.Principal.UserID, `WORK\person`},
		"logon type":         {decoded.Principal.LogonType, "InteractiveToken"},
		"run level":          {decoded.Principal.RunLevel, "LeastPrivilege"},
		"one instance":       {decoded.Settings.MultipleInstancesPolicy, "IgnoreNew"},
		"runs on battery":    {decoded.Settings.DisallowStartIfOnBatteries, "false"},
		"keeps on battery":   {decoded.Settings.StopIfGoingOnBatteries, "false"},
		"no time limit":      {decoded.Settings.ExecutionTimeLimit, "PT0S"},
		"action context":     {decoded.Actions.Context, decoded.Principal.ID},
		"command, unescaped": {decoded.Actions.Command, `C:\Users\a&b\the-front-buttons.exe`},
	}

	for checkName, gotAndWant := range checks {
		if gotAndWant[0] != gotAndWant[1] || gotAndWant[1] == "" {
			t.Errorf("%s: want %q, got %q", checkName, gotAndWant[1], gotAndWant[0])
		}
	}

	if strings.Index(definition, "<Enabled>") > strings.Index(definition, "<UserId>") {
		t.Error("the logon trigger's Enabled must come before its UserId, which the schema orders")
	}
}
