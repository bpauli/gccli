package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/bpauli/gccli/internal/outfmt"
)

// WorkoutsCmd groups workout subcommands.
type WorkoutsCmd struct {
	List     WorkoutsListCmd    `cmd:"" default:"withargs" help:"List workouts."`
	Detail   WorkoutDetailCmd   `cmd:"" help:"View workout details."`
	Download WorkoutDownloadCmd `cmd:"" help:"Download workout as FIT file."`
	Upload   WorkoutUploadCmd   `cmd:"" help:"Upload workout from JSON file."`
	Create   WorkoutCreateCmd   `cmd:"" help:"Create a workout with sport type and optional targets."`
	Schedule WorkoutScheduleCmd `cmd:"" help:"Manage scheduled workouts."`
	Delete   WorkoutDeleteCmd   `cmd:"" help:"Delete a workout."`
}

// WorkoutsListCmd lists workouts.
type WorkoutsListCmd struct {
	Limit int `help:"Maximum number of workouts to return." default:"20" short:"l"`
	Start int `help:"Start index for pagination." default:"0" short:"s"`
}

func (c *WorkoutsListCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetWorkouts(g.Context, c.Start, c.Limit)
	if err != nil {
		return fmt.Errorf("list workouts: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	workouts, err := parseWorkouts(data)
	if err != nil {
		return err
	}

	rows := formatWorkoutRows(workouts)
	header := []string{"ID", "NAME", "TYPE", "OWNER"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
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

	data, err := client.GetWorkout(g.Context, c.ID)
	if err != nil {
		return fmt.Errorf("get workout: %w", err)
	}

	return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
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

	data, err := client.UploadWorkout(g.Context, jsonData)
	if err != nil {
		return fmt.Errorf("upload workout: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Uploaded workout from %s", c.File)
	return nil
}

// WorkoutScheduleCmd groups schedule subcommands.
type WorkoutScheduleCmd struct {
	Add    WorkoutScheduleAddCmd    `cmd:"" help:"Add a workout to the schedule."`
	List   WorkoutScheduleListCmd   `cmd:"" help:"List scheduled workouts for a date."`
	Remove WorkoutScheduleRemoveCmd `cmd:"" help:"Remove a scheduled workout from the calendar."`
}

var dateRegexp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// WorkoutScheduleAddCmd schedules a workout on a calendar date.
type WorkoutScheduleAddCmd struct {
	ID   string `arg:"" help:"Workout ID."`
	Date string `arg:"" help:"Date to schedule (YYYY-MM-DD)."`
}

func (c *WorkoutScheduleAddCmd) Run(g *Globals) error {
	if !dateRegexp.MatchString(c.Date) {
		return fmt.Errorf("invalid date format %q: expected YYYY-MM-DD", c.Date)
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.ScheduleWorkout(g.Context, c.ID, c.Date)
	if err != nil {
		return fmt.Errorf("schedule workout: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Scheduled workout %s on %s", c.ID, c.Date)
	return nil
}

// WorkoutScheduleListCmd lists scheduled workouts for a date.
type WorkoutScheduleListCmd struct {
	Date string `arg:"" help:"Date to list (YYYY-MM-DD)."`
}

func (c *WorkoutScheduleListCmd) Run(g *Globals) error {
	if !dateRegexp.MatchString(c.Date) {
		return fmt.Errorf("invalid date format %q: expected YYYY-MM-DD", c.Date)
	}

	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.GetCalendarWeek(g.Context, c.Date)
	if err != nil {
		return fmt.Errorf("list scheduled workouts: %w", err)
	}

	items, err := filterCalendarWorkouts(data, c.Date)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(g.Context) {
		out, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal calendar items: %w", err)
		}
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(out))
	}

	rows := formatCalendarWorkoutRows(items)
	header := []string{"SCHEDULE ID", "WORKOUT ID", "TITLE", "SPORT", "DATE"}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, header, rows)
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

// formatCalendarWorkoutRows extracts table rows from calendar workout items.
func formatCalendarWorkoutRows(items []map[string]any) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			jsonString(item, "scheduleId"),
			jsonString(item, "workoutId"),
			jsonString(item, "title"),
			jsonString(item, "sportTypeKey"),
			jsonString(item, "date"),
		})
	}
	return rows
}

// WorkoutScheduleRemoveCmd removes a scheduled workout from the calendar.
type WorkoutScheduleRemoveCmd struct {
	ID    string `arg:"" help:"Schedule ID (from 'workouts schedule list')."`
	Force bool   `help:"Skip confirmation prompt." short:"f"`
}

func (c *WorkoutScheduleRemoveCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	ok, err := confirm(os.Stderr, fmt.Sprintf("Remove scheduled workout %s?", c.ID), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	if err := client.UnscheduleWorkout(g.Context, c.ID); err != nil {
		return fmt.Errorf("remove scheduled workout: %w", err)
	}

	g.UI.Successf("Removed scheduled workout %s", c.ID)
	return nil
}

// WorkoutDeleteCmd deletes a workout.
type WorkoutDeleteCmd struct {
	ID    string `arg:"" help:"Workout ID."`
	Force bool   `help:"Skip confirmation prompt." short:"f"`
}

func (c *WorkoutDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	ok, err := confirm(os.Stderr, fmt.Sprintf("Delete workout %s?", c.ID), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	if err := client.DeleteWorkout(g.Context, c.ID); err != nil {
		return fmt.Errorf("delete workout: %w", err)
	}

	g.UI.Successf("Deleted workout %s", c.ID)
	return nil
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
