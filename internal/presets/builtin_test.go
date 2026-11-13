package presets_test

import (
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/allbricks"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openurl"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
)

func builtInTable(t *testing.T) presets.Table {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := allbricks.Register(registry)
	if registerError != nil {
		t.Fatalf("registering every kind: %v", registerError)
	}

	table, tableError := presets.BuiltIn(registry)
	if tableError != nil {
		t.Fatalf("building the built in presets: %v", tableError)
	}

	return table
}

// presetExpansionCase is one built in preset, its fields and the URL it must
// produce.
type presetExpansionCase struct {
	presetName string
	fields     map[string]any
	wantURL    string
}

// builtInPresetCases holds the formats from section 6.6 of the evaluation
// record: Microsoft's Teams deep link page, Discord's developer documentation,
// and D&D Beyond's own site.
func builtInPresetCases() []presetExpansionCase {
	return []presetExpansionCase{
		{
			"discord_channel",
			map[string]any{"server_id": "123", "channel_id": "456"},
			"discord://discord.com/channels/123/456",
		},
		{
			"teams_chat",
			map[string]any{"participants": "a@example.test,b@example.test"},
			"https://teams.microsoft.com/l/chat/0/0?users=a%40example.test%2Cb%40example.test",
		},
		{
			"teams_chat_by_id",
			map[string]any{"chat_id": "19:abc@thread.v2"},
			"https://teams.microsoft.com/l/chat/19%3Aabc%40thread.v2/conversations",
		},
		{
			"teams_channel", teamsChannelFields(),
			"https://teams.microsoft.com/l/channel/19%3A9be3%40thread.tacv2/General%20Chat?groupId=g-1&tenantId=t-1",
		},
		{
			"teams_private_channel", teamsChannelFields(),
			"https://teams.microsoft.com/l/channel/19%3A9be3%40thread.tacv2/General%20Chat?groupId=g-1&tenantId=t-1&ngc=true",
		},
		{
			"teams_shared_channel", teamsChannelFields(),
			"https://teams.microsoft.com/l/channel/19%3A9be3%40thread.tacv2/General%20Chat?groupId=g-1&tenantId=t-1" +
				"&ngc=true&allowXTenantAccess=true",
		},
		{
			"dndbeyond_character",
			map[string]any{"character_id": "133058471"},
			"https://www.dndbeyond.com/characters/133058471",
		},
		{
			"roll20_campaign",
			map[string]any{"campaign_id": "19929142"},
			"https://app.roll20.net/editor/setcampaign/19929142",
		},
	}
}

func TestBuiltIn_ExpandsEveryPresetIntoTheDocumentedURL(t *testing.T) {
	t.Parallel()

	table := builtInTable(t)

	for _, presetCase := range builtInPresetCases() {
		row, found := table.Row(presetCase.presetName)
		if !found {
			t.Errorf("want preset %s", presetCase.presetName)

			continue
		}

		expanded, expandError := table.Expand(row, assignFields(t, row, presetCase.fields))
		if expandError != nil {
			t.Errorf("%s: %v", presetCase.presetName, expandError)

			continue
		}

		parameters, isOpenURL := expanded.(*openurl.Parameters)
		if !isOpenURL || parameters.URL != presetCase.wantURL {
			t.Errorf("%s:\nwant %s\ngot  %+v", presetCase.presetName, presetCase.wantURL, expanded)
		}
	}
}

// channelIDField is the channel ID field shared by the Teams channel presets.
const channelIDField = "channel_id"

func teamsChannelFields() map[string]any {
	return map[string]any{
		channelIDField: "19:9be3@thread.tacv2", "channel_name": "General Chat", "group_id": "g-1", "tenant_id": "t-1",
	}
}

func TestBuiltIn_HasNoRoll20Preset_UntilItsLaunchURLIsKnown(t *testing.T) {
	t.Parallel()

	// TD13: Roll20 documents no launch URL, so the preset waits for the author
	// to copy one real Launch Game link.
	for _, presetName := range builtInTable(t).Names() {
		if presetName == "roll20_game" {
			t.Fatal("want no Roll20 preset yet")
		}
	}
}

func TestBuiltIn_DeclaresEachPresetsAction(t *testing.T) {
	t.Parallel()

	// The editor shows a preset's action rather than its target kind, which
	// for every built in preset is open_url and tells the author nothing.
	const openChannel = "open_channel"

	wantActions := map[string]string{
		"discord_channel":       "open_server",
		"dndbeyond_character":   "open_character",
		"roll20_campaign":       "open_campaign",
		"teams_chat":            "open_chat",
		"teams_chat_by_id":      "open_chat",
		"teams_channel":         openChannel,
		"teams_private_channel": openChannel,
		"teams_shared_channel":  openChannel,
	}

	table := builtInTable(t)

	if len(table.Names()) != len(wantActions) {
		t.Fatalf("want an action for each of %d presets, have %v", len(wantActions), table.Names())
	}

	for presetName, wantAction := range wantActions {
		row, found := table.Row(presetName)
		if !found {
			t.Errorf("want preset %s", presetName)

			continue
		}

		if row.Action() != wantAction {
			t.Errorf("%s: want action %q, got %q", presetName, wantAction, row.Action())
		}
	}
}
