package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/units"
)

var activityGroup = Group{Name: "activity", Help: "View activity details.", Children: []Node{
	Custom{Name: "summary", Help: "Show activity summary.", Default: true, Cmd: &ActivitySummaryCmd{}},
	Custom{Name: "details", Help: "Show full activity details.", Cmd: &ActivityDetailsCmd{}},
	Custom{Name: "splits", Help: "Show activity splits.", Cmd: &ActivitySplitsCmd{}},
	Custom{Name: "typed-splits", Help: "Show typed splits.", Cmd: &ActivityTypedSplitsCmd{}},
	Custom{Name: "split-summaries", Help: "Show split summaries.", Cmd: &ActivitySplitSummariesCmd{}},
	Custom{Name: "weather", Help: "Show weather conditions.", Cmd: &ActivityWeatherCmd{}},
	Custom{Name: "hr-zones", Help: "Show heart rate time in zones.", Cmd: &ActivityHRZonesCmd{}},
	Custom{Name: "power-zones", Help: "Show power time in zones.", Cmd: &ActivityPowerZonesCmd{}},
	activityExerciseSetsGroup,
	Custom{Name: "gear", Help: "Show linked gear.", Cmd: &ActivityGearCmd{}},
	Custom{Name: "download", Help: "Download activity file.", Cmd: &ActivityDownloadCmd{}},
	Custom{Name: "upload", Help: "Upload an activity file (FIT/GPX/TCX).", Cmd: &ActivityUploadCmd{}},
	Custom{Name: "create", Help: "Create a manual activity.", Cmd: &ActivityCreateCmd{}},
	Custom{Name: "rename", Help: "Rename an activity.", Cmd: &ActivityRenameCmd{}},
	Custom{Name: "retype", Help: "Change activity type.", Cmd: &ActivityRetypeCmd{}},
	Custom{Name: "delete", Help: "Delete an activity.", Cmd: &ActivityDeleteCmd{}},
}}

// ActivitySummaryCmd shows an activity summary.
type ActivitySummaryCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivitySummaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.get", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
	}

	var activity map[string]any
	if err := json.Unmarshal(data, &activity); err != nil {
		return fmt.Errorf("parse activity: %w", err)
	}

	cfg, err := readConfigFn()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	rows := formatActivitySummary(resolveUnits(g, client), activity, cfg.ActivitySummary)

	return outfmt.WriteRows(g.Context, g.Stdout, nil, rows)
}

// readConfigFn is the function used to read config; overridable in tests.
var readConfigFn = config.Read

// formatActivitySummary formats key activity fields as label-value rows.
// The first 3 rows (NAME, TYPE, DATE) are always shown, followed by
// type-specific fields resolved from the activity category.
// Optional overrides allow per-category field customization via config.
func formatActivitySummary(sys units.System, a map[string]any, overrides map[string][]string) [][]string {
	s := summaryDTO(a)
	category := resolveCategory(a)
	fields := resolveFields(category, overrides)

	rows := [][]string{
		{"NAME", jsonString(a, "activityName")},
		{"TYPE", activityTypeName(a)},
		{"DATE", formatDate(jsonString(s, "startTimeLocal"))},
	}
	for _, f := range fields {
		rows = append(rows, []string{f.Label, f.Format(s, sys)})
	}
	return rows
}

// summaryDTO extracts the summaryDTO sub-object from an activity.
// Falls back to the activity itself for list-endpoint responses where
// fields are at the root level.
func summaryDTO(a map[string]any) map[string]any {
	if dto, ok := a["summaryDTO"].(map[string]any); ok {
		return dto
	}
	return a
}

// activityTypeName extracts the activity type from the detail endpoint's
// activityTypeDTO.typeKey, falling back to activityType.typeKey for
// list-endpoint responses.
func activityTypeName(a map[string]any) string {
	if dto, ok := a["activityTypeDTO"].(map[string]any); ok {
		return jsonString(dto, "typeKey")
	}
	return activityTypeKey(a)
}

// formatHeartRate formats a heart rate value.
func formatHeartRate(hr float64) string {
	if hr == 0 {
		return "-"
	}
	return fmt.Sprintf("%d bpm", int(hr))
}

// ActivityDetailsCmd shows full activity details including charts and polylines.
type ActivityDetailsCmd struct {
	ID       string `arg:"" help:"Activity ID."`
	MaxChart int    `help:"Maximum chart data points." default:"2000" name:"max-chart"`
	MaxPoly  int    `help:"Maximum polyline data points." default:"4000" name:"max-poly"`
}

func (c *ActivityDetailsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.details", map[string]string{"activityID": c.ID, "maxChart": strconv.Itoa(c.MaxChart), "maxPoly": strconv.Itoa(c.MaxPoly)}, nil)
	if err != nil {
		return fmt.Errorf("get activity details: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// ActivitySplitsCmd shows activity splits.
type ActivitySplitsCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivitySplitsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.splits", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity splits: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// ActivityTypedSplitsCmd shows typed splits for an activity.
type ActivityTypedSplitsCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityTypedSplitsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.typed-splits", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity typed splits: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// ActivitySplitSummariesCmd shows split summaries for an activity.
type ActivitySplitSummariesCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivitySplitSummariesCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.split-summaries", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity split summaries: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// ActivityWeatherCmd shows weather conditions for an activity.
type ActivityWeatherCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityWeatherCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.weather", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity weather: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// ActivityHRZonesCmd shows heart rate time-in-zone data for an activity.
type ActivityHRZonesCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityHRZonesCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.hr-zones", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity HR zones: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// ActivityPowerZonesCmd shows power time-in-zone data for an activity.
type ActivityPowerZonesCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityPowerZonesCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.power-zones", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity power zones: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

var activityExerciseSetsGroup = Group{Name: "exercise-sets", Help: "Manage exercise sets.", Children: []Node{
	Custom{Name: "show", Help: "Show exercise sets.", Default: true, Cmd: &ActivityExerciseSetsShowCmd{}},
	Custom{Name: "set", Help: "Set exercise sets for a strength training activity.", Cmd: &ActivityExerciseSetsSetCmd{}},
}}

// ActivityExerciseSetsShowCmd shows exercise set data for strength training activities.
type ActivityExerciseSetsShowCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityExerciseSetsShowCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.exercise-sets", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity exercise sets: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// ActivityGearCmd shows gear linked to an activity.
type ActivityGearCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityGearCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.gear", map[string]string{"activityID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get activity gear: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}
