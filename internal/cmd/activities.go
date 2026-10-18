package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/bpauli/gccli/internal/outfmt"
)

var activityColumns = []Column{
	{Label: "ID", Path: "activityId"},
	{Label: "DATE", Path: "startTimeLocal", Format: DateOnly},
	{Label: "TYPE", Path: "activityType.typeKey"},
	{Label: "NAME", Path: "activityName"},
	{Label: "DISTANCE", Path: "distance", Format: Distance},
	{Label: "DURATION", Path: "duration", Format: Duration},
	{Label: "CALORIES", Path: "calories", Format: Count},
}

var activitiesGroup = Group{Name: "activities", Help: "List and search activities.", Children: []Node{
	Leaf{
		Name: "list", Help: "List recent activities.", Default: true,
		Spec: argsOf(&struct {
			Pagination `embed:"" set:"items=activities"`
			Type       string `help:"Filter by activity type (e.g. running, cycling)." short:"t"`
		}{}),
		Endpoint: "activities.list",
		Params:   Params{"start": In("Offset"), "limit": In("Limit"), "activityType": In("Type")},
		Verb:     "list activities",
		Output:   Rows{Noun: "activities", Columns: activityColumns},
	},
	Custom{Name: "count", Help: "Show total activity count.", Cmd: &ActivitiesCountCmd{}},
	Leaf{
		Name: "search", Help: "Search activities by date range.",
		Spec: argsOf(&struct {
			Pagination `embed:"" set:"items=activities"`
			StartDate  Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d)." required:"" name:"start"`
			EndDate    Date `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d)." name:"end"`
		}{}),
		Endpoint: "activities.search",
		Params:   Params{"start": In("Offset"), "limit": In("Limit"), "startDate": In("StartDate"), "endDate": In("EndDate")},
		Verb:     "search activities",
		Output:   Rows{Noun: "activities", Columns: activityColumns},
	},
	Custom{Name: "types", Help: "List all activity types supported by Garmin.", Cmd: &ActivitiesTypesCmd{}},
	Leaf{Name: "pending", Help: "List pending activities.", Endpoint: "activities.pending", Verb: "get pending activities", Output: JSON{}},
}}

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
		return outfmt.WriteJSON(g.Stdout, map[string]int{"count": count})
	}

	_, _ = fmt.Fprintf(g.Stdout, "%d\n", count)
	return nil
}

// jsonString extracts a string (or stringified number) from a JSON map.
func jsonString(m map[string]any, key string) string {
	return textOf(m[key])
}

// jsonFloat extracts a float64 from a JSON map.
func jsonFloat(m map[string]any, key string) float64 {
	return numberOf(m[key])
}

// jsonBool extracts a bool from a JSON map. The second return value reports
// whether the key was present and held a bool, distinguishing "false" from
// "missing" or "null" (which both render as "-" in tables).
func jsonBool(m map[string]any, key string) (bool, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// formatDate extracts just the date portion from a datetime string like "2024-01-15 08:30:00".
func formatDate(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.SplitN(s, " ", 2)
	return parts[0]
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

// ActivitiesTypesCmd lists all activity types supported by Garmin.
type ActivitiesTypesCmd struct{}

func (c *ActivitiesTypesCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.types", nil, nil)
	if err != nil {
		return fmt.Errorf("get activity types: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
	}

	types, err := parseActivityTypes(data)
	if err != nil {
		return err
	}

	rows := formatActivityTypeRows(types)
	header := activityTypesHeader()

	return outfmt.WriteRows(g.Context, g.Stdout, header, rows)
}

func activityTypesHeader() []string {
	return []string{"ID", "TYPE KEY", "PARENT ID", "TRIMMABLE", "HIDDEN", "RESTRICTED"}
}

// parseActivityTypes unmarshals the activity types payload and sorts by typeId
// so table/plain output is stable across runs.
func parseActivityTypes(data json.RawMessage) ([]map[string]any, error) {
	var types []map[string]any
	if err := json.Unmarshal(data, &types); err != nil {
		return nil, fmt.Errorf("parse activity types: %w", err)
	}
	sort.SliceStable(types, func(i, j int) bool {
		return jsonFloat(types[i], "typeId") < jsonFloat(types[j], "typeId")
	})
	return types, nil
}

// formatActivityTypeRows extracts table rows from activity-type data.
func formatActivityTypeRows(types []map[string]any) [][]string {
	rows := make([][]string, 0, len(types))
	for _, t := range types {
		rows = append(rows, []string{
			fmt.Sprintf("%d", int(jsonFloat(t, "typeId"))),
			jsonString(t, "typeKey"),
			formatParentID(t),
			formatYesNo(t, "trimmable"),
			formatYesNo(t, "isHidden"),
			formatYesNo(t, "restricted"),
		})
	}
	return rows
}

// formatParentID formats parentTypeId (handles null as "-").
func formatParentID(t map[string]any) string {
	v, ok := t["parentTypeId"]
	if !ok || v == nil {
		return "-"
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%d", int(f))
	}
	return "-"
}

// formatYesNo renders a bool field as "yes"/"no", or "-" when missing.
func formatYesNo(m map[string]any, key string) string {
	b, ok := jsonBool(m, key)
	if !ok {
		return "-"
	}
	if b {
		return "yes"
	}
	return "no"
}
