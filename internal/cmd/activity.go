package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/outfmt"
)

// ActivityCmd groups activity detail subcommands.
type ActivityCmd struct {
	Summary        ActivitySummaryCmd        `cmd:"" default:"withargs" help:"Show activity summary."`
	Details        ActivityDetailsCmd        `cmd:"" help:"Show full activity details."`
	Splits         ActivitySplitsCmd         `cmd:"" help:"Show activity splits."`
	TypedSplits    ActivityTypedSplitsCmd    `cmd:"" name:"typed-splits" help:"Show typed splits."`
	SplitSummaries ActivitySplitSummariesCmd `cmd:"" name:"split-summaries" help:"Show split summaries."`
	Weather        ActivityWeatherCmd        `cmd:"" help:"Show weather conditions."`
	HRZones        ActivityHRZonesCmd        `cmd:"" name:"hr-zones" help:"Show heart rate time in zones."`
	PowerZones     ActivityPowerZonesCmd     `cmd:"" name:"power-zones" help:"Show power time in zones."`
	ExerciseSets   ActivityExerciseSetsGroup `cmd:"" name:"exercise-sets" help:"Manage exercise sets."`
	Gear           ActivityGearCmd           `cmd:"" help:"Show linked gear."`
	Download       ActivityDownloadCmd       `cmd:"" help:"Download activity file."`
	Upload         ActivityUploadCmd         `cmd:"" help:"Upload an activity file (FIT/GPX/TCX)."`
	Create         ActivityCreateCmd         `cmd:"" help:"Create a manual activity."`
	Rename         ActivityRenameCmd         `cmd:"" help:"Rename an activity."`
	Retype         ActivityRetypeCmd         `cmd:"" help:"Change activity type."`
	Delete         ActivityDeleteCmd         `cmd:"" help:"Delete an activity."`
}

// ActivitySummaryCmd shows an activity summary.
type ActivitySummaryCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivitySummaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetActivity(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	var activity map[string]any
	if err := json.Unmarshal(data, &activity); err != nil {
		return fmt.Errorf("parse activity: %w", err)
	}

	cfg, err := readConfigFn()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	rows := formatActivitySummary(activity, cfg.ActivitySummary)

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, nil, rows)
}

// readConfigFn is the function used to read config; overridable in tests.
var readConfigFn = config.Read

// formatActivitySummary formats key activity fields as label-value rows.
// The first 3 rows (NAME, TYPE, DATE) are always shown, followed by
// type-specific fields resolved from the activity category.
// Optional overrides allow per-category field customization via config.
func formatActivitySummary(a map[string]any, overrides map[string][]string) [][]string {
	s := summaryDTO(a)
	category := resolveCategory(a)
	fields := resolveFields(category, overrides)

	rows := [][]string{
		{"NAME", jsonString(a, "activityName")},
		{"TYPE", activityTypeName(a)},
		{"DATE", formatDate(jsonString(s, "startTimeLocal"))},
	}
	for _, f := range fields {
		rows = append(rows, []string{f.Label, f.Format(s, a)})
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

	data, err := client.GetActivityDetails(g.Context, c.ID, c.MaxChart, c.MaxPoly)
	if err != nil {
		return fmt.Errorf("get activity details: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.GetActivitySplits(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity splits: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.GetActivityTypedSplits(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity typed splits: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.GetActivitySplitSummaries(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity split summaries: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.GetActivityWeather(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity weather: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.GetActivityHRZones(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity HR zones: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.GetActivityPowerZones(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity power zones: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
}

// ActivityExerciseSetsGroup groups exercise-sets subcommands.
type ActivityExerciseSetsGroup struct {
	Show ActivityExerciseSetsShowCmd `cmd:"" default:"withargs" help:"Show exercise sets."`
	Set  ActivityExerciseSetsSetCmd  `cmd:"" help:"Set exercise sets for a strength training activity."`
}

// ActivityExerciseSetsShowCmd shows exercise set data for strength training activities.
type ActivityExerciseSetsShowCmd struct {
	ID string `arg:"" help:"Activity ID."`
}

func (c *ActivityExerciseSetsShowCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetActivityExerciseSets(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity exercise sets: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.GetActivityGear(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get activity gear: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
}
