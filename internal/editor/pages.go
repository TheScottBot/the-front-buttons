package editor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The messages announced in each page's live region after a change.
const (
	savedMessage      = "Saved. The deck shows the change within a second."
	credentialMessage = "Credential stored. It is never shown again." //nolint:gosec // a message, not a credential
)

// fileRefusedMessage explains why editing waits while the file on disk is
// invalid: saving over it would throw the hand edit away.
const fileRefusedMessage = errorPrefix +
	"the configuration file has problems, listed above. Fix the file, then edit here; " +
	"saving now would overwrite the hand edit."

// errFileRefused reports an edit refused because the file on disk is invalid.
var errFileRefused = errors.New("configuration file on disk is invalid")

// pageHandlers serves the editor's pages.
type pageHandlers struct {
	options   Options
	templates *template.Template
	icons     iconIndex
	apps      *applicationCache
	removals  *removalMemory
}

func newPageHandlers(options Options) *pageHandlers {
	// The templates are compiled into the binary and proven by the tests, so a
	// failure here is a broken build, found at start up rather than on a request.
	templates := template.Must(template.ParseFS(embeddedFiles, "templates/*.html"))

	return &pageHandlers{
		options: options, templates: templates, icons: newIconIndex(options.Icons),
		apps: newApplicationCache(options.Apps), removals: newRemovalMemory(),
	}
}

// baseView is what every page shows: the product name, the token for its forms,
// a message for the live region and any problems with the file on disk.
type baseView struct {
	Title    string
	Product  string
	Token    string
	Message  string
	Problems []string
	// Undo offers to put back the key just removed.
	Undo bool
	// Theme is the palette chosen with the switch, light or dark, or empty to
	// follow the system's setting.
	Theme string
}

// themeCookie is where the theme switch keeps its choice. It is read here so
// the page is drawn in the chosen palette from the start, rather than
// flashing the other until the script runs.
const themeCookie = "theme"

// chosenTheme is the theme the request's cookie names, if it is one there is.
func chosenTheme(request *http.Request) string {
	cookie, cookieError := request.Cookie(themeCookie)
	if cookieError != nil {
		return ""
	}

	switch cookie.Value {
	case "light", "dark":
		return cookie.Value
	default:
		return ""
	}
}

func (pages *pageHandlers) base(title string, request *http.Request, problems []string) baseView {
	message, undo := "", false

	switch request.URL.Query().Get("saved") {
	case "key":
		message = savedMessage
	case "removed":
		message, undo = removedMessage, pages.removals.holding()
	case "credential":
		message = credentialMessage
	case copiedQuery:
		message = copiedMessage
	}

	return baseView{
		Title:    title,
		Product:  platform.ProductDisplayName,
		Token:    pages.options.Token,
		Message:  message,
		Problems: problems,
		Undo:     undo,
		Theme:    chosenTheme(request),
	}
}

// configuration returns the file on disk and no problems, or, if the file is
// refused, the last good configuration and the file's problems.
func (pages *pageHandlers) configuration() (config.Configuration, []string) {
	onDisk, diskError := pages.options.Store.OnDisk()
	if diskError == nil {
		return onDisk, nil
	}

	var problems []string

	if rejection, isRejection := errors.AsType[*config.RejectionError](diskError); isRejection {
		for _, problem := range rejection.Problems {
			problems = append(problems, fmt.Sprintf("%s%s (line %d): %s: %s",
				errorPrefix, problem.Path, problem.Line, problem.Reason, problem.Detail))
		}
	} else {
		problems = append(problems, errorPrefix+"the configuration file could not be read")
	}

	lastGood, hasLastGood := pages.options.Store.Current()
	if !hasLastGood {
		lastGood = config.Configuration{SchemaVersion: config.CurrentSchemaVersion, Pages: nil}
	}

	return lastGood, problems
}

// editable returns the configuration to change, refusing while the file on disk
// is invalid.
func (pages *pageHandlers) editable() (config.Configuration, error) {
	configuration, problems := pages.configuration()
	if len(problems) > 0 {
		return configuration, errFileRefused
	}

	return configuration, nil
}

func (pages *pageHandlers) render(
	writer http.ResponseWriter,
	status int,
	templateName string,
	view any,
) {
	var rendered bytes.Buffer

	executeError := pages.templates.ExecuteTemplate(&rendered, templateName, view)
	if executeError != nil {
		pages.options.Logger.Error("editor template failed", "template", templateName)
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = writer.Write(rendered.Bytes())
}

func (pages *pageHandlers) redirectHome(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, "/page?page=0", http.StatusSeeOther)
}

