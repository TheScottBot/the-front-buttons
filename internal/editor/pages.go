package editor

import (
	"bytes"
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
}

func newPageHandlers(options Options) *pageHandlers {
	// The templates are compiled into the binary and proven by the tests, so a
	// failure here is a broken build, found at start up rather than on a request.
	templates := template.Must(template.ParseFS(embeddedFiles, "templates/*.html"))

	return &pageHandlers{options: options, templates: templates, icons: newIconIndex(options.Icons)}
}

// baseView is what every page shows: the product name, the token for its forms,
// a message for the live region and any problems with the file on disk.
type baseView struct {
	Title    string
	Product  string
	Token    string
	Message  string
	Problems []string
}

func (pages *pageHandlers) base(title string, request *http.Request, problems []string) baseView {
	message := ""

	switch request.URL.Query().Get("saved") {
	case "key":
		message = savedMessage
	case "credential":
		message = credentialMessage
	}

	return baseView{
		Title:    title,
		Product:  platform.ProductDisplayName,
		Token:    pages.options.Token,
		Message:  message,
		Problems: problems,
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
	Rows       [][]cellView
}

func (pages *pageHandlers) showPage(writer http.ResponseWriter, request *http.Request) {
	configuration, problems := pages.configuration()
	pages.renderPage(writer, request, http.StatusOK, configuration,
		integerParameter(request.URL.Query(), "page"), problems)
}

func (pages *pageHandlers) renderPage(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	configuration config.Configuration,
	pageIndex int,
	problems []string,
) {
	view := pageView{
		baseView:   pages.base("Pages", request, problems),
		PageIndex:  0,
		PageNumber: 0,
		PageCount:  len(configuration.Pages),
		PageName:   "",
		HasPage:    false,
		Pages:      nil,
		Rows:       nil,
	}

	for index, page := range configuration.Pages {
		view.Pages = append(view.Pages, pageLink{Index: index, Name: page.Name, Current: index == pageIndex})
	}

	if pageIndex >= 0 && pageIndex < len(configuration.Pages) {
		view.HasPage = true
		view.PageIndex = pageIndex
		view.PageNumber = pageIndex + 1
		view.PageName = configuration.Pages[pageIndex].Name
		view.Rows = pages.gridRows(configuration.Pages[pageIndex], pageIndex)
	}

	pages.render(writer, status, "page.html", view)
}

func (pages *pageHandlers) gridRows(page config.Page, pageIndex int) [][]cellView {
	grid := pages.options.Grid
	rows := make([][]cellView, 0, grid.Rows())

	for row := range grid.Rows() {
		cells := make([]cellView, 0, grid.Columns())

		for column := range grid.Columns() {
			position := layout.Position{Row: row, Column: column}
			place := fmt.Sprintf("Row %d, column %d: ", row+1, column+1)
			tile := fmt.Sprintf("%d.%d.%d", pageIndex, row, column)

			switch {
			case position == grid.BackPosition():
				cells = append(cells, reservedCell(place+"reserved for paging back", "Back", tile))
			case position == grid.ForwardPosition():
				cells = append(cells, reservedCell(place+"reserved for paging forward", "Forward", tile))
			default:
				text, kindName := describeKey(page, position)

				state := tileIsEmpty
				if _, found := keyAt(page, position); found {
					state = tileHoldsKey
				}

				name := place + text
				if kindName != "" {
					name += " (" + kindName + ")"
				}

				cells = append(cells, cellView{
					Reserved: false,
					Name:     name,
					Text:     text,
					Kind:     kindName,
					Href:     fmt.Sprintf("/key?page=%d&row=%d&column=%d", pageIndex, row, column),
					Tile:     tile,
					State:    state,
				})
			}
		}

		rows = append(rows, cells)
	}

	return rows
}

// reservedCell is a paging position: named, shown, never a link or a drop
// target.
func reservedCell(name string, text string, tile string) cellView {
	return cellView{
		Reserved: true, Name: name, Text: text, Kind: "", Href: "", Tile: tile, State: tileIsReserve,
	}
}

// describeKey names a grid cell: its label and, apart, its kind, which the grid
// shows on a smaller line of its own; a key with no label is named by its kind
// alone, and a free position is "empty".
func describeKey(page config.Page, position layout.Position) (string, string) {
	key, found := keyAt(page, position)
	if !found {
		return "empty", ""
	}

	appearance, _ := brick.AppearanceOf(key.Parameters)
	if appearance.Label != "" {
		return appearance.Label, key.Kind
	}

	return key.Kind, ""
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
		choice = kindChoicePrefix + pages.options.Registry.KindNames()[0]
	}

	resolved, typeError := pages.resolveType(choice)
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
		destinationPages: configuration.Pages, moveFailed: false,
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
}

