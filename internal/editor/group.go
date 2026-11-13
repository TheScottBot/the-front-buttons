package editor

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
)

// A group is several preset keys placed down one column at once, from the
// bricks drawer's 1x2 and 1x3 drawers: the author's D&D session, a D&D Beyond
// sheet and a Roll20 campaign, and a Discord channel for the 1x3, set up in one
// form whose parts can be put in any order (evaluation, deck testing). Each
// part becomes an ordinary preset key, re-edited like any other.

// The parts' names in the form.
const (
	characterPart = "character"
	roll20Part    = "roll20"
	discordPart   = "discord"
)

// groupPart is one key a group places: its name in the form, the preset it
// becomes, what the form calls it, and the label and icon it starts with.
type groupPart struct {
	name   string
	preset string
	title  string
	label  string
	icon   string
}

// The reasons a group is refused. Each is a whole sentence shown after
// "Error: "; nothing is placed unless every key can be.
var (
	errGroupOrder    = errors.New("the order must name each of the group's keys once; reload the form and try again")
	errGroupNoPage   = errors.New("that page no longer exists; the pages may have changed since this one was opened")
	errGroupOffGrid  = errors.New("the group runs off the bottom of the deck")
	errGroupReserved = errors.New("the group would cover a tile kept for paging")
	errGroupOccupied = errors.New("the group would cover a key")
)

// The group form's own field names.
const (
	groupOrderField = "order"
	groupSizeField  = "size"
)

// groupParts are the keys a group can place, by name.
func groupParts() map[string]groupPart {
	return map[string]groupPart{
		characterPart: {
			name: characterPart, preset: "dndbeyond_character", title: "D&D Beyond character sheet",
			label: "Sheet", icon: "wand",
		},
		roll20Part: {
			name: roll20Part, preset: "roll20_campaign", title: "Roll20 campaign",
			label: "Roll20", icon: "dice-5",
		},
		discordPart: {
			name: discordPart, preset: "discord_channel", title: "Discord channel",
			label: "Discord", icon: "brand-discord",
		},
	}
}

// groupSizes are the groups the drawer offers, each its parts in their first
// order.
func groupSizes() map[string][]string {
	return map[string][]string{
		"1x2": {characterPart, roll20Part},
		"1x3": {characterPart, roll20Part, discordPart},
	}
}

// groupPartView is one part of the form, numbered by its place down the
// column.
type groupPartView struct {
	Name   string
	Title  string
	Slot   int
	Fields []fieldView
}

// groupView is the group form.
type groupView struct {
	baseView

	PageIndex     int
	Row           int
	Column        int
	DisplayRow    int
	DisplayColumn int
	Size          string
	Parts         []groupPartView
	Errors        []string
}

// groupAt is where a group goes and what it is.
type groupAt struct {
	pageIndex int
	top       layout.Position
	size      string
}

func groupAtFrom(values url.Values) groupAt {
	return groupAt{
		pageIndex: integerParameter(values, "page"),
		top:       layout.Position{Row: integerParameter(values, "row"), Column: integerParameter(values, "column")},
		size:      values.Get(groupSizeField),
	}
}

// showGroup is a new group's form, its parts in their first order.
func (pages *pageHandlers) showGroup(writer http.ResponseWriter, request *http.Request) {
	placement := groupAtFrom(request.URL.Query())

	order, known := groupSizes()[placement.size]
	if !known {
		http.Error(writer, "unknown group size", http.StatusBadRequest)

		return
	}

	pages.renderGroup(writer, request, http.StatusOK, placement, order, nil, nil)
}

// saveGroup places every part of a group, or none: each tile down the column
// must be on the deck, free and not kept for paging.
func (pages *pageHandlers) saveGroup(writer http.ResponseWriter, request *http.Request) {
	form := request.PostForm
	placement := groupAtFrom(form)

	defaultOrder, known := groupSizes()[placement.size]
	if !known {
		http.Error(writer, "unknown group size", http.StatusBadRequest)

		return
	}

	order := form[groupOrderField]
	if !isOrderOf(order, defaultOrder) {
		pages.renderGroup(writer, request, http.StatusUnprocessableEntity, placement, defaultOrder, form,
			[]string{errorPrefix + errGroupOrder.Error()})

		return
	}

	keys, fieldErrors := pages.groupKeys(placement, order, form)
	if len(fieldErrors) > 0 {
		pages.renderGroupWithFieldErrors(writer, request, placement, order, form, fieldErrors)

		return
	}

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if placement.pageIndex < 0 || placement.pageIndex >= len(configuration.Pages) {
			return errGroupNoPage
		}

		placeError := checkGroupTiles(configuration.Pages[placement.pageIndex], pages.options.Grid, keys)
		if placeError != nil {
			return placeError
		}

		page := &configuration.Pages[placement.pageIndex]
		page.Keys = append(page.Keys, keys...)

		return nil
	})
	if saveError != nil {
		pages.renderGroup(writer, request, http.StatusUnprocessableEntity, placement, order, form, groupProblems(saveError))

		return
	}

	redirectToSavedPage(writer, request, placement.pageIndex)
}

// isOrderOf reports whether an order names each of a group's parts once.
func isOrderOf(order []string, parts []string) bool {
	return len(order) == len(parts) &&
		slices.Equal(slices.Sorted(slices.Values(order)), slices.Sorted(slices.Values(parts)))
}

