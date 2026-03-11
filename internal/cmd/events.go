package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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

// EventsAddCmd adds a new calendar event from a JSON payload.
type EventsAddCmd struct {
	Params string `help:"Event JSON payload (see Garmin Connect API)." required:""`
}

func (c *EventsAddCmd) Run(g *Globals) error {
	if !json.Valid([]byte(c.Params)) {
		return fmt.Errorf("invalid JSON in --params")
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.AddEvent(g.Context, json.RawMessage(c.Params))
	if err != nil {
		return fmt.Errorf("add event: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err == nil {
		name := jsonString(resp, "eventName")
		date := jsonString(resp, "date")
		id := jsonString(resp, "id")
		if id != "" {
			g.UI.Successf("Added event %q on %s (ID: %s)", name, date, id)
			return nil
		}
	}

	g.UI.Successf("Added event")
	return nil
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
