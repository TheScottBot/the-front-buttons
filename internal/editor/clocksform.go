package editor

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// The world clocks form: under the dashboard's preview, a row for each of up
// to four clocks, a label and a zone, filled in from the file, or with the
// defaults the deck shows when the file names none. Saving puts the rows in
// the file, in order, a blank row left out, through the store, which checks
// them as it checks a hand edit; refused, the form comes back with what was
// typed and each problem beside its field. Use the defaults takes the clocks
// out of the file. Like every change, a save can be undone.

// The world clocks form's announcement after a save.
const (
	clocksQuery   = "clocks"
	clocksMessage = "Clocks saved. The deck shows them the next time its world clocks open."
)

// clockProblemPath picks the clock and its field out of a problem's path,
// such as world_clocks[1].zone; the field is empty for the clock as a whole.
var clockProblemPath = regexp.MustCompile(`^world_clocks\[(\d+)\](?:\.(label|zone))?$`)

// The fields of a clock, as a problem's path names them.
const (
	clockLabelField = "label"
	clockZoneField  = "zone"
)

// clockRow is one row of the form: its place, what it holds, and any problem
// with its label or zone.
type clockRow struct {
	Index      int
	Label      string
	Zone       string
	LabelError string
	ZoneError  string
}

// Number is the row as a person counts it, from one.
func (row clockRow) Number() int { return row.Index + 1 }

// clocksForm is the world clocks form: its rows, whether the file names no
// clocks, and problems that belong to no one field.
type clocksForm struct {
	Rows          []clockRow
	UsingDefaults bool
	Errors        []string
}

// clocksFormFor is the form filled in from a configuration's clocks, or the
// defaults where it names none.
func clocksFormFor(configuration config.Configuration) clocksForm {
	form := clocksForm{Rows: blankClockRows(), UsingDefaults: configuration.WorldClocks == nil, Errors: nil}

	for index, clock := range configuration.WorldClocksOrDefault() {
		if index < len(form.Rows) {
			form.Rows[index].Label, form.Rows[index].Zone = clock.Label, clock.Zone
		}
	}

	return form
}

// blankClockRows is a row for each clock the deck can show, all blank.
func blankClockRows() []clockRow {
	rows := make([]clockRow, config.MaximumWorldClocks)
	for index := range rows {
		rows[index] = clockRow{Index: index, Label: "", Zone: "", LabelError: "", ZoneError: ""}
	}

	return rows
}

// clocksFormFrom is the form as it was sent, each field trimmed.
func clocksFormFrom(values url.Values) clocksForm {
	form := clocksForm{Rows: blankClockRows(), UsingDefaults: false, Errors: nil}

	for index := range form.Rows {
		form.Rows[index].Label = strings.TrimSpace(values.Get("label." + strconv.Itoa(index)))
		form.Rows[index].Zone = strings.TrimSpace(values.Get("zone." + strconv.Itoa(index)))
	}

	return form
}

// clocks is the form's rows as clocks, a blank row left out, with the row
// each came from.
func (form clocksForm) clocks() ([]config.WorldClock, []int) {
	clocks := []config.WorldClock{}

	var rowOf []int

	for _, row := range form.Rows {
		if row.Label == "" && row.Zone == "" {
			continue
		}

		clocks = append(clocks, config.WorldClock{Label: row.Label, Zone: row.Zone})
		rowOf = append(rowOf, row.Index)
	}

	return clocks, rowOf
}

// withProblems puts why a save was refused beside the fields it concerns,
// counting each clock back to the row it came from.
func (form clocksForm) withProblems(saveError error, rowOf []int) clocksForm {
	rejection, isRejection := errors.AsType[*config.RejectionError](saveError)
	if !isRejection {
		form.Errors = append(form.Errors, saveMessage(saveError))

		return form
	}

	for _, problem := range rejection.Problems {
		message := errorPrefix + problem.Detail
		match := clockProblemPath.FindStringSubmatch(problem.Path)

		clockIndex := -1
		if match != nil {
			clockIndex, _ = strconv.Atoi(match[1])
		}

		if clockIndex < 0 || clockIndex >= len(rowOf) {
			form.Errors = append(form.Errors, message)

			continue
		}

		row := &form.Rows[rowOf[clockIndex]]

		switch match[2] {
		case clockLabelField:
			row.LabelError = message
		case clockZoneField:
			row.ZoneError = message
		default:
			form.Errors = append(form.Errors, fmt.Sprintf("%sClock %d: %s", errorPrefix, row.Number(), problem.Detail))
		}
	}

	return form
}

// saveClocks saves the form's clocks, or with Use the defaults takes them out
// of the file.
func (pages *pageHandlers) saveClocks(writer http.ResponseWriter, request *http.Request) {
	form := clocksFormFrom(request.PostForm)

	clocks, rowOf := form.clocks()
	if request.PostForm.Get("defaults") != "" {
		clocks = nil
	}

	saveError := pages.changeConfiguration(func(configuration *config.Configuration) error {
		configuration.WorldClocks = clocks

		return nil
	})
	if saveError == nil {
		http.Redirect(writer, request, "/dashboard?saved="+clocksQuery, http.StatusSeeOther)

		return
	}

	configuration, problems := pages.configuration()
	form.UsingDefaults = configuration.WorldClocks == nil
	pages.renderDashboard(writer, request, http.StatusUnprocessableEntity, configuration, problems,
		form.withProblems(saveError, rowOf))
}