// groupKeys builds each part's key at its tile down the column, or the field
// errors of each part, keyed by the part's form name for the field.
func (pages *pageHandlers) groupKeys(
	placement groupAt,
	order []string,
	form url.Values,
) ([]config.Key, map[string]string) {
	keys := make([]config.Key, 0, len(order))
	allErrors := map[string]string{}

	for index, partName := range order {
		part := groupParts()[partName]

		resolved, typeError := pages.resolveType(presetChoicePrefix + part.preset)
		if typeError != nil {
			allErrors[partName] = errorPrefix + typeError.Error()

			continue
		}

		fields, fieldErrors := parseFields(resolved.schema, partForm(partName, form))
		for fieldName, message := range fieldErrors {
			allErrors[partName+"."+fieldPrefix+fieldName] = message
		}

		if len(fieldErrors) > 0 {
			continue
		}

		position := layout.Position{Row: placement.top.Row + index, Column: placement.top.Column}

		key, buildError := pages.buildKey(resolved, position, fields)
		if buildError != nil {
			allErrors[partName] = errorPrefix + buildError.Error()

			continue
		}

		keys = append(keys, key)
	}

	return keys, allErrors
}

// partForm is one part's fields as a key form names them.
func partForm(partName string, form url.Values) url.Values {
	values := url.Values{}

	for name, value := range form {
		if field, isPart := cutPartPrefix(partName, name); isPart {
			values[field] = value
		}
	}

	return values
}

func cutPartPrefix(partName string, name string) (string, bool) {
	prefix := partName + "."
	if len(name) <= len(prefix) || name[:len(prefix)] != prefix {
		return "", false
	}

	return name[len(prefix):], true
}

// checkGroupTiles refuses a group whose keys would leave the deck, cover a
// paging tile or cover a key, naming the first such tile.
func checkGroupTiles(page config.Page, grid layout.Grid, keys []config.Key) error {
	for _, key := range keys {
		switch {
		case !grid.Contains(key.Position):
			return errGroupOffGrid
		case grid.IsReserved(key.Position):
			return fmt.Errorf("%w at %s", errGroupReserved, tileWords(key.Position))
		}

		if _, taken := keyAt(page, key.Position); taken {
			return fmt.Errorf("%w at %s; choose a column with enough free tiles",
				errGroupOccupied, tileWords(key.Position))
		}
	}

	return nil
}

// groupProblems explains a refused group in its own words, or as a failed
// save.
func groupProblems(saveError error) []string {
	for _, known := range []error{errGroupNoPage, errGroupOffGrid, errGroupReserved, errGroupOccupied} {
		if errors.Is(saveError, known) {
			return []string{errorPrefix + saveError.Error()}
		}
	}

	return saveProblems(saveError)
}

func (pages *pageHandlers) renderGroupWithFieldErrors(
	writer http.ResponseWriter,
	request *http.Request,
	placement groupAt,
	order []string,
	form url.Values,
	fieldErrors map[string]string,
) {
	general := []string{errorPrefix + "some details are missing or wrong; each is marked below."}

	for _, partName := range order {
		if message, partFailed := fieldErrors[partName]; partFailed {
			general = append(general, message)
		}
	}

	pages.renderGroupForm(writer, request, http.StatusUnprocessableEntity, placement, order, form, general, fieldErrors)
}

func (pages *pageHandlers) renderGroup(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	placement groupAt,
	order []string,
	form url.Values,
	general []string,
) {
	pages.renderGroupForm(writer, request, status, placement, order, form, general, nil)
}

// renderGroupForm shows the form: each part's label, icon and its preset's
// own details, named for the part, filled from a refused form or with the
// part's first label and icon.
func (pages *pageHandlers) renderGroupForm(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	placement groupAt,
	order []string,
	form url.Values,
	general []string,
	fieldErrors map[string]string,
) {
	_, problems := pages.configuration()

	view := groupView{
		baseView:      pages.base("Keys down a column", request, problems),
		PageIndex:     placement.pageIndex,
		Row:           placement.top.Row,
		Column:        placement.top.Column,
		DisplayRow:    placement.top.Row + 1,
		DisplayColumn: placement.top.Column + 1,
		Size:          placement.size,
		Parts:         make([]groupPartView, 0, len(order)),
		Errors:        general,
	}

	for index, partName := range order {
		view.Parts = append(view.Parts, pages.partView(groupParts()[partName], index+1, form, fieldErrors))
	}

	pages.render(writer, status, "group.html", view)
}

// partView is one part's fields: its label and icon, then its preset's own.
func (pages *pageHandlers) partView(
	part groupPart,
	slot int,
	form url.Values,
	fieldErrors map[string]string,
) groupPartView {
	resolved, _ := pages.resolveType(presetChoicePrefix + part.preset)
	view := groupPartView{Name: part.name, Title: part.title, Slot: slot, Fields: nil}

	for _, field := range fieldViews(resolved.schema, nil, nil) {
		fieldName := field.Name[len(fieldPrefix):]
		if fieldName == "background" || fieldName == "icon_file" {
			continue
		}

		formName := part.name + "." + field.Name
		field.ID = part.name + "-" + field.ID
		field.DescriptionID = part.name + "-" + field.DescriptionID
		field.ErrorID = part.name + "-" + field.ErrorID
		field.Name = formName
		field.Error = fieldErrors[formName]

		switch {
		case form != nil:
			field.Value = form.Get(formName)
		case fieldName == "label":
			field.Value = part.label
		case fieldName == "icon":
			field.Value = part.icon
		}

		view.Fields = append(view.Fields, field)
	}

	return view
}
