package presets

import (
	"fmt"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/openurl"
)

// The built in presets' friendly fields. Formats are from the evaluation record,
// section 6.6, as TD13 settled: Teams over https, Discord over https, D&D Beyond
// as observed on its site, and no Roll20 preset until its launch URL is known.

// DiscordChannelFields are the IDs Discord's Developer Mode copies.
type DiscordChannelFields struct {
	brick.Appearance

	ServerID  string `brick:"server_id,required"  bounds:"1,32" format:"digits" description:"The server ID, from Copy Server ID with Developer Mode on."`
	ChannelID string `brick:"channel_id,required" bounds:"1,32" format:"digits" description:"The channel ID, from Copy Channel ID with Developer Mode on."`
}

// TeamsChatFields start or open a chat with given people.
type TeamsChatFields struct {
	brick.Appearance

	Participants string `brick:"participants,required" bounds:"3,2048" description:"Comma separated user principal names, usually email addresses."`
}

// TeamsChatByIDFields open an existing chat.
type TeamsChatByIDFields struct {
	brick.Appearance

	ChatID string `brick:"chat_id,required" bounds:"4,256" description:"The chat ID beginning 19:, from a message's Copy link."`
}

// TeamsChannelFields open a channel; the three channel presets share them.
type TeamsChannelFields struct {
	brick.Appearance

	ChannelID   string `brick:"channel_id,required"   bounds:"4,256" description:"The channel ID beginning 19:, from the channel's URL."`
	ChannelName string `brick:"channel_name,required" bounds:"1,256" description:"The channel's name."`
	GroupID     string `brick:"group_id,required"     bounds:"1,64"  format:"identifier"                                                 description:"The team's group ID, from the channel's URL."`
	TenantID    string `brick:"tenant_id,required"    bounds:"1,64"  format:"identifier"                                                 description:"The organisation's tenant ID, from the channel's URL."`
}

// DnDBeyondCharacterFields open a character sheet.
type DnDBeyondCharacterFields struct {
	brick.Appearance

	CharacterID string `brick:"character_id,required" bounds:"1,32" format:"digits" description:"The number at the end of the character sheet's address."`
}

// teamsChannelBase is the documented channel link; private and shared channels
// add their documented parameters to it.
const teamsChannelBase = "https://teams.microsoft.com/l/channel/{{uri_component .channel_id}}/" +
	"{{uri_component .channel_name}}?groupId={{query .group_id}}&tenantId={{query .tenant_id}}"

// BuiltIn returns the preset table every machine gets.
func BuiltIn(registry *brick.Registry) (Table, error) {
	urlTemplate := func(templateText string) map[string]string {
		return map[string]string{"url": templateText}
	}

	rowErrors := make([]error, 0)
	rows := make([]Row, 0)

	addRow := func(row Row, defineError error) {
		rowErrors = append(rowErrors, defineError)
		rows = append(rows, row)
	}

	addRow(Define[DiscordChannelFields]("discord_channel", openurl.KindName, urlTemplate(
		"https://discord.com/channels/{{path_segment .server_id}}/{{path_segment .channel_id}}")))
	addRow(Define[TeamsChatFields]("teams_chat", openurl.KindName, urlTemplate(
		"https://teams.microsoft.com/l/chat/0/0?users={{query .participants}}")))
	addRow(Define[TeamsChatByIDFields]("teams_chat_by_id", openurl.KindName, urlTemplate(
		"https://teams.microsoft.com/l/chat/{{uri_component .chat_id}}/conversations")))
	addRow(Define[TeamsChannelFields]("teams_channel", openurl.KindName, urlTemplate(teamsChannelBase)))
	addRow(Define[TeamsChannelFields]("teams_private_channel", openurl.KindName, urlTemplate(
		teamsChannelBase+"&ngc=true")))
	addRow(Define[TeamsChannelFields]("teams_shared_channel", openurl.KindName, urlTemplate(
		teamsChannelBase+"&ngc=true&allowXTenantAccess=true")))
	addRow(Define[DnDBeyondCharacterFields]("dndbeyond_character", openurl.KindName, urlTemplate(
		"https://www.dndbeyond.com/characters/{{path_segment .character_id}}")))

	for _, rowError := range rowErrors {
		if rowError != nil {
			return Table{}, fmt.Errorf("defining the built in presets: %w", rowError)
		}
	}

	return NewTable(registry, rows...)
}