func (pages *pageHandlers) renderKey(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	context keyContext,
) {
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
		Types:         pages.typeOptions(context.resolved.choice),
		Fields:        context.fields,
		Errors:        context.errors,
		Move:          pages.moveChoices(context),
	})
}

func (pages *pageHandlers) saveKey(writer http.ResponseWriter, request *http.Request) {
	form := request.PostForm
	pageIndex := integerParameter(form, "page")
	position := layout.Position{Row: integerParameter(form, "row"), Column: integerParameter(form, "column")}

	resolved, typeError := pages.resolveType(form.Get("type"))
	if typeError != nil {
		http.Error(writer, "unknown key type", http.StatusBadRequest)

		return
	}

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

	saveError := pages.changePage(pageIndex, func(page *config.Page) {
		page.Keys = slices.DeleteFunc(page.Keys, func(existing config.Key) bool { return existing.Position == position })
		page.Keys = append(page.Keys, key)
	})
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

func (pages *pageHandlers) deleteKey(writer http.ResponseWriter, request *http.Request) {
	form := request.PostForm
	pageIndex := integerParameter(form, "page")
	position := layout.Position{Row: integerParameter(form, "row"), Column: integerParameter(form, "column")}

	saveError := pages.changePage(pageIndex, func(page *config.Page) {
		page.Keys = slices.DeleteFunc(page.Keys, func(existing config.Key) bool { return existing.Position == position })
	})

	pages.afterPageChange(writer, request, pageIndex, saveError)
}

func (pages *pageHandlers) addPage(writer http.ResponseWriter, request *http.Request) {
	name := strings.TrimSpace(request.PostForm.Get("name"))

	var newIndex int

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		configuration.Pages = append(configuration.Pages, config.Page{Name: name, Keys: nil})
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
	pageIndex := integerParameter(query, "page")
	position := layout.Position{Row: integerParameter(query, "row"), Column: integerParameter(query, "column")}

	appearance := render.Appearance{
		Label: query.Get(fieldPrefix + "label"), Icon: query.Get(fieldPrefix + "icon"),
	}
	iconFile := query.Get(fieldPrefix + "icon_file")

	if query.Get("type") == "" && pageIndex >= 0 && pageIndex < len(configuration.Pages) {
		if key, found := keyAt(configuration.Pages[pageIndex], position); found {
			configured, _ := brick.AppearanceOf(key.Parameters)
			appearance.Label, appearance.Icon, iconFile = configured.Label, configured.Icon, configured.IconFile
			query.Set(fieldPrefix+"background", configured.Background)
		}
	}

	if iconFile != "" {
		appearance.IconImage = pages.loadIconFile(iconFile)
		appearance.IconMissing = appearance.IconImage == nil
	}

	background, parseError := render.ParseColour(query.Get(fieldPrefix + "background"))
	if parseError == nil {
		appearance.Background = background
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

	writeError := pages.options.Credentials.WriteCredential(credentialName, request.PostForm.Get("secret"))
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
