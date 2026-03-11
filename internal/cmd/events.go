package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

// EventsCmd groups event subcommands.
type EventsCmd struct {
	List   EventsListCmd   `cmd:"" default:"withargs" help:"List calendar events."`
	Add    EventsAddCmd    `cmd:"" help:"Add a calendar event."`
	Delete EventsDeleteCmd `cmd:"" help:"Delete a calendar event."`
}

// EventsListCmd lists calendar events from a start date.
type EventsListCmd struct {
	StartDate string `help:"Start date (YYYY-MM-DD, today, 3d, +30d). Defaults to today." name:"start"`
	Limit     int    `help:"Maximum number of events to return." default:"20" short:"l"`
	Page      int    `help:"Page index for pagination." default:"1" short:"p"`
	Sort      string `help:"Sort order." default:"eventDate_asc" enum:"eventDate_asc,eventDate_desc" short:"s"`
}

func (c *EventsListCmd) Run(g *Globals) error {
	startDate := c.StartDate
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

	data, err := client.GetEvents(g.Context, date, c.Page, c.Limit, c.Sort)
	if err != nil {
		return fmt.Errorf("list events: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	events, err := parseEvents(data)
	if err != nil {
		return err
	}

	rows := formatEventListRows(events)
	header := []string{"ID", "DATE", "NAME", "TYPE", "LOCATION"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
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

	payload := map[string]any{
		"eventName": c.Name,
		"date":      date,
		"eventType": c.Type,
		"race":      c.Race,
	}

	if c.Location != "" {
		payload["location"] = c.Location
	}
	if c.Note != "" {
		payload["note"] = c.Note
	}
	if c.URL != "" {
		payload["url"] = c.URL
	}
	if c.Private {
		payload["eventPrivacy"] = map[string]string{"label": "PRIVATE"}
	}

	if c.Time != "" {
		tl := map[string]string{"startTimeHhMm": c.Time}
		if c.Timezone != "" {
			tl["timeZoneId"] = c.Timezone
		}
		payload["eventTimeLocal"] = tl
	}

	if c.Distance != "" {
		val, unit, err := parseDistance(c.Distance)
		if err != nil {
			return err
		}
		payload["completionTarget"] = map[string]any{
			"value":    val,
			"unit":     unit,
			"unitType": "distance",
		}
	}

	if c.Goal != "" || c.Primary || c.Training {
		cust := map[string]any{
			"isPrimaryEvent":  c.Primary,
			"isTrainingEvent": c.Training,
		}
		if c.Goal != "" {
			secs, err := parseGoalDuration(c.Goal)
			if err != nil {
				return err
			}
			cust["customGoal"] = map[string]any{
				"value":    secs,
				"unit":     "second",
				"unitType": "time",
			}
		}
		payload["eventCustomization"] = cust
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("build event payload: %w", err)
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.AddEvent(g.Context, json.RawMessage(raw))
	if err != nil {
		return fmt.Errorf("add event: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err == nil {
		name := jsonString(resp, "eventName")
		respDate := jsonString(resp, "date")
		id := jsonString(resp, "id")
		if id != "" {
			g.UI.Successf("Added event %q on %s (ID: %s)", name, respDate, id)
			return nil
		}
	}

	g.UI.Successf("Added event")
	return nil
}

var distanceRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*(km|mi|m)$`)

func parseDistance(s string) (float64, string, error) {
	m := distanceRe.FindStringSubmatch(s)
	if m == nil {
		return 0, "", fmt.Errorf("invalid distance %q (use e.g. 10km, 26.2mi, 400m)", s)
	}
	val, _ := strconv.ParseFloat(m[1], 64)
	units := map[string]string{"km": "kilometer", "mi": "mile", "m": "meter"}
	return val, units[m[2]], nil
}

func parseGoalDuration(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid goal duration %q (use e.g. 50m, 1h30m, 2400s)", s)
	}
	secs := int(d.Seconds())
	if secs <= 0 {
		return 0, fmt.Errorf("goal duration must be positive")
	}
	return secs, nil
}

// EventsDeleteCmd deletes a calendar event.
type EventsDeleteCmd struct {
	ID    string `arg:"" help:"Event ID."`
	Force bool   `help:"Skip confirmation prompt." short:"f"`
}

func (c *EventsDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	ok, err := confirm(os.Stderr, fmt.Sprintf("Delete event %s?", c.ID), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	if err := client.DeleteEvent(g.Context, c.ID); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	g.UI.Successf("Deleted event %s", c.ID)
	return nil
}
