package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
)

var calendarGroup = Group{Name: "calendar", Help: "Manage the Garmin Connect calendar.", Children: []Node{
	Custom{Name: "month", Help: "List the items of a calendar month, such as activities and notes.", Cmd: &CalendarMonthCmd{}},
	calendarNotesGroup,
	calendarExportGroup,
	calendarPreferencesGroup,
}}

// CalendarMonthCmd lists the items of one calendar month.
type CalendarMonthCmd struct {
	Year  int   `arg:"" optional:"" help:"Year (e.g. 2026). Give it with MONTH. Without both, the command shows this month."`
	Month Month `arg:"" optional:"" help:"Month (1-12)."`
}

func (c *CalendarMonthCmd) Validate() error {
	if (c.Year == 0) != (c.Month == "") {
		return errors.New("give both YEAR and MONTH, or neither for this month")
	}
	return nil
}

// calendarItemRows prints the items of a month view. The view also holds
// the days of the weeks before and after the month that fill its grid.
var calendarItemRows = Rows{Noun: "calendar month", Items: "calendarItems", Columns: []Column{
	{Label: "DATE", Path: "date"},
	{Label: "TYPE", Path: "itemType"},
	{Label: "ID", Path: "id"},
	{Label: "TITLE", Path: "title"},
}}

