package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bpauli/gccli/internal/outfmt"
)

// ActivitiesCmd groups activity listing and search subcommands.
type ActivitiesCmd struct {
	List   ActivitiesListCmd   `cmd:"" default:"withargs" help:"List recent activities."`
	Count  ActivitiesCountCmd  `cmd:"" help:"Show total activity count."`
	Search ActivitiesSearchCmd `cmd:"" help:"Search activities by date range."`
}

// ActivitiesListCmd lists recent activities.
type ActivitiesListCmd struct {
	Limit int    `help:"Maximum number of activities to return." default:"20" short:"l"`
	Start int    `help:"Start index for pagination." default:"0" short:"s"`
	Type  string `help:"Filter by activity type (e.g. running, cycling)." short:"t"`
}

func (c *ActivitiesListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetActivities(g.Context, c.Start, c.Limit, c.Type)
	if err != nil {
		return fmt.Errorf("list activities: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	activities, err := parseActivities(data)
	if err != nil {
		return err
	}

	rows := formatActivityRows(activities)
	header := activityHeader()

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
}

// ActivitiesCountCmd shows the total activity count.
type ActivitiesCountCmd struct{}

func (c *ActivitiesCountCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	count, err := client.CountActivities(g.Context)
	if err != nil {
		return fmt.Errorf("count activities: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, map[string]int{"count": count})
	}

	_, _ = fmt.Fprintf(os.Stdout, "%d\n", count)
	return nil
}

// ActivitiesSearchCmd searches activities by date range.
type ActivitiesSearchCmd struct {
	Limit     int    `help:"Maximum number of activities to return." default:"20" short:"l"`
	Start     int    `help:"Start index for pagination." default:"0" short:"s"`
	StartDate string `help:"Start date (YYYY-MM-DD)." required:"" name:"start-date"`
	EndDate   string `help:"End date (YYYY-MM-DD)." name:"end-date"`
}

func (c *ActivitiesSearchCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.SearchActivities(g.Context, c.Start, c.Limit, c.StartDate, c.EndDate)
	if err != nil {
		return fmt.Errorf("search activities: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	activities, err := parseActivities(data)
	if err != nil {
		return err
	}

	rows := formatActivityRows(activities)
	header := activityHeader()

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
}

func activityHeader() []string {
	return []string{"ID", "DATE", "TYPE", "NAME", "DISTANCE", "DURATION", "CALORIES"}
}

// parseActivities unmarshals a JSON array of activities into a slice of maps.
func parseActivities(data json.RawMessage) ([]map[string]any, error) {
	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		return nil, fmt.Errorf("parse activities: %w", err)
	}
	return activities, nil
}

// formatActivityRows extracts table rows from activity data.
func formatActivityRows(activities []map[string]any) [][]string {
	rows := make([][]string, 0, len(activities))
	for _, a := range activities {
		rows = append(rows, []string{
			jsonString(a, "activityId"),
			formatDate(jsonString(a, "startTimeLocal")),
			activityTypeKey(a),
			jsonString(a, "activityName"),
			formatDistance(jsonFloat(a, "distance")),
			formatDuration(jsonFloat(a, "duration")),
			formatCalories(jsonFloat(a, "calories")),
		})
	}
	return rows
}

// jsonString extracts a string (or stringified number) from a JSON map.
func jsonString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// jsonFloat extracts a float64 from a JSON map.
func jsonFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

// activityTypeKey extracts the activity type key from nested activityType.typeKey.
func activityTypeKey(a map[string]any) string {
	at, ok := a["activityType"]
	if !ok || at == nil {
		return ""
	}
	if m, ok := at.(map[string]any); ok {
		return jsonString(m, "typeKey")
	}
	return ""
}

// formatDate extracts just the date portion from a datetime string like "2024-01-15 08:30:00".
func formatDate(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.SplitN(s, " ", 2)
	return parts[0]
}

// formatDistance converts meters to km with 2 decimal places.
func formatDistance(meters float64) string {
	if meters == 0 {
		return "-"
	}
	km := meters / 1000.0
	return fmt.Sprintf("%.2f km", km)
}

// formatDuration converts seconds to HH:MM:SS.
func formatDuration(seconds float64) string {
	if seconds == 0 {
		return "-"
	}
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// formatCalories formats calorie count.
func formatCalories(cal float64) string {
	if cal == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", int(cal))
}