// integerParameter reads a whole number from a query or form value.
func integerParameter(values url.Values, name string) int {
	value, parseError := strconv.Atoi(values.Get(name))
	if parseError != nil {
		return -1
	}

	return value
}

// cellView is one key in the grid. Every cell a person can act on is a link the
// keyboard reaches in order, named for a screen reader by its row, column and
// contents.
type cellView struct {
	Reserved bool
	Name     string
	Text     string
	Kind     string
	Href     string
	// Tile is the cell's page, row and column joined by dots, and State is
	// "key", "empty" or "reserved": what the drag script needs to know.
	Tile  string
	State string
	// Image is a picture the cell shows in place of its text: the
	// application's icon on an app screen's return tile.
	Image string
	// Selectable marks a key select mode offers to tick, Select is its tile
	// as the copy posts it, "row.column", SelectID its check box's id, and
	// Checked is ticked already.
	Selectable bool
	Select     string
	SelectID   string
	Checked    bool
}

// The states a grid cell can be in, for the drag script.
const (
	tileHoldsKey  = "key"
	tileIsEmpty   = "empty"
	tileIsReserve = "reserved"
)

type pageLink struct {
	Index   int
	Name    string
	Current bool
}

type pageView struct {
	baseView

	PageIndex  int
	PageNumber int
	PageCount  int
	PageName   string
	HasPage    bool
	Pages      []pageLink
	// AppScreens are listed apart, under the regular pages, unnumbered.
	AppScreens []pageLink
	// ScreenTitle names the application while an app screen is shown.
	ScreenTitle string
	Rows        [][]cellView
	// CopyTargets are the pages this one can be copied onto: every other.
	CopyTargets []copyTarget
	// Selecting is select mode, where each key is a check box for copying,
	// and SelectTargets are the pages its keys can be copied onto.
	Selecting     bool
	SelectTargets []selectTarget
	// Types are the bricks and presets the drawer offers to drag onto an
	// empty tile, the same list the key form chooses from.
	Types []typeOption
	// AppName, AppTypes and AppGroups are an app screen's own application's
	// bricks and groups, in a drawer of their own.
	AppName   string
	AppTypes  []typeOption
	AppGroups []appGroupView
}

func (pages *pageHandlers) showPage(writer http.ResponseWriter, request *http.Request) {
	pages.apps.warm(request.Context())

	configuration, problems := pages.configuration()
	query := request.URL.Query()
	pageIndex := integerParameter(query, "page")

	if query.Get(selectQuery) == "1" {
		selection := copySelection{source: pageIndex, tiles: nil, targets: nil}
		pages.renderPageSelecting(writer, request, http.StatusOK, configuration, selection, problems)

		return
	}

	pages.renderPage(writer, request, http.StatusOK, configuration, pageIndex, problems)
}

// renderPageSelecting draws a page in select mode, with the keys and pages of
// a selection ticked, so a refused copy keeps what was ticked.
func (pages *pageHandlers) renderPageSelecting(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	configuration config.Configuration,
	selection copySelection,
	problems []string,
) {
	view := pages.pageViewFor(request, configuration, selection.source, problems)

	if view.HasPage {
		view.Selecting = true
		view.SelectTargets = pages.selectTargets(configuration.Pages, selection)

		for _, row := range view.Rows {
			for column := range row {
				cell := &row[column]
				cell.Checked = cell.Selectable && slices.Contains(selection.tiles, tileFrom(cell.Select))
			}
		}
	}

	pages.render(writer, status, "page.html", view)
}

func (pages *pageHandlers) renderPage(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	configuration config.Configuration,
	pageIndex int,
	problems []string,
) {
	pages.render(writer, status, "page.html", pages.pageViewFor(request, configuration, pageIndex, problems))
}