func (c *CalendarMonthCmd) Run(g *Globals) error {
	now := nowFn()
	year, month := now.Year(), int(now.Month())
	if c.Year != 0 {
		year = c.Year
		month, _ = strconv.Atoi(string(c.Month))
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	params := map[string]string{"year": strconv.Itoa(year), "month": strconv.Itoa(month - 1)}
	data, err := client.Call(g.Context, "calendar.month", params, nil)
	if err != nil {
		return fmt.Errorf("get calendar month: %w", err)
	}
	return calendarItemRows.print(g, client, data)
}

var calendarExportGroup = Group{Name: "export", Help: "Show, turn on and turn off the iCalendar feed for calendar apps.", Children: []Node{
	Custom{Name: "show", Help: "Show the calendar feed URL.", Cmd: &CalendarExportShowCmd{}},
	Custom{Name: "enable", Help: "Turn on the calendar feed and show its URL.", Cmd: &CalendarExportEnableCmd{}},
	Custom{Name: "disable", Help: "Turn off the calendar feed. Its URL then gives an empty answer.", Cmd: &CalendarExportDisableCmd{}},
}}

// calendarFeedOff is the export ID that Garmin answers while the calendar
// feed is off.
const calendarFeedOff = "NOT_PUBLISHED"

// readCalendarExport returns Garmin's export answer and the feed ID, or ""
// while the feed is off.
func readCalendarExport(g *Globals, client *garminapi.Client) (json.RawMessage, string, error) {
	data, err := client.Call(g.Context, "calendar.export.get", nil, nil)
	if err != nil {
		return nil, "", fmt.Errorf("get calendar export: %w", err)
	}
	id, err := calendarFeedID(data)
	return data, id, err
}

func calendarFeedID(data json.RawMessage) (string, error) {
	var export struct {
		ID string `json:"id"`
	}
	if err := unmarshalAnswer(data, &export); err != nil {
		return "", fmt.Errorf("parse calendar export: %w", err)
	}
	if export.ID == calendarFeedOff {
		return "", nil
	}
	return export.ID, nil
}

var calendarFeedHeader = []string{"URL"}

// calendarFeedRows is the one row of the feed URL. Garmin's own baseUrl
// names localhost, so the URL is built from the web app host of the account.
func calendarFeedRows(client *garminapi.Client, id string) [][]string {
	feed := garminauth.NewEndpoints(client.Tokens().Domain).Connect + "/modern/calendar/export/" + url.PathEscape(id)
	return [][]string{{feed}}
}

// CalendarExportShowCmd shows the calendar feed URL.
type CalendarExportShowCmd struct{}

func (c *CalendarExportShowCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, id, err := readCalendarExport(g, client)
	if err != nil {
		return err
	}
	if outfmt.IsJSON(g.Context) {
		return writeObject(g, data)
	}
	if id == "" {
		g.UI.Infof("The calendar feed is off. Run gccli calendar export enable to turn it on.")
		return nil
	}
	return outfmt.WriteRows(g.Context, g.Stdout, calendarFeedHeader, calendarFeedRows(client, id))
}

// CalendarExportEnableCmd turns on the calendar feed. Garmin answers 403 to
// a second create, so a feed that is on stays as it is.
type CalendarExportEnableCmd struct{}

func (c *CalendarExportEnableCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, id, err := readCalendarExport(g, client)
	if err != nil {
		return err
	}
	if id == "" {
		data, err = client.Call(g.Context, "calendar.export.create", nil, []byte("{}"))
		if err != nil {
			return fmt.Errorf("enable calendar export: %w", err)
		}
		if id, err = calendarFeedID(data); err != nil {
			return err
		}
	}
	const noURL = "Turned on the calendar feed. Run gccli calendar export show to see its URL."
	return reportWriteTable(g, data, noURL, calendarFeedHeader, func() ([][]string, error) {
		if id == "" {
			return nil, errors.New("garmin turned on the calendar feed, but its answer has no feed ID: run gccli calendar export show")
		}
		return calendarFeedRows(client, id), nil
	})
}

// CalendarExportDisableCmd turns off the calendar feed.
type CalendarExportDisableCmd struct {
	Force bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *CalendarExportDisableCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, id, err := readCalendarExport(g, client)
	if err != nil {
		return err
	}
	if id == "" {
		return reportWrite(g, data, "The calendar feed is already off")
	}
	ok, err := confirm(g.Stderr, "Turn off the calendar feed? Calendar apps that subscribe to it then get an empty answer.", c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	data, err = client.Call(g.Context, "calendar.export.delete", map[string]string{"exportID": id}, nil)
	if err != nil {
		return fmt.Errorf("disable calendar export: %w", err)
	}
	return reportWrite(g, data, "Turned off the calendar feed")
}

var calendarPreferencesGroup = Group{Name: "preferences", Help: "Show and change what the Garmin Connect web calendar shows.", Children: []Node{
	Leaf{Name: "view", Help: "Show the calendar preferences.", Default: true, Endpoint: "calendar.preferences.get", Verb: "get calendar preferences", Output: JSON{}},
	Custom{Name: "set", Help: "Turn a calendar preference on or off.", Cmd: &CalendarPreferencesSetCmd{}},
}}

// CalendarPreferencesSetCmd turns one calendar preference on or off.
type CalendarPreferencesSetCmd struct {
	Key   FreeText `arg:"" help:"Preference key, such as showWeeklyTotal. Run gccli calendar preferences to see the keys."`
	Value string   `arg:"" help:"true turns the preference on, false turns it off." enum:"true,false"`
}

func (c *CalendarPreferencesSetCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.SetCalendarPreference(g.Context, string(c.Key), c.Value == "true")
	if err != nil {
		return fmt.Errorf("set calendar preference: %w", err)
	}
	return reportWrite(g, data, "Set calendar preference %s to %s", c.Key, c.Value)
}

var calendarNotesGroup = Group{Name: "notes", Help: "Add, change, move and delete calendar notes.", Children: []Node{
	Custom{Name: "add", Help: "Add a note to a calendar day.", Cmd: &CalendarNotesAddCmd{}},
	Custom{Name: "update", Help: "Change the name or text of a calendar note.", Cmd: &CalendarNotesUpdateCmd{}},
	Leaf{
		Name: "move", Help: "Move a calendar note to another day.",
		Spec: argsOf(&struct {
			ID   ID   `arg:"" help:"Note ID."`
			Date Date `arg:"" help:"New date (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
		}{}),
		Endpoint: "calendar.note.move",
		Params:   Params{"noteID": In("ID"), "date": In("Date")},
		Verb:     "move note",
		Output:   Write{Done: "Moved note {ID} to {Date}"},
	},
	Leaf{
		Name: "delete", Help: "Delete a calendar note.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Note ID."`
		}{}),
		Endpoint: "calendar.note.delete",
		Params:   Params{"noteID": In("ID")},
		Verb:     "delete note",
		Output:   Write{Done: "Deleted note {ID}", Confirm: "Delete note {ID}?"},
	},
}}

// CalendarNotesAddCmd adds a note to a calendar day.
type CalendarNotesAddCmd struct {
	Date    Date   `arg:"" help:"Date of the note (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
	Name    string `arg:"" help:"Note name."`
	Content string `help:"Note text."`
}

func (c *CalendarNotesAddCmd) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("the note name must not be empty")
	}
	return nil
}

func (c *CalendarNotesAddCmd) Run(g *Globals) error {
	note := map[string]any{"noteName": c.Name, "date": string(c.Date)}
	if c.Content != "" {
		note["content"] = c.Content
	}
	body, err := json.Marshal(note)
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "calendar.note.create", nil, body)
	if err != nil {
		return fmt.Errorf("add note: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "id"); id != "" {
		return reportWrite(g, data, "Added note %q on %s (ID: %s)", c.Name, c.Date, id)
	}
	return reportWrite(g, data, "Added note %q on %s", c.Name, c.Date)
}

// CalendarNotesUpdateCmd changes the name or text of a calendar note.
type CalendarNotesUpdateCmd struct {
	ID      ID      `arg:"" help:"Note ID."`
	Name    *string `help:"New name." placeholder:"STRING"`
	Content *string `help:"New text. An empty text clears it." placeholder:"STRING"`
}

func (c *CalendarNotesUpdateCmd) Validate() error {
	if c.Name == nil && c.Content == nil {
		return errors.New("nothing to change: give --name or --content")
	}
	if c.Name != nil && strings.TrimSpace(*c.Name) == "" {
		return errors.New("the note name must not be empty")
	}
	return nil
}

func (c *CalendarNotesUpdateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.UpdateNote(g.Context, string(c.ID), garminapi.NoteEdit{Name: c.Name, Content: c.Content})
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}
	return reportWrite(g, data, "Updated note %s", c.ID)
}
