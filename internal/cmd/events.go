package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/payload"
)

var eventsGroup = Group{Name: "events", Help: "Manage calendar events.", Children: []Node{
	Custom{Name: "list", Help: "List calendar events.", Default: true, Cmd: &EventsListCmd{}},
	Leaf{Name: "upcoming", Help: "List upcoming events.", Endpoint: "events.upcoming", Verb: "get upcoming events", Output: JSON{}},
	Custom{Name: "add", Help: "Add a calendar event.", Cmd: &EventsAddCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a calendar event.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Event ID."`
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

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "events.list", map[string]string{"startDate": c.StartDate.orToday(), "pageIndex": strconv.Itoa(page), "limit": strconv.Itoa(c.Limit), "sortOrder": c.Sort}, nil)
	if err != nil {
		return fmt.Errorf("list events: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
	}

	events, err := parseEvents(data)
	if err != nil {
		return err
	}

	rows := formatEventListRows(events)
	header := []string{"ID", "DATE", "NAME", "TYPE", "LOCATION"}

	return outfmt.WriteRows(g.Context, g.Stdout, header, rows)
}

// parseEvents unmarshals the events list response.
func parseEvents(data json.RawMessage) ([]map[string]any, error) {
	var events []map[string]any
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parse events: %w", err)
	}
	return events, nil
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
	Name     string        `help:"Event name." required:"" short:"n"`
	Date     Date          `help:"Event date (YYYY-MM-DD, today, +30d)." required:"" short:"d"`
	Type     string        `help:"Event type." required:"" short:"t" enum:"running,trail_running,cycling,gravel_cycling,mountain_biking,swimming,triathlon,multi_sport,hiking,walking,fitness_equipment,motorcycling,winter_sport,other"`
	Race     bool          `help:"Mark as a race."`
	Location string        `help:"Event location."`
	Time     Clock         `help:"Start time (HH:MM)." placeholder:"HH:MM"`
	Timezone string        `help:"Time zone ID (e.g. Europe/Berlin)."`
	Distance EventDistance `help:"Target distance (e.g. 10km, 26.2mi, 400m)." placeholder:"DISTANCE"`
	Goal     Goal          `help:"Time goal duration (e.g. 50m, 1h30m, 2400s)."`
	Primary  bool          `help:"Set as primary training event." xor:"priority"`
	Training bool          `help:"Set as supporting training event." xor:"priority"`
	Private  bool          `help:"Make event private."`
	Note     string        `help:"Event note."`
	URL      string        `help:"Event URL."`
}

// EventDistance and Goal are the Kong types of the event target values. Kong
// decodes them through the payload parsers at parse time, so a bad value is
// a usage error before the login. Distance is taken by the rows Format.
type (
	EventDistance payload.Distance
	Goal          payload.Goal
)

func (d *EventDistance) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "distance")
	if err != nil {
		return err
	}
	v, err := payload.ParseDistance(raw)
	if err != nil {
		return err
	}
	*d = EventDistance(v)
	return nil
}

func (g *Goal) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "goal")
	if err != nil {
		return err
	}
	v, err := payload.ParseGoal(raw)
	if err != nil {
		return err
	}
	*g = Goal(v)
	return nil
}

func (c *EventsAddCmd) Run(g *Globals) error {
	body, err := payload.Event{
		Name:     c.Name,
		Date:     string(c.Date),
		Type:     c.Type,
		Race:     c.Race,
		Location: c.Location,
		Time:     string(c.Time),
		Timezone: c.Timezone,
		Distance: payload.Distance(c.Distance),
		Goal:     payload.Goal(c.Goal),
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
