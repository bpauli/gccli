package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

var workoutsGroup = Group{Name: "workouts", Help: "Manage workouts.", Children: []Node{
	Custom{Name: "list", Help: "List workouts.", Default: true, Cmd: &WorkoutsListCmd{}},
	Custom{Name: "detail", Help: "View workout details.", Cmd: &WorkoutDetailCmd{}},
	Custom{Name: "download", Help: "Download workout as FIT file.", Cmd: &WorkoutDownloadCmd{}},
	Custom{Name: "upload", Help: "Upload workout from JSON file.", Cmd: &WorkoutUploadCmd{}},
	Custom{Name: "update", Help: "Update an existing workout from a JSON file.", Cmd: &WorkoutUpdateCmd{}},
	Custom{Name: "create", Help: "Create a workout with sport type and optional targets.", Cmd: &WorkoutCreateCmd{}},
	Custom{Name: "copy", Help: "Copy a workout.", Cmd: &WorkoutCopyCmd{}},
	Leaf{Name: "types", Help: "List workout types.", Endpoint: "workouts.types", Verb: "get workout types", Output: JSON{}},
	workoutScheduleGroup,
	workoutBenchmarksGroup,
	Leaf{
		Name: "delete", Help: "Delete a workout.",
		Spec: argsOf(&struct {
			ID string `arg:"" help:"Workout ID."`
		}{}),
		Endpoint: "workouts.delete",
		Params:   Params{"workoutID": In("ID")},
		Verb:     "delete workout",
		Output:   Write{Done: "Deleted workout {ID}", Confirm: "Delete workout {ID}?"},
	},
}}

// WorkoutsListCmd lists workouts.
type WorkoutsListCmd struct {
	Pagination `embed:"" set:"items=workouts"`
}

func (c *WorkoutsListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "workouts.list", map[string]string{"start": strconv.Itoa(c.Offset), "limit": strconv.Itoa(c.Limit)}, nil)
	if err != nil {
		return fmt.Errorf("list workouts: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
	}

	workouts, err := parseWorkouts(data)
	if err != nil {
		return err
	}

	rows := formatWorkoutRows(workouts)
	header := []string{"ID", "NAME", "TYPE", "OWNER"}

	return outfmt.WriteRows(g.Context, g.Stdout, header, rows)
}

// WorkoutDetailCmd shows workout details.
type WorkoutDetailCmd struct {
	ID string `arg:"" help:"Workout ID."`
}

func (c *WorkoutDetailCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "workouts.get", map[string]string{"workoutID": c.ID}, nil)
	if err != nil {
		return fmt.Errorf("get workout: %w", err)
	}

	return outfmt.WriteJSON(g.Stdout, json.RawMessage(data))
}

// WorkoutDownloadCmd downloads a workout as FIT.
type WorkoutDownloadCmd struct {
	ID     string `arg:"" help:"Workout ID."`
	Output string `help:"Output file path (default: workout_{id}.fit)." short:"o"`
}

