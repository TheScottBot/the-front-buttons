package testsupport

import (
	"fmt"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
)

// The fake kind and preset names, written once for every test that uses them.
const (
	LinkKindName       = "test_link"
	PollerKindName     = "test_poller"
	ChannelPresetName  = "test_channel"
	SegmentPresetName  = "test_segment"
	LinkAddressField   = "address"
	PollerSourceField  = "source"
	PollerEveryField   = "every"
	PollerEnabledField = "enabled"
	PollerRetriesField = "retries"
)

// LinkParameters is a fake kind shaped like open_url: one required URL.
type LinkParameters struct {
	brick.Appearance

	Address string `brick:"address,required" bounds:"1,2048" format:"https_url" description:"Where the fake link goes."`
}

// PollerParameters is a fake kind shaped like a live brick, covering every
// field type and a secret credential reference.
type PollerParameters struct {
	brick.Appearance

	Source     string        `brick:"source,required"   bounds:"1,100"                              format:"identifier"                                   description:"What the fake poller watches."`
	Every      time.Duration `brick:"every"             bounds:"30s,1h"                             description:"How often the fake poller polls."`
	Enabled    bool          `brick:"enabled"           description:"Whether the fake poller runs."`
	Retries    int           `brick:"retries"           bounds:"0,5"                                description:"How many times the fake poller retries."`
	Credential string        `brick:"credential,secret" bounds:"1,64"                               format:"identifier"                                   description:"Name of the fake poller's credential."`
}

// ChannelFields is a fake preset's friendly fields, shaped like the Discord
// channel preset: two IDs of digits.
type ChannelFields struct {
	brick.Appearance

	ServerID  string `brick:"server_id,required"  bounds:"1,32" format:"digits" description:"Fake server ID."`
	ChannelID string `brick:"channel_id,required" bounds:"1,32" format:"digits" description:"Fake channel ID."`
}

// SegmentFields is a fake preset whose one field is free text, so tests can
// prove a template escapes what it places in a URL.
type SegmentFields struct {
	brick.Appearance

	Name string `brick:"name,required" bounds:"1,64" description:"Free text placed in a URL path."`
}

// LinkTestDouble is what the fake link kind constructs.
type LinkTestDouble struct {
	Parameters LinkParameters
}

// PollerTestDouble is what the fake poller kind constructs.
type PollerTestDouble struct {
	Parameters PollerParameters
}

// NewRegistry returns a registry holding the fake kinds.
func NewRegistry() (*brick.Registry, error) {
	registry := brick.NewRegistry()

	linkError := brick.Register(registry, LinkKindName, func(parameters LinkParameters, _ brick.Ports) (any, error) {
		return &LinkTestDouble{Parameters: parameters}, nil
	})
	if linkError != nil {
		return nil, fmt.Errorf("registering the fake link kind: %w", linkError)
	}

	pollerError := brick.Register(registry, PollerKindName, func(parameters PollerParameters, _ brick.Ports) (any, error) {
		return &PollerTestDouble{Parameters: parameters}, nil
	})
	if pollerError != nil {
		return nil, fmt.Errorf("registering the fake poller kind: %w", pollerError)
	}

	return registry, nil
}

// ChannelTemplates expands the fake channel preset into the fake link kind.
func ChannelTemplates() map[string]string {
	return map[string]string{
		LinkAddressField: "https://chat.example.test/channels/{{path_segment .server_id}}/{{path_segment .channel_id}}",
	}
}

// SegmentTemplates expands the fake segment preset into the fake link kind.
func SegmentTemplates() map[string]string {
	return map[string]string{
		LinkAddressField: "https://chat.example.test/rooms/{{path_segment .name}}",
	}
}

// NewPresetTable returns a table of the fake presets over the given registry.
func NewPresetTable(registry *brick.Registry) (presets.Table, error) {
	channelRow, channelError := presets.Define[ChannelFields](ChannelPresetName, LinkKindName, ChannelTemplates())
	if channelError != nil {
		return presets.Table{}, channelError
	}

	segmentRow, segmentError := presets.Define[SegmentFields](SegmentPresetName, LinkKindName, SegmentTemplates())
	if segmentError != nil {
		return presets.Table{}, segmentError
	}

	table, tableError := presets.NewTable(registry, channelRow, segmentRow)
	if tableError != nil {
		return presets.Table{}, fmt.Errorf("building the fake preset table: %w", tableError)
	}

	return table, nil
}