// pageViewFor is everything the page of the grid shows for one page.
func (pages *pageHandlers) pageViewFor(
	request *http.Request,
	configuration config.Configuration,
	pageIndex int,
	problems []string,
) pageView {
	page := pageAt(configuration, pageIndex)
	regular := regularPages(configuration.Pages)
	appName := ""

	if set, hasSet := pages.appSetOf(page); hasSet {
		appName = set.NameFor(page.Application)
	}

	view := pageView{
		baseView:    pages.base("Pages", request, problems),
		PageIndex:   0,
		PageNumber:  0,
		PageCount:   len(regular),
		PageName:    "",
		HasPage:     false,
		Pages:       nil,
		AppScreens:  nil,
		ScreenTitle: "",
		Rows:        nil,

		CopyTargets: copyTargets(configuration, pageIndex),
		Types:       pages.typeOptions(""),
		AppName:     appName,
		AppTypes:    pages.appTypeOptions(page, ""),
		AppGroups:   pages.appGroupViews(page),

		Selecting:     false,
		SelectTargets: nil,
	}

	for index, listed := range configuration.Pages {
		link := pageLink{Index: index, Name: listed.Name, Current: index == pageIndex}
		if listed.IsAppScreen() {
			view.AppScreens = append(view.AppScreens, link)
		} else {
			view.Pages = append(view.Pages, link)
		}
	}

	if pageIndex >= 0 && pageIndex < len(configuration.Pages) {
		view.HasPage = true
		view.PageIndex = pageIndex
		view.PageNumber = slices.Index(regular, pageIndex) + 1
		view.PageName = page.Name
		view.Rows = pages.gridRows(configuration.Pages, pageIndex)

		if page.IsAppScreen() {
			view.ScreenTitle = pages.screenTitle(page)
		}
	}

	return view
}

func (pages *pageHandlers) gridRows(allPages []config.Page, pageIndex int) [][]cellView {
	grid := pages.options.Grid
	rows := make([][]cellView, 0, grid.Rows())
	page := allPages[pageIndex]
	back := pagingCell(allPages, pageIndex, -1, "Back")
	forward := pagingCell(allPages, pageIndex, 1, "Forward")
	returnTile := page.IsAppScreen()

	for row := range grid.Rows() {
		cells := make([]cellView, 0, grid.Columns())

		for column := range grid.Columns() {
			position := layout.Position{Row: row, Column: column}
			place := fmt.Sprintf("Row %d, column %d: ", row+1, column+1)
			tile := fmt.Sprintf("%d.%d.%d", pageIndex, row, column)

			switch {
			case position == grid.BackPosition():
				cells = append(cells, back(place, tile))
			case position == grid.ForwardPosition():
				cells = append(cells, forward(place, tile))
			case returnTile && position == grid.ReturnPosition():
				cells = append(cells, cellView{
					Reserved: true,
					Name:     place + "reserved for the application's icon, which goes back to the regular pages",
					Text:     "Back to the regular pages", Kind: "", Href: "", Tile: tile, State: tileIsReserve,
					Image:      fmt.Sprintf("/preview.png?page=%d&row=%d&column=%d", pageIndex, row, column),
					Selectable: false, Select: "", SelectID: "", Checked: false,
				})
			default:
				cells = append(cells, pages.keyCell(page, pageIndex, position, place, tile))
			}
		}

		rows = append(rows, cells)
	}

	return rows
}

// keyCell is a tile that holds a key or can be given one: a link to its form,
// or in select mode a check box for a key.
func (pages *pageHandlers) keyCell(
	page config.Page, pageIndex int, position layout.Position, place string, tile string,
) cellView {
	text, kindName := describeKey(pages.options.Presets, page, position)

	state := tileIsEmpty
	key, found := keyAt(page, position)

	if found {
		state = tileHoldsKey
	}

	name := place + text
	if kindName != "" {
		name += " (" + kindName + ")"
	}

	return cellView{
		Reserved: false,
		Name:     name,
		Text:     text,
		Kind:     kindName,
		Href:     fmt.Sprintf("/key?page=%d&row=%d&column=%d", pageIndex, position.Row, position.Column),
		Tile:     tile,
		State:    state,
		Image:    "",

		Selectable: found && !brick.IsPageMarker(key.Kind),
		Select:     fmt.Sprintf("%d.%d", position.Row, position.Column),
		SelectID:   fmt.Sprintf("%d-%d", position.Row, position.Column),
		Checked:    false,
	}
}