func (c *WorkoutDownloadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.DownloadWorkout(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("download workout: %w", err)
	}

	outPath := c.Output
	if outPath == "" {
		outPath = fmt.Sprintf("workout_%s.fit", c.ID)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	g.UI.Successf("Downloaded %s (%d bytes)", outPath, len(data))
	return nil
}

// WorkoutUploadCmd uploads a workout from a JSON file.
type WorkoutUploadCmd struct {
	File string `arg:"" help:"Path to workout JSON file." type:"existingfile"`
}

func (c *WorkoutUploadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	jsonData, err := os.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Validate that the file contains valid JSON.
	if !json.Valid(jsonData) {
		return fmt.Errorf("invalid JSON in %s", c.File)
	}

	data, err := client.Call(g.Context, "workouts.upload", nil, jsonData)
	if err != nil {
		return fmt.Errorf("upload workout: %w", err)
	}

	return reportWrite(g, data, "Uploaded workout from %s", c.File)
}

// WorkoutUpdateCmd updates an existing workout from a JSON file.
type WorkoutUpdateCmd struct {
	ID   string `arg:"" help:"Workout ID to update."`
	File string `arg:"" help:"Path to workout JSON file." type:"existingfile"`
}

func (c *WorkoutUpdateCmd) Run(g *Globals) error {
	if _, err := strconv.ParseInt(c.ID, 10, 64); err != nil {
		return fmt.Errorf("invalid workout ID: %w", err)
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	jsonData, err := os.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	payload, replaced, err := setWorkoutID(jsonData, c.ID)
	if err != nil {
		return fmt.Errorf("%s: %w", c.File, err)
	}
	if replaced != "" {
		g.UI.Warnf("Workout JSON has workoutId %s, updating workout %s instead", replaced, c.ID)
	}

	data, err := client.Call(g.Context, "workouts.update", map[string]string{"workoutID": c.ID}, payload)
	if err != nil {
		return fmt.Errorf("update workout: %w", err)
	}

	return reportWrite(g, data, "Updated workout %s from %s", c.ID, c.File)
}

// WorkoutCopyCmd copies a workout.
type WorkoutCopyCmd struct {
	ID   string `arg:"" help:"Workout ID."`
	Name string `arg:"" optional:"" help:"Name of the copy. Without it, the copy gets the name of the original."`
}

func (c *WorkoutCopyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.CopyWorkout(g.Context, c.ID, c.Name)
	if err != nil {
		return fmt.Errorf("copy workout: %w", err)
	}

	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "workoutId"); id != "" {
		return reportWrite(g, data, "Copied workout %s to %q (ID: %s)", c.ID, jsonString(resp, "workoutName"), id)
	}
	return reportWrite(g, data, "Copied workout %s", c.ID)
}

// setWorkoutID returns the payload with workoutId set to workoutID. Garmin
// resolves the update by the ID in the request path, so the body must carry the
// same one. A payload written for "workouts upload" has no workoutId at all.
// The second return value holds the ID that was in the payload when it differed
// from workoutID, so the caller can warn about the mismatch.
func setWorkoutID(payload []byte, workoutID string) (json.RawMessage, string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return nil, "", fmt.Errorf("invalid JSON workout object: %w", err)
	}

	var replaced string
	if existing, ok := fields["workoutId"]; ok {
		if id := strings.TrimSpace(strings.Trim(string(existing), `"`)); id != workoutID && id != "null" {
			replaced = id
		}
	}

	fields["workoutId"] = json.RawMessage(workoutID)
	updated, err := json.Marshal(fields)
	if err != nil {
		return nil, "", fmt.Errorf("marshal workout: %w", err)
	}
	return updated, replaced, nil
}

