package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bpauli/gccli/internal/payload"
)

var eventsGroup = Group{Name: "events", Help: "Manage calendar events.", Children: []Node{
	Custom{Name: "list", Help: "List calendar events.", Default: true, Cmd: &EventsListCmd{}},
	Leaf{Name: "upcoming", Help: "List upcoming events.", Endpoint: "events.upcoming", Verb: "get upcoming events", Output: JSON{}},
	Custom{Name: "add", Help: "Add a calendar event.", Cmd: &EventsAddCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a calendar event.",
		Spec: argsOf(&struct {
			ID string `arg:"" help:"Event ID."`
		}{}),
		Endpoint: "events.delete",
		Params:   Params{"eventID": In("ID")},
		Verb:     "delete event",
		Output:   Write{Done: "Deleted event {ID}", Confirm: "Delete event {ID}?"},
	},
}}

// EventsListCmd lists calendar events from a start date.
type EventsListCmd struct {
	StartDate  Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
	Pagination `embed:"" set:"items=events"`
	Sort       string `help:"Sort order." default:"eventDate_asc" enum:"eventDate_asc,eventDate_desc"`
}

func (c *EventsListCmd) Validate() error {
	if err := c.Pagination.Validate(); err != nil {
		return err
	}
	_, err := c.pageIndex()
	return err
}

func (c *EventsListCmd) pageIndex() (int, error) {
	if c.Limit < 1 {
		return 0, fmt.Errorf("--limit must be at least 1, got %d", c.Limit)
	}
	if c.Offset%c.Limit != 0 {
		return 0, fmt.Errorf("--offset %d is not a multiple of --limit %d; events are fetched in pages of --limit", c.Offset, c.Limit)
	}
	return c.Offset/c.Limit + 1, nil
}

func (c *EventsListCmd) Run(g *Globals) error {
	page, err := c.pageIndex()
	if err != nil {
		return err
	}

	startDate := string(c.StartDate)
	if startDate == "" {
		startDate = "today"
	}

	date, err := resolveDate(startDate)
	if err != nil {
		return err
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "events.list", map[string]string{"startDate": date, "pageIndex": strconv.Itoa(page), "limit": strconv.Itoa(c.Limit), "sortOrder": c.Sort}, nil)
	if err != nil {
		return fmt.Errorf("list events: %w", err)
	}

	return list{noun: "events", header: []string{"ID", "DATE", "NAME", "TYPE", "LOCATION"}, rows: formatEventListRows}.print(g, data)
}

// formatEventListRows formats events for table output.
func formatEventListRows(events []map[string]any) [][]string {
	rows := make([][]string, 0, len(events))
	for _, e := range events {
		id := jsonString(e, "shareableEventUuid")
		if id == "" {
			id = jsonString(e, "id")
		}
		rows = append(rows, []string{
			id,
			jsonString(e, "date"),
			jsonString(e, "eventName"),
			jsonString(e, "eventType"),
			jsonString(e, "location"),
		})
	}
	return rows
}

// EventsAddCmd adds a new calendar event.
type EventsAddCmd struct {
	Name     string `help:"Event name." required:"" short:"n"`
	Date     string `help:"Event date (YYYY-MM-DD, today, +30d)." required:"" short:"d"`
	Type     string `help:"Event type." required:"" short:"t" enum:"running,trail_running,cycling,gravel_cycling,mountain_biking,swimming,triathlon,multi_sport,hiking,walking,fitness_equipment,motorcycling,winter_sport,other"`
	Race     bool   `help:"Mark as a race."`
	Location string `help:"Event location."`
	Time     string `help:"Start time (HH:MM)."`
	Timezone string `help:"Time zone ID (e.g. Europe/Berlin)."`
	Distance string `help:"Target distance (e.g. 10km, 26.2mi, 400m)."`
	Goal     string `help:"Time goal duration (e.g. 50m, 1h30m, 2400s)."`
	Primary  bool   `help:"Set as primary training event." xor:"priority"`
	Training bool   `help:"Set as supporting training event." xor:"priority"`
	Private  bool   `help:"Make event private."`
	Note     string `help:"Event note."`
	URL      string `help:"Event URL."`
}

func (c *EventsAddCmd) Run(g *Globals) error {
	date, err := resolveDate(c.Date)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}

	body, err := payload.Event{
		Name:     c.Name,
		Date:     date,
		Type:     c.Type,
		Race:     c.Race,
		Location: c.Location,
		Time:     c.Time,
		Timezone: c.Timezone,
		Distance: c.Distance,
		Goal:     c.Goal,
		Primary:  c.Primary,
		Training: c.Training,
		Private:  c.Private,
		Note:     c.Note,
		URL:      c.URL,
	}.JSON()
	if err != nil {
		return err
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "events.add", nil, body)
	if err != nil {
		return fmt.Errorf("add event: %w", err)
	}

	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "id"); id != "" {
		return reportWrite(g, data, "Added event %q on %s (ID: %s)", jsonString(resp, "eventName"), jsonString(resp, "date"), id)
	}
	return reportWrite(g, data, "Added event")
}
