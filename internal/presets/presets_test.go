package presets_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

func fakeRegistry(t *testing.T) *brick.Registry {
	t.Helper()

	registry, registryError := testsupport.NewRegistry()
	if registryError != nil {
		t.Fatalf("building the fake registry: %v", registryError)
	}

	return registry
}

func defineChannelRow(t *testing.T) presets.Row {
	t.Helper()

	row, defineError := presets.Define[testsupport.ChannelFields](
		testsupport.ChannelPresetName, testsupport.LinkKindName, testsupport.ChannelTemplates())
	if defineError != nil {
		t.Fatalf("defining the channel preset: %v", defineError)
	}

	return row
}

// assignFields fills a preset's friendly fields through its own schema, as the
// configuration decoder does.
func assignFields(t *testing.T, row presets.Row, values map[string]any) any {
	t.Helper()

	fields := row.FieldsSchema().NewParameters()
	for fieldName, value := range values {
		field, fieldFound := row.FieldsSchema().Field(fieldName)
		if !fieldFound {
			t.Fatalf("preset %s has no field %s", row.Name(), fieldName)
		}

		assignError := row.FieldsSchema().Assign(fields, field, value)
		if assignError != nil {
			t.Fatalf("assigning %s: %v", fieldName, assignError)
		}
	}

	return fields
}

func TestExpand_RendersTemplateIntoTargetParameters_AndPassesAppearanceThrough(t *testing.T) {
	t.Parallel()

	registry := fakeRegistry(t)
	row := defineChannelRow(t)

	table, tableError := presets.NewTable(registry, row)
	if tableError != nil {
		t.Fatalf("building the table: %v", tableError)
	}

	channelRow, _ := table.Row(testsupport.ChannelPresetName)
	fields := assignFields(t, channelRow, map[string]any{"server_id": "123", "channel_id": "456", "label": "Party"})

	expanded, expandError := table.Expand(channelRow, fields)
	if expandError != nil {
		t.Fatalf("expanding: %v", expandError)
	}

	link, isLink := expanded.(*testsupport.LinkParameters)
	if !isLink {
		t.Fatalf("want *LinkParameters, got %T", expanded)
	}

	if link.Address != "https://chat.example.test/channels/123/456" || link.Label != "Party" {
		t.Fatalf("expansion wrong: %+v", link)
	}
}

func TestExpand_EscapesFieldValuesPlacedInAPath(t *testing.T) {
	t.Parallel()

	registry := fakeRegistry(t)

	row, defineError := presets.Define[testsupport.SegmentFields](
		testsupport.SegmentPresetName, testsupport.LinkKindName, testsupport.SegmentTemplates())
	if defineError != nil {
		t.Fatalf("defining: %v", defineError)
	}

	table, tableError := presets.NewTable(registry, row)
	if tableError != nil {
		t.Fatalf("building the table: %v", tableError)
	}

	fields := assignFields(t, row, map[string]any{"name": "a/b ?c"})

	expanded, expandError := table.Expand(row, fields)
	if expandError != nil {
		t.Fatalf("expanding: %v", expandError)
	}

	link, _ := expanded.(*testsupport.LinkParameters)
	if link.Address != "https://chat.example.test/rooms/a%2Fb%20%3Fc" {
		t.Fatalf("want the segment escaped, got %q", link.Address)
	}
}

func TestDefine_RejectsTemplatesThatDoNotParse(t *testing.T) {
	t.Parallel()

	_, defineError := presets.Define[testsupport.ChannelFields](
		testsupport.ChannelPresetName, testsupport.LinkKindName,
		map[string]string{testsupport.LinkAddressField: "https://x.test/{{.server_id"})

	if !errors.Is(defineError, presets.ErrInvalidPreset) {
		t.Fatalf("want ErrInvalidPreset, got %v", defineError)
	}
}