// pagingCell builds a paging tile. It links to the regular page a step away,
// wrapping round at either end as the deck does, so the grid pages like the
// deck (evaluation, deck testing). App screens are never paged to: from one,
// back goes to the last regular page and forward to the first, as if the
// screens sat after the regular pages, where the page list shows them. With
// nowhere else to go, the tile only says what it is for. It stays reserved
// either way, so nothing is dropped on it or dragged from it.
func pagingCell(allPages []config.Page, pageIndex int, step int, text string) func(place string, tile string) cellView {
	regular := regularPages(allPages)
	position := slices.Index(regular, pageIndex)

	return func(place string, tile string) cellView {
		cell := cellView{
			Reserved: true, Name: place + "reserved for paging " + strings.ToLower(text), Text: text, Kind: "",
			Href: "", Tile: tile, State: tileIsReserve, Image: "",
			Selectable: false, Select: "", SelectID: "", Checked: false,
		}

		target := -1

		switch {
		case position >= 0 && len(regular) > 1:
			target = (position + step + len(regular)) % len(regular)
		case position < 0 && len(regular) > 0 && step < 0:
			target = len(regular) - 1
		case position < 0 && len(regular) > 0:
			target = 0
		}

		if target >= 0 {
			cell.Href = fmt.Sprintf("/page?page=%d", regular[target])
			cell.Name = fmt.Sprintf("%s%s to page %d, %s", place, text, target+1, allPages[regular[target]].Name)
		}

		return cell
	}
}

// describeKey names a grid cell: its label and, apart, its kind, which the grid
// shows on a smaller line of its own; a key with no label is named by its kind
// alone, and a free position is "empty".
func describeKey(presetTable presets.Table, page config.Page, position layout.Position) (string, string) {
	key, found := keyAt(page, position)
	if !found {
		return "empty", ""
	}

	kindName := shownKind(presetTable, key)

	appearance, _ := brick.AppearanceOf(key.Parameters)
	if appearance.Label != "" {
		return appearance.Label, kindName
	}

	return kindName, ""
}

// shownKind is what the grid calls a key's kind. A preset key's kind is its
// target, usually open_url, which says nothing of what the key does, so the
// preset's action is shown instead.
func shownKind(presetTable presets.Table, key config.Key) string {
	if key.Preset == nil {
		return key.Kind
	}

	row, found := presetTable.Row(key.Preset.Name)
	if !found {
		return key.Kind
	}

	return row.Action()
}

func keyAt(page config.Page, position layout.Position) (config.Key, bool) {
	for _, key := range page.Keys {
		if key.Position == position {
			return key, true
		}
	}

	return config.Key{Position: position, Kind: "", Parameters: nil, Preset: nil}, false
}

type keyView struct {
	baseView

	PageIndex     int
	Row           int
	Column        int
	DisplayRow    int
	DisplayColumn int
	HasKey        bool
	TypeChoice    string
	Types         []typeOption
	Fields        []fieldView
	Errors        []string
	Move          moveView
	// ScreenHref and ScreenName link a key that opens an app screen to it.
	ScreenHref string
	ScreenName string
}

func (pages *pageHandlers) showKey(writer http.ResponseWriter, request *http.Request) {
	configuration, problems := pages.configuration()
	query := request.URL.Query()
	pageIndex, position := integerParameter(query, "page"), layout.Position{
		Row: integerParameter(query, "row"), Column: integerParameter(query, "column"),
	}

	context, typeError := pages.storedKeyContext(configuration, pageIndex, position, query.Get("type"))
	if typeError != nil {
		http.Error(writer, "unknown key type", http.StatusBadRequest)

		return
	}

	context.problems = problems
	pages.renderKey(writer, request, http.StatusOK, context)
}

// storedKeyContext is the key form for a position as the configuration holds
// it: the stored key's type and fields, or, for a free position, the chosen or
// first type with an empty form.
func (pages *pageHandlers) storedKeyContext(
	configuration config.Configuration,
	pageIndex int,
	position layout.Position,
	choice string,
) (keyContext, error) {
	existing, hasKey := config.Key{Position: position, Kind: "", Parameters: nil, Preset: nil}, false
	if pageIndex >= 0 && pageIndex < len(configuration.Pages) {
		existing, hasKey = keyAt(configuration.Pages[pageIndex], position)
	}

	switch {
	case choice == "" && hasKey:
		choice = typeChoiceOf(existing)
	case choice == "":
		choice = kindChoicePrefix + pages.options.Registry.KindNamesIn("")[0]
	}

	resolved, typeError := pages.resolveOn(choice, pageAt(configuration, pageIndex))
	if typeError != nil {
		var unknown keyContext

		return unknown, typeError
	}

	var filled any

	if hasKey && typeChoiceOf(existing) == choice {
		filled = storedFields(existing)
	}

	return keyContext{
		pageIndex: pageIndex, position: position, hasKey: hasKey, resolved: resolved,
		fields: fieldViews(resolved.schema, filled, nil), errors: nil, problems: nil,
		destinationPages: configuration.Pages, moveFailed: false, screen: pages.screenLink(configuration, existing),
		appTypes: pages.appTypeOptions(pageAt(configuration, pageIndex), choice),
	}, nil
}

