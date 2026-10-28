package cmd

import (
	"fmt"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/units"
)

var activityGroup = Group{Name: "activity", Help: "View activity details.", Children: []Node{
	Custom{Name: "summary", Help: "Show activity summary.", Default: true, Cmd: &ActivitySummaryCmd{}},
	Leaf{
		Name: "details", Help: "Show full activity details.",
		Spec: argsOf(&struct {
			ID       ID  `arg:"" help:"Activity ID."`
			MaxChart int `help:"Maximum chart data points." default:"2000" name:"max-chart"`
			MaxPoly  int `help:"Maximum polyline data points." default:"4000" name:"max-poly"`
		}{}),
		Endpoint: "activities.details",
		Params:   Params{"activityID": In("ID"), "maxChart": In("MaxChart"), "maxPoly": In("MaxPoly")},
		Verb:     "get activity details", Output: JSON{},
	},
	Leaf{
		Name: "splits", Help: "Show activity splits.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.splits",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity splits", Output: JSON{},
	},
	Leaf{
		Name: "typed-splits", Help: "Show typed splits.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.typed-splits",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity typed splits", Output: JSON{},
	},
	Leaf{
		Name: "split-summaries", Help: "Show split summaries.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.split-summaries",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity split summaries", Output: JSON{},
	},
	Leaf{
		Name: "weather", Help: "Show weather conditions.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.weather",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity weather", Output: JSON{},
	},
	Leaf{
		Name: "hr-zones", Help: "Show heart rate time in zones.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.hr-zones",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity HR zones", Output: JSON{List: true},
	},
	Leaf{
		Name: "power-zones", Help: "Show power time in zones.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.power-zones",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity power zones", Output: JSON{List: true},
	},
	Leaf{
		Name: "segments", Help: "List the segments that an activity crossed.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "segments.activity",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity segments",
		Output: Rows{Noun: "segments", Columns: []Column{
			{Label: "UUID", Path: "segmentUuid"},
			{Label: "NAME", Path: "segmentName"},
			{Label: "DISTANCE", Path: "segmentDistance", Format: Kilometers},
		}},
	},
	activityExerciseSetsGroup,
	Leaf{
		Name: "gear", Help: "Show linked gear.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.gear",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity gear", Output: JSON{},
	},
	activityCommentsGroup,
	activityLikesGroup,
	Custom{Name: "download", Help: "Download activity file.", Cmd: &ActivityDownloadCmd{}},
	Custom{Name: "upload", Help: "Upload an activity file (FIT/GPX/TCX).", Cmd: &ActivityUploadCmd{}},
	Custom{Name: "create", Help: "Create a manual activity.", Cmd: &ActivityCreateCmd{}},
	Custom{Name: "rename", Help: "Rename an activity.", Cmd: &ActivityRenameCmd{}},
	Custom{Name: "retype", Help: "Change activity type.", Cmd: &ActivityRetypeCmd{}},
	Custom{Name: "describe", Help: "Set or clear the description of an activity.", Cmd: &ActivityDescribeCmd{}},
	Custom{Name: "privacy", Help: "Change who can see an activity.", Cmd: &ActivityPrivacyCmd{}},
	Custom{Name: "trim", Help: "Trim the start or end of an activity.", Cmd: &ActivityTrimCmd{}},
	Custom{Name: "restore", Help: "Restore the original of a trimmed activity.", Cmd: &ActivityRestoreCmd{}},
	Custom{Name: "favorite", Help: "Mark an activity as a favorite.", Cmd: &ActivityFavoriteCmd{}},
	Custom{Name: "unfavorite", Help: "Remove the favorite mark from an activity.", Cmd: &ActivityUnfavoriteCmd{}},
	Leaf{
		Name: "delete", Help: "Delete an activity.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.delete",
		Params:   Params{"activityID": In("ID")},
		Verb:     "delete activity",
		Output:   Write{Done: "Deleted activity {ID}", Confirm: "Delete activity {ID}?"},
	},
}}

// ActivitySummaryCmd shows an activity summary.
type ActivitySummaryCmd struct {
	ID ID `arg:"" help:"Activity ID."`
}

func (c *ActivitySummaryCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "activities.get", map[string]string{"activityID": string(c.ID)}, nil)
	if err != nil {
		return fmt.Errorf("get activity: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return writeObject(g, data)
	}

	var activity map[string]any
	if err := unmarshalAnswer(data, &activity); err != nil {
		return fmt.Errorf("parse activity: %w", err)
	}
	if activity == nil {
		return nil
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
	return textOf(valueAt(a, "activityType.typeKey"))
}

// formatHeartRate formats a heart rate value.
func formatHeartRate(hr float64) string {
	if hr == 0 {
		return "-"
	}
	return fmt.Sprintf("%d bpm", int(hr))
}

var activityExerciseSetsGroup = Group{Name: "exercise-sets", Help: "Manage exercise sets.", Children: []Node{
	Leaf{
		Name: "show", Help: "Show exercise sets.", Default: true,
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.exercise-sets",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get activity exercise sets", Output: JSON{},
	},
	Custom{Name: "set", Help: "Set exercise sets for a strength training activity.", Cmd: &ActivityExerciseSetsSetCmd{}},
}}