var workoutScheduleGroup = Group{Name: "schedule", Help: "Manage scheduled workouts.", Children: []Node{
	Custom{Name: "add", Help: "Add a workout to the schedule.", Cmd: &WorkoutScheduleAddCmd{}},
	Custom{Name: "list", Help: "List scheduled workouts for a date.", Cmd: &WorkoutScheduleListCmd{}},
	Leaf{
		Name: "remove", Help: "Remove a scheduled workout from the calendar.",
		Spec: argsOf(&struct {
			ID string `arg:"" help:"Schedule ID (from 'workouts schedule list')."`
		}{}),
		Endpoint: "workouts.unschedule",
		Params:   Params{"scheduleID": In("ID")},
		Verb:     "remove scheduled workout",
		Output:   Write{Done: "Removed scheduled workout {ID}", Confirm: "Remove scheduled workout {ID}?"},
	},
	Leaf{
		Name: "export", Help: "Export the workout schedule from a start date.",
		Spec: argsOf(&struct {
			StartDate Date `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
		}{}),
		Endpoint: "workouts.export-schedule",
		Params:   Params{"startDate": In("StartDate").Or(Today)},
		Verb:     "export workout schedule", Output: JSON{},
	},
}}

var workoutBenchmarksGroup = Group{Name: "benchmarks", Help: "View strength benchmarks.", Children: []Node{
	Leaf{Name: "list", Help: "List strength benchmarks.", Default: true, Endpoint: "workouts.benchmarks", Verb: "get benchmarks", Output: JSON{}},
}}

// WorkoutScheduleAddCmd schedules a workout on a calendar date.
type WorkoutScheduleAddCmd struct {
	ID   string `arg:"" help:"Workout ID."`
	Date Date   `arg:"" help:"Date to schedule (YYYY-MM-DD, today, yesterday, 3d)."`
}

func (c *WorkoutScheduleAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date := string(c.Date)
	body, err := json.Marshal(map[string]string{"date": date})
	if err != nil {
		return fmt.Errorf("schedule workout: %w", err)
	}
	data, err := client.Call(g.Context, "workouts.schedule", map[string]string{"workoutID": c.ID}, body)
	if err != nil {
		return fmt.Errorf("schedule workout: %w", err)
	}

	return reportWrite(g, data, "Scheduled workout %s on %s", c.ID, date)
}

// WorkoutScheduleListCmd lists scheduled workouts for a date or date range.
type WorkoutScheduleListCmd struct {
	Date      Date `arg:"" optional:"" default:"today" help:"Date to list (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
	StartDate Date `help:"Start date for range (YYYY-MM-DD, today, yesterday, 3d, +30d)." name:"start"`
	EndDate   Date `help:"End date for range (YYYY-MM-DD, today, yesterday, 3d, +30d)." name:"end"`
}

func (c *WorkoutScheduleListCmd) Run(g *Globals) error {
	if c.StartDate != "" || c.EndDate != "" {
		return c.runRange(g)
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date := string(c.Date)
	data, err := client.GetCalendarWeek(g.Context, date)
	if err != nil {
		return fmt.Errorf("list scheduled workouts: %w", err)
	}

	items, err := filterCalendarWorkouts(data, date)
	if err != nil {
		return err
	}

	return writeScheduledWorkouts(g, items)
}

func (c *WorkoutScheduleListCmd) runRange(g *Globals) error {
	if c.StartDate == "" || c.EndDate == "" {
		return fmt.Errorf("both --start and --end are required for range queries")
	}

	startStr, endStr := string(c.StartDate), string(c.EndDate)
	start, _ := time.Parse("2006-01-02", startStr)
	end, _ := time.Parse("2006-01-02", endStr)
	if end.Before(start) {
		return fmt.Errorf("--end %s is before --start %s", endStr, startStr)
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	var allWorkouts []map[string]any
	seen := make(map[string]bool)

	for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 0, 7) {
		date := cursor.Format("2006-01-02")
		data, err := client.GetCalendarWeek(g.Context, date)
		if err != nil {
			return fmt.Errorf("list scheduled workouts: %w", err)
		}

		items, err := filterCalendarWorkoutsInRange(data, startStr, endStr)
		if err != nil {
			return err
		}

		for _, item := range items {
			key := jsonString(item, "id") + ":" + jsonString(item, "date")
			if !seen[key] {
				seen[key] = true
				allWorkouts = append(allWorkouts, item)
			}
		}
	}

	return writeScheduledWorkouts(g, allWorkouts)
}

func writeScheduledWorkouts(g *Globals, items []map[string]any) error {
	if outfmt.IsJSON(g.Context) {
		out, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal calendar items: %w", err)
		}
		return outfmt.WriteJSON(g.Stdout, json.RawMessage(out))
	}

	rows := formatCalendarWorkoutRows(items)
	header := []string{"SCHEDULE ID", "WORKOUT ID", "TITLE", "SPORT", "DATE"}

	return outfmt.WriteRows(g.Context, g.Stdout, header, rows)
}

// filterCalendarWorkouts extracts workout items matching the given date from calendar data.
func filterCalendarWorkouts(data json.RawMessage, date string) ([]map[string]any, error) {
	var calendar map[string]any
	if err := json.Unmarshal(data, &calendar); err != nil {
		return nil, fmt.Errorf("parse calendar: %w", err)
	}

	rawItems, ok := calendar["calendarItems"]
	if !ok {
		return nil, nil
	}

	items, ok := rawItems.([]any)
	if !ok {
		return nil, nil
	}

	var workouts []map[string]any
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if jsonString(m, "itemType") != "workout" {
			continue
		}
		if jsonString(m, "date") != date {
			continue
		}
		workouts = append(workouts, m)
	}
	return workouts, nil
}

// filterCalendarWorkoutsInRange extracts workout items within a date range from calendar data.
func filterCalendarWorkoutsInRange(data json.RawMessage, startDate, endDate string) ([]map[string]any, error) {
	var calendar map[string]any
	if err := json.Unmarshal(data, &calendar); err != nil {
		return nil, fmt.Errorf("parse calendar: %w", err)
	}

	rawItems, ok := calendar["calendarItems"]
	if !ok {
		return nil, nil
	}

	items, ok := rawItems.([]any)
	if !ok {
		return nil, nil
	}

	var workouts []map[string]any
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if jsonString(m, "itemType") != "workout" {
			continue
		}
		date := jsonString(m, "date")
		if date < startDate || date > endDate {
			continue
		}
		workouts = append(workouts, m)
	}
	return workouts, nil
}

// formatCalendarWorkoutRows extracts table rows from calendar workout items.
func formatCalendarWorkoutRows(items []map[string]any) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			jsonString(item, "id"),
			jsonString(item, "workoutId"),
			jsonString(item, "title"),
			jsonString(item, "sportTypeKey"),
			jsonString(item, "date"),
		})
	}
	return rows
}