// storedFields is what fills a stored key's form: a preset key's own fields,
// not the parameters the preset expands them into.
func storedFields(key config.Key) any {
	if key.Preset != nil {
		return key.Preset.Fields
	}

	return key.Parameters
}

// keyContext is everything the key form shows.
type keyContext struct {
	pageIndex int
	position  layout.Position
	hasKey    bool
	resolved  keyType
	fields    []fieldView
	errors    []string
	problems  []string
	// destinationPages is every page with its keys, from which the move form
	// lists the free tiles; the form is shown only for a stored key.
	destinationPages []config.Page
	moveFailed       bool
	// screen is the app screen a stored key opens, if it opens one.
	screen screenLink
	// appTypes are the page's own application's bricks, offered on its app
	// screen beside the rest.
	appTypes []typeOption
}

// screenLink is where a key's app screen is edited.
type screenLink struct {
	href string
	name string
}

// screenLink links a key that opens an app screen to that screen.
func (pages *pageHandlers) screenLink(configuration config.Configuration, key config.Key) screenLink {
	application, wanted := wantedScreen(key.Parameters)
	if !wanted {
		return screenLink{href: "", name: ""}
	}

	screenIndex, exists := appScreenIndex(configuration.Pages, application)
	if !exists {
		return screenLink{href: "", name: ""}
	}

	return screenLink{
		href: fmt.Sprintf("/page?page=%d", screenIndex), name: pages.screenTitle(configuration.Pages[screenIndex]),
	}
}

func (pages *pageHandlers) renderKey(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	context keyContext,
) {
	pages.attachChoices(request.Context(), context.fields)
	nameModifiers(context.fields, pages.options.Platform)

	title := "Key"
	if focusFirstError(context.fields) || len(context.errors) > 0 {
		title = errorPrefix + title
	}

	pages.render(writer, status, "key.html", keyView{
		baseView:      pages.base(title, request, context.problems),
		PageIndex:     context.pageIndex,
		Row:           context.position.Row,
		Column:        context.position.Column,
		DisplayRow:    context.position.Row + 1,
		DisplayColumn: context.position.Column + 1,
		HasKey:        context.hasKey,
		TypeChoice:    context.resolved.choice,
		Types:         append(pages.typeOptions(context.resolved.choice), context.appTypes...),
		Fields:        context.fields,
		Errors:        context.errors,
		Move:          pages.moveChoices(context),
		ScreenHref:    context.screen.href,
		ScreenName:    context.screen.name,
	})
}

func (pages *pageHandlers) saveKey(writer http.ResponseWriter, request *http.Request) {
	form := request.PostForm
	pageIndex := integerParameter(form, "page")
	position := layout.Position{Row: integerParameter(form, "row"), Column: integerParameter(form, "column")}

	configuration, _ := pages.configuration()
	page := pageAt(configuration, pageIndex)

	resolved, typeError := pages.resolveOn(form.Get("type"), page)
	if typeError != nil {
		http.Error(writer, "unknown key type", http.StatusBadRequest)

		return
	}

	appTypes := pages.appTypeOptions(page, resolved.choice)
	parameters, fieldErrors := parseFields(resolved.schema, form)
	failed := func(general []string) {
		views := fieldViews(resolved.schema, nil, fieldErrors)
		for index := range views {
			views[index].Value = form.Get(views[index].Name)
			views[index].Checked = form.Get(views[index].Name) != ""
		}

		pages.renderKey(writer, request, http.StatusUnprocessableEntity, keyContext{
			pageIndex: pageIndex, position: position, hasKey: false, resolved: resolved,
			fields: views, errors: general, problems: nil, destinationPages: nil, moveFailed: false,
			screen: screenLink{href: "", name: ""}, appTypes: appTypes,
		})
	}

	if len(fieldErrors) > 0 {
		failed(nil)

		return
	}

	key, buildError := pages.buildKey(resolved, position, parameters)
	if buildError != nil {
		failed([]string{errorPrefix + buildError.Error()})

		return
	}

	screen, wantsScreen := pages.wantedScreenPage(request.Context(), key.Parameters)

	saveError := pages.storeKey(pageIndex, key, screen, wantsScreen)
	if saveError != nil {
		general := []string{saveMessage(saveError)}
		if rejection, isRejection := errors.AsType[*config.RejectionError](saveError); isRejection {
			fieldErrors, general = problemsByField(rejection)
		}

		failed(general)

		return
	}

	redirectToSavedPage(writer, request, pageIndex)
}