func TestDefine_RejectsBadPresetName(t *testing.T) {
	t.Parallel()

	_, defineError := presets.Define[testsupport.ChannelFields](
		"Test Channel", testsupport.LinkKindName, testsupport.ChannelTemplates())

	if !errors.Is(defineError, presets.ErrInvalidPreset) {
		t.Fatalf("want ErrInvalidPreset, got %v", defineError)
	}
}

func TestNewTable_RejectsRowsThatCannotExpandIntoTheirTarget(t *testing.T) {
	t.Parallel()

	registry := fakeRegistry(t)

	rowCases := []struct {
		name       string
		targetKind string
		templates  map[string]string
	}{
		{"unknown target kind", "no_such_kind", testsupport.ChannelTemplates()},
		{"template for an unknown target field", testsupport.LinkKindName, map[string]string{
			testsupport.LinkAddressField: "https://x.test/", "nowhere": "x",
		}},
		{"template for an appearance field", testsupport.LinkKindName, map[string]string{
			testsupport.LinkAddressField: "https://x.test/", "label": "x",
		}},
		{"required target field left uncovered", testsupport.LinkKindName, map[string]string{}},
		{"template for a non text field", testsupport.PollerKindName, map[string]string{
			testsupport.PollerSourceField: "x", testsupport.PollerEveryField: "60s",
		}},
	}

	for _, rowCase := range rowCases {
		t.Run(rowCase.name, func(t *testing.T) {
			t.Parallel()

			row, defineError := presets.Define[testsupport.ChannelFields](
				testsupport.ChannelPresetName, rowCase.targetKind, rowCase.templates)
			if defineError != nil {
				t.Fatalf("defining: %v", defineError)
			}

			_, tableError := presets.NewTable(registry, row)
			if !errors.Is(tableError, presets.ErrInvalidPreset) {
				t.Fatalf("want ErrInvalidPreset, got %v", tableError)
			}
		})
	}
}

func TestNewTable_RejectsDuplicatePresetNames(t *testing.T) {
	t.Parallel()

	registry := fakeRegistry(t)
	row := defineChannelRow(t)

	_, tableError := presets.NewTable(registry, row, row)
	if !errors.Is(tableError, presets.ErrInvalidPreset) {
		t.Fatalf("want ErrInvalidPreset, got %v", tableError)
	}
}

func TestExpand_ReportsFieldError_WhenRenderedValueBreaksTargetRule(t *testing.T) {
	t.Parallel()

	registry := fakeRegistry(t)

	row, defineError := presets.Define[testsupport.SegmentFields](
		testsupport.SegmentPresetName, testsupport.LinkKindName,
		map[string]string{testsupport.LinkAddressField: "http://insecure.test/{{path_segment .name}}"})
	if defineError != nil {
		t.Fatalf("defining: %v", defineError)
	}

	table, tableError := presets.NewTable(registry, row)
	if tableError != nil {
		t.Fatalf("building the table: %v", tableError)
	}

	_, expandError := table.Expand(row, assignFields(t, row, map[string]any{"name": "x"}))

	fieldError, isFieldError := errors.AsType[*brick.FieldError](expandError)
	if !isFieldError || fieldError.Reason != brick.ReasonBadFormat {
		t.Fatalf("want a bad format FieldError, got %v", expandError)
	}
}

func TestTable_NamesAreSorted(t *testing.T) {
	t.Parallel()

	registry := fakeRegistry(t)

	segmentRow, defineError := presets.Define[testsupport.SegmentFields](
		testsupport.SegmentPresetName, testsupport.LinkKindName, testsupport.SegmentTemplates())
	if defineError != nil {
		t.Fatalf("defining: %v", defineError)
	}

	table, tableError := presets.NewTable(registry, segmentRow, defineChannelRow(t))
	if tableError != nil {
		t.Fatalf("building the table: %v", tableError)
	}

	names := table.Names()
	if len(names) != 2 || names[0] != testsupport.ChannelPresetName || names[1] != testsupport.SegmentPresetName {
		t.Fatalf("want sorted names, got %v", names)
	}
}