// parseWorkouts unmarshals workout list data.
// The Garmin API may return either an array directly or an object with a nested array.
func parseWorkouts(data json.RawMessage) ([]map[string]any, error) {
	// Try as array first.
	var workouts []map[string]any
	if err := json.Unmarshal(data, &workouts); err == nil {
		return workouts, nil
	}

	// Try as object with nested array.
	var wrapper map[string]any
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse workouts: %w", err)
	}

	// Common wrapper keys from Garmin API.
	for _, key := range []string{"workouts", "results"} {
		if arr, ok := wrapper[key]; ok {
			if items, ok := arr.([]any); ok {
				workouts = make([]map[string]any, 0, len(items))
				for _, item := range items {
					if m, ok := item.(map[string]any); ok {
						workouts = append(workouts, m)
					}
				}
				return workouts, nil
			}
		}
	}

	return nil, fmt.Errorf("parse workouts: unexpected response format")
}

// formatWorkoutRows extracts table rows from workout data.
func formatWorkoutRows(workouts []map[string]any) [][]string {
	rows := make([][]string, 0, len(workouts))
	for _, w := range workouts {
		rows = append(rows, []string{
			jsonString(w, "workoutId"),
			jsonString(w, "workoutName"),
			workoutSportType(w),
			workoutOwner(w),
		})
	}
	return rows
}

// workoutSportType extracts the sport type from nested sportType.sportTypeKey.
func workoutSportType(w map[string]any) string {
	st, ok := w["sportType"]
	if !ok || st == nil {
		return ""
	}
	if m, ok := st.(map[string]any); ok {
		return jsonString(m, "sportTypeKey")
	}
	return ""
}

// workoutOwner extracts the owner display name from nested owner.displayName.
func workoutOwner(w map[string]any) string {
	o, ok := w["owner"]
	if !ok || o == nil {
		return ""
	}
	if m, ok := o.(map[string]any); ok {
		return jsonString(m, "displayName")
	}
	return ""
}