// storeKey puts a key on its page, replacing any at its position. A key that
// asks for an app screen gets one in the same save, so the file never names a
// screen that is not there.
func (pages *pageHandlers) storeKey(pageIndex int, key config.Key, screen config.Page, wantsScreen bool) error {
	return pages.changeConfiguration(func(configuration *config.Configuration) error {
		if pageIndex < 0 || pageIndex >= len(configuration.Pages) {
			return fmt.Errorf("%w: page %d", errUnknownType, pageIndex)
		}

		page := &configuration.Pages[pageIndex]
		page.Keys = slices.DeleteFunc(page.Keys, func(existing config.Key) bool { return existing.Position == key.Position })
		page.Keys = append(page.Keys, key)

		if wantsScreen {
			addScreen(configuration, screen)
		}

		return nil
	})
}

// redirectToSavedPage sends the browser to a page after a saved change, with
// the saved message for its live region.
func redirectToSavedPage(writer http.ResponseWriter, request *http.Request, pageIndex int) {
	// The redirect's only variable part is a formatted integer, so it can only
	// ever name one of the editor's own pages.
	pageAddress := fmt.Sprintf("/page?page=%d&saved=key", pageIndex)
	http.Redirect(writer, request, pageAddress, http.StatusSeeOther) //nolint:gosec // see above
}

// saveMessage explains a save that failed for a reason other than a field.
func saveMessage(saveError error) string {
	if errors.Is(saveError, errFileRefused) {
		return fileRefusedMessage
	}

	return errorPrefix + "the change could not be saved: " + saveError.Error()
}

// changePage applies a change to one page of the file on disk and saves it
// through the store, which validates it and writes it atomically.
func (pages *pageHandlers) changePage(pageIndex int, change func(page *config.Page)) error {
	return pages.changeConfiguration(func(configuration *config.Configuration) error {
		if pageIndex < 0 || pageIndex >= len(configuration.Pages) {
			return fmt.Errorf("%w: page %d", errUnknownType, pageIndex)
		}

		change(&configuration.Pages[pageIndex])

		return nil
	})
}

func (pages *pageHandlers) changeConfiguration(change func(configuration *config.Configuration) error) error {
	configuration, editableError := pages.editable()
	if editableError != nil {
		return editableError
	}

	changeError := change(&configuration)
	if changeError != nil {
		return changeError
	}

	return pages.options.Store.Save(configuration) //nolint:wrapcheck // the store's rejection is shown field by field
}

func (pages *pageHandlers) addPage(writer http.ResponseWriter, request *http.Request) {
	name := strings.TrimSpace(request.PostForm.Get("name"))

	var newIndex int

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		configuration.Pages = append(configuration.Pages, config.Page{Name: name, Application: "", Keys: nil})
		newIndex = len(configuration.Pages) - 1

		return nil
	})

	pages.afterPageChange(writer, request, newIndex, saveError)
}

func (pages *pageHandlers) renamePage(writer http.ResponseWriter, request *http.Request) {
	pageIndex := integerParameter(request.PostForm, "page")
	name := strings.TrimSpace(request.PostForm.Get("name"))

	saveError := pages.changePage(pageIndex, func(page *config.Page) { page.Name = name })

	pages.afterPageChange(writer, request, pageIndex, saveError)
}

func (pages *pageHandlers) deletePage(writer http.ResponseWriter, request *http.Request) {
	pageIndex := integerParameter(request.PostForm, "page")

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		if pageIndex < 0 || pageIndex >= len(configuration.Pages) {
			return fmt.Errorf("%w: page %d", errUnknownType, pageIndex)
		}

		configuration.Pages = slices.Delete(configuration.Pages, pageIndex, pageIndex+1)

		return nil
	})

	pages.afterPageChange(writer, request, 0, saveError)
}

// afterPageChange redirects after a saved change, or shows the page again with
// why the change was refused.
func (pages *pageHandlers) afterPageChange(
	writer http.ResponseWriter,
	request *http.Request,
	pageIndex int,
	saveError error,
) {
	if saveError == nil {
		redirectToSavedPage(writer, request, pageIndex)

		return
	}

	configuration, _ := pages.configuration()
	pages.renderPage(writer, request, http.StatusUnprocessableEntity, configuration, pageIndex, saveProblems(saveError))
}

// saveProblems lists why a save was refused, in words: each of the store's
// problems, or the one reason the save could not happen.
func saveProblems(saveError error) []string {
	rejection, isRejection := errors.AsType[*config.RejectionError](saveError)
	if !isRejection {
		return []string{saveMessage(saveError)}
	}

	fieldProblems, problems := problemsByField(rejection)
	for _, message := range fieldProblems {
		problems = append(problems, message)
	}

	return problems
}

func (pages *pageHandlers) preview(writer http.ResponseWriter, request *http.Request) {
	configuration, _ := pages.configuration()
	query := request.URL.Query()
	page := pageAt(configuration, integerParameter(query, "page"))
	position := layout.Position{Row: integerParameter(query, "row"), Column: integerParameter(query, "column")}

	var appearance render.Appearance

	// An app screen's return tile is the application's icon alone.
	if query.Get("type") == "" && page.IsAppScreen() && position == pages.options.Grid.ReturnPosition() {
		icon := pages.loadAppIcon(request.Context(), page.Application)
		appearance = render.Appearance{IconImage: icon, IconMissing: icon == nil, BackgroundFromIcon: true}
	} else {
		appearance = pages.keyPreview(request.Context(), page, position, query)
	}

	var encoded bytes.Buffer

	encodeError := png.Encode(&encoded, pages.options.Renderer.Render(appearance))
	if encodeError != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	writer.Header().Set("Content-Type", "image/png")
	_, _ = writer.Write(encoded.Bytes())
}

// keyPreview is how a key is drawn: the form's unsaved fields, or with no
// type chosen, the key stored at the position.
func (pages *pageHandlers) keyPreview(
	previewContext context.Context,
	page config.Page,
	position layout.Position,
	query url.Values,
) render.Appearance {
	appearance := render.Appearance{
		Label: query.Get(fieldPrefix + "label"), Icon: query.Get(fieldPrefix + "icon"),
	}
	iconFile := query.Get(fieldPrefix + "icon_file")
	parameters := pages.unsavedParameters(query)

	if key, found := keyAt(page, position); found && query.Get("type") == "" {
		configured, _ := brick.AppearanceOf(key.Parameters)
		appearance.Label, appearance.Icon, iconFile = configured.Label, configured.Icon, configured.IconFile
		query.Set(fieldPrefix+"background", configured.Background)

		parameters = key.Parameters
	}

	if iconFile != "" {
		appearance.IconImage = pages.loadIconFile(iconFile)
		appearance.IconMissing = appearance.IconImage == nil
	}

	if source, offers := parameters.(brick.ApplicationIconSource); offers {
		if identifier, wanted := source.ApplicationIcon(); wanted {
			appearance.IconImage = pages.loadAppIcon(previewContext, identifier)
			appearance.IconMissing = appearance.IconImage == nil
			appearance.BackgroundFromIcon = true
		}
	}

	background, parseError := render.ParseColour(query.Get(fieldPrefix + "background"))
	if parseError == nil {
		appearance.Background = background
	}

	return appearance
}

type credentialRow struct {
	Name   string
	Status string
}

type credentialsView struct {
	baseView

	Credentials []credentialRow
	Error       string
}

func (pages *pageHandlers) showCredentials(writer http.ResponseWriter, request *http.Request) {
	pages.renderCredentials(writer, request, http.StatusOK, "")
}

func (pages *pageHandlers) renderCredentials(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	message string,
) {
	configuration, problems := pages.configuration()
	view := credentialsView{baseView: pages.base("Credentials", request, problems), Credentials: nil, Error: message}

	for _, credentialName := range pages.credentialNames(configuration) {
		state := "Missing"

		_, readError := pages.options.Credentials.ReadCredential(credentialName)
		if readError == nil {
			state = "Stored"

			host, hostError := platform.CredentialHost(pages.options.Credentials, credentialName)
			if hostError == nil {
				state = "Stored, sent only to " + host
			}
		}

		view.Credentials = append(view.Credentials, credentialRow{Name: credentialName, Status: state})
	}

	pages.render(writer, status, "credentials.html", view)
}

// credentialNames lists every credential the configuration names, from each
// kind's secret fields, so a person can see which are stored without any secret
// being shown.
func (pages *pageHandlers) credentialNames(configuration config.Configuration) []string {
	var names []string

	for _, page := range configuration.Pages {
		for _, key := range page.Keys {
			kind, found := pages.options.Registry.Kind(key.Kind)
			if !found {
				continue
			}

			for _, field := range kind.Schema().Fields() {
				credentialName, isText := kind.Schema().Read(key.Parameters, field).(string)
				if field.Secret && isText && credentialName != "" && !slices.Contains(names, credentialName) {
					names = append(names, credentialName)
				}
			}
		}
	}

	slices.Sort(names)

	return names
}

// storeCredential writes a secret and redirects; the secret is never written
// back into any response, redirect or log (specification section 2).
func (pages *pageHandlers) storeCredential(writer http.ResponseWriter, request *http.Request) {
	credentialName := strings.TrimSpace(request.PostForm.Get("name"))

	// The host the token may be sent to is checked before anything is stored,
	// and bound beside it after: api.github.com unless an Enterprise host is
	// given.
	host := strings.TrimSpace(request.PostForm.Get("host"))
	if host == "" {
		host = platform.DefaultCredentialHost
	}

	if platform.CheckCredentialHost(host) != nil || len(credentialName) > platform.MaximumBindableNameLength {
		pages.renderCredentials(writer, request, http.StatusUnprocessableEntity,
			errorPrefix+"the host must be a plain name such as github.example.com, with no https:// or path, "+
				"and the name at most 59 characters")

		return
	}

	writeError := pages.options.Credentials.WriteCredential(credentialName, request.PostForm.Get("secret"))
	if writeError == nil {
		writeError = platform.BindCredentialHost(pages.options.Credentials, credentialName, host)
	}

	if writeError != nil {
		message := errorPrefix + "the credential could not be stored"
		if errors.Is(writeError, platform.ErrCredentialRefused) {
			message = errorPrefix + "the name must be 1 to 64 characters and the secret 1 to 2,048, with no line breaks"
		}

		pages.renderCredentials(writer, request, http.StatusUnprocessableEntity, message)

		return
	}

	http.Redirect(writer, request, "/credentials?saved=credential", http.StatusSeeOther)
}

type counterRow struct {
	Name  string
	Value int64
}

type diagnosticsView struct {
	baseView

	Counters  []counterRow
	Rejection string
}

func (pages *pageHandlers) showDiagnostics(writer http.ResponseWriter, request *http.Request) {
	_, problems := pages.configuration()
	view := diagnosticsView{baseView: pages.base("Diagnostics", request, problems), Counters: nil, Rejection: ""}

	snapshot := pages.options.Counters.Snapshot()
	for name, value := range snapshot {
		view.Counters = append(view.Counters, counterRow{Name: name, Value: value})
	}

	slices.SortFunc(view.Counters, func(first counterRow, second counterRow) int {
		return strings.Compare(first.Name, second.Name)
	})

	rejection := pages.options.Store.LastRejection()
	if rejection != nil {
		view.Rejection = rejection.Error()
	}

	pages.render(writer, http.StatusOK, "diagnostics.html", view)
}

// unsavedParameters reads a previewed form's fields through its kind's schema,
// so the preview can ask the parameters, as the daemon does, whether the key
// shows its application's own icon. It returns nil for anything but a known
// kind.
func (pages *pageHandlers) unsavedParameters(query url.Values) any {
	kindName, isKind := strings.CutPrefix(query.Get("type"), kindChoicePrefix)
	if !isKind || pages.options.Registry == nil {
		return nil
	}

	kind, found := pages.options.Registry.Kind(kindName)
	if !found {
		return nil
	}

	parameters, _ := parseFields(kind.Schema(), query)

	return parameters
}

// loadAppIcon reads an application's own icon for the preview, or returns nil,
// drawn as the missing marker, if there is no way to read one or it will not
// load. The daemon logs the reason; the preview only shows the result.
func (pages *pageHandlers) loadAppIcon(iconContext context.Context, identifier string) image.Image {
	if pages.options.AppIcons == nil {
		return nil
	}

	icon, readError := pages.options.AppIcons.Icon(iconContext, identifier)
	if readError != nil {
		return nil
	}

	return icon
}

// loadIconFile reads a key's own icon for the preview, or returns nil, drawn as
// the missing marker, if there is no icons folder or the file will not load.
func (pages *pageHandlers) loadIconFile(fileName string) image.Image {
	if pages.options.IconFiles == nil {
		return nil
	}

	icon, loadError := pages.options.IconFiles.Load(fileName)
	if loadError != nil {
		return nil
	}

	return icon
}
