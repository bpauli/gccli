package cmd

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

var workoutsGroup = Group{Name: "workouts", Help: "Manage workouts.", Children: []Node{
	Custom{Name: "list", Help: "List workouts.", Default: true, Cmd: &WorkoutsListCmd{}},
	Leaf{
		Name: "detail", Help: "View workout details.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Workout ID."`
		}{}),
		Endpoint: "workouts.get",
		Params:   Params{"workoutID": In("ID")},
		Verb:     "get workout", Output: JSON{},
	},
	Custom{Name: "download", Help: "Download workout as FIT file.", Cmd: &WorkoutDownloadCmd{}},
	Custom{Name: "upload", Help: "Upload workout from JSON file.", Cmd: &WorkoutUploadCmd{}},
	Custom{Name: "update", Help: "Update an existing workout from a JSON file.", Cmd: &WorkoutUpdateCmd{}},
	Custom{Name: "create", Help: "Create a workout with sport type and optional targets.", Cmd: &WorkoutCreateCmd{}},
	Custom{Name: "copy", Help: "Copy a workout.", Cmd: &WorkoutCopyCmd{}},
	Custom{Name: "send", Help: "Send a workout to a device at its next sync. Undo: devices messages cancel.", Cmd: &WorkoutSendCmd{}},
	Leaf{Name: "types", Help: "List workout types.", Endpoint: "workouts.types", Verb: "get workout types", Output: JSON{}},
	Leaf{
		Name: "sport-types", Help: "List the sport types a workout can have.",
		Endpoint: "workouts.sport-types", Verb: "get workout sport types",
		Output: Rows{Noun: "sport types", Columns: []Column{
			{Label: "ID", Path: "sportTypeId"},
			{Label: "KEY", Path: "sportTypeKey"},
			{Label: "ORDER", Path: "displayOrder"},
		}},
	},
	workoutScheduleGroup,
	workoutBenchmarksGroup,
	Leaf{
		Name: "delete", Help: "Delete a workout.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Workout ID."`
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

	return list{noun: "workouts", keys: []string{"workouts", "results"}, header: []string{"ID", "NAME", "TYPE", "OWNER"}, rows: formatWorkoutRows}.print(g, data)
}

// WorkoutDownloadCmd downloads a workout as FIT.
type WorkoutDownloadCmd struct {
	ID     ID     `arg:"" help:"Workout ID."`
	Output string `help:"Output file path (default: workout_{id}.fit)." short:"o"`
}

func (c *WorkoutDownloadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.DownloadWorkout(g.Context, string(c.ID))
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
	ID   NumericID `arg:"" help:"Workout ID to update."`
	File string    `arg:"" help:"Path to workout JSON file." type:"existingfile"`
}

func (c *WorkoutUpdateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	jsonData, err := os.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	payload, replaced, err := setWorkoutID(jsonData, c.ID.String())
	if err != nil {
		return fmt.Errorf("%s: %w", c.File, err)
	}
	if replaced != "" {
		g.UI.Warnf("Workout JSON has workoutId %s, updating workout %s instead", replaced, c.ID)
	}

	data, err := client.Call(g.Context, "workouts.update", map[string]string{"workoutID": c.ID.String()}, payload)
	if err != nil {
		return fmt.Errorf("update workout: %w", err)
	}

	return reportWrite(g, data, "Updated workout %s from %s", c.ID, c.File)
}

// WorkoutCopyCmd copies a workout.
type WorkoutCopyCmd struct {
	ID   ID     `arg:"" help:"Workout ID."`
	Name string `arg:"" optional:"" help:"Name of the copy. Without it, the copy gets the name of the original."`
}

func (c *WorkoutCopyCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.CopyWorkout(g.Context, string(c.ID), c.Name)
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

// WorkoutSendCmd sends a workout to a device.
type WorkoutSendCmd struct {
	ID     NumericID `arg:"" help:"Workout ID."`
	Device ID        `help:"Device ID (from 'devices list'). Needed when more than one device can receive the workout."`
}

func (c *WorkoutSendCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "workouts.get", map[string]string{"workoutID": c.ID.String()}, nil)
	if err != nil {
		return fmt.Errorf("get workout: %w", err)
	}
	var workout garminapi.Workout
	if err := unmarshalAnswer(data, &workout); err != nil {
		return fmt.Errorf("parse workout: %w", err)
	}
	if workout.Name == "" {
		return errors.New("parse workout: no workoutName")
	}

	data, err = client.Call(g.Context, "devices.list", nil, nil)
	if err != nil {
		return fmt.Errorf("get devices: %w", err)
	}
	var devices []garminapi.Device
	if err := unmarshalAnswer(data, &devices); err != nil {
		return fmt.Errorf("parse devices: %w", err)
	}
	device, err := pickWorkoutDevice(devices, workout, string(c.Device))
	if err != nil {
		return err
	}

	data, err = client.SendWorkoutToDevice(g.Context, int64(c.ID), workout, device)
	if err != nil {
		return fmt.Errorf("send workout: %w", err)
	}
	name := cmp.Or(device.Name, device.ID.String())
	var msgs []map[string]any
	_ = json.Unmarshal(data, &msgs)
	if len(msgs) > 0 {
		if id := jsonString(msgs[0], "messageId"); id != "" {
			return reportWrite(g, data, "Sent workout %q to %s (message %s). It reaches the device at its next sync. To cancel it before then, run: gccli devices messages cancel %s",
				workout.Name, name, id, id)
		}
	}
	return reportWrite(g, data, "Sent workout %q to %s. It reaches the device at its next sync. To cancel it before then, find the message in: gccli devices messages",
		workout.Name, name)
}

// pickWorkoutDevice returns the device that want names, or without want the
// only device that can receive the workout. It fails before any write.
func pickWorkoutDevice(devices []garminapi.Device, w garminapi.Workout, want string) (garminapi.Device, error) {
	var able []garminapi.Device
	for _, d := range devices {
		if d.CanReceive(w) {
			able = append(able, d)
		}
	}
	choose := "no registered device can receive this workout"
	if len(able) > 0 {
		labels := make([]string, len(able))
		for i, d := range able {
			labels[i] = deviceLabel(d)
		}
		choose = "pass --device with one of: " + strings.Join(labels, ", ")
	}

	if want == "" {
		switch len(able) {
		case 0:
			return garminapi.Device{}, errors.New(choose)
		case 1:
			return able[0], nil
		}
		return garminapi.Device{}, fmt.Errorf("%d devices can receive this workout, %s", len(able), choose)
	}
	for _, d := range devices {
		if d.ID.String() != want {
			continue
		}
		if !d.CanReceive(w) {
			return garminapi.Device{}, fmt.Errorf("device %s cannot receive this workout, %s", deviceLabel(d), choose)
		}
		return d, nil
	}
	return garminapi.Device{}, fmt.Errorf("device %s is not registered, %s", want, choose)
}

// deviceLabel names a device as "<id> (<name>)".
func deviceLabel(d garminapi.Device) string {
	if d.Name == "" {
		return d.ID.String()
	}
	return fmt.Sprintf("%s (%s)", d.ID, d.Name)
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
			ID ID `arg:"" help:"Schedule ID (from 'workouts schedule list')."`
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

var workoutBenchmarksGroup = Group{Name: "benchmarks", Help: "View, set and clear strength benchmarks.", Children: []Node{
	Leaf{Name: "list", Help: "List strength benchmarks.", Default: true, Endpoint: "workouts.benchmarks", Verb: "get benchmarks", Output: JSON{List: true}},
	Custom{Name: "set", Help: "Set a strength benchmark from a weight and reps.", Cmd: &WorkoutBenchmarkSetCmd{}},
	Custom{Name: "clear", Help: "Clear a strength benchmark.", Cmd: &WorkoutBenchmarkClearCmd{}},
}}

// benchmarkUnits maps each --unit value to its Garmin unit key and the
// heaviest weight the Garmin Connect web app accepts in that unit.
var benchmarkUnits = map[string]struct {
	key string
	max float64
}{
	"kg":  {key: "kilogram", max: 453.5},
	"lbs": {key: "pound", max: 999.9},
}

const (
	minBenchmarkWeight = 0.1
	maxBenchmarkReps   = 20
)

// WorkoutBenchmarkSetCmd sets a strength benchmark.
type WorkoutBenchmarkSetCmd struct {
	Key    FreeText `arg:"" help:"Benchmark key, such as BARBELL_ROW (from 'workouts benchmarks list')."`
	Weight float64  `required:"" placeholder:"WEIGHT" help:"Heaviest weight lifted for the reps, 0.1 to 453.5 kg or 0.1 to 999.9 lbs."`
	Reps   int      `required:"" placeholder:"N" help:"Number of reps lifted with the weight, 1 to 20."`
	Unit   string   `help:"Unit of the weight: kg or lbs." default:"kg" enum:"kg,lbs"`
}

func (c *WorkoutBenchmarkSetCmd) Validate() error {
	limit := benchmarkUnits[c.Unit].max
	// The negated range check also refuses NaN.
	switch {
	case c.Reps < 1 || c.Reps > maxBenchmarkReps:
		return fmt.Errorf("--reps must be between 1 and %d", maxBenchmarkReps)
	case !(c.Weight >= minBenchmarkWeight && c.Weight <= limit):
		return fmt.Errorf("--weight must be between %g and %g %s", minBenchmarkWeight, limit, c.Unit)
	}
	return nil
}

func (c *WorkoutBenchmarkSetCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	key := strings.ToUpper(string(c.Key))
	v := garminapi.BenchmarkValue{Weight: c.Weight, Reps: c.Reps, UnitKey: benchmarkUnits[c.Unit].key}
	data, err := client.SetBenchmark(g.Context, key, v)
	if err != nil {
		return fmt.Errorf("set benchmark: %w", err)
	}
	return reportWrite(g, data, "Set benchmark %s to %g %s x %d (1RM %g %s)", key, c.Weight, c.Unit, c.Reps, v.OneRepMax(), c.Unit)
}

// WorkoutBenchmarkClearCmd clears a strength benchmark.
type WorkoutBenchmarkClearCmd struct {
	Key   FreeText `arg:"" help:"Benchmark key, such as BARBELL_ROW (from 'workouts benchmarks list')."`
	Force bool     `help:"Skip confirmation prompt." short:"f"`
}

func (c *WorkoutBenchmarkClearCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	key := strings.ToUpper(string(c.Key))
	b, err := client.Benchmark(g.Context, key)
	if err != nil {
		return fmt.Errorf("clear benchmark: %w", err)
	}
	if b.ID == 0 {
		return fmt.Errorf("benchmark %s is not set", key)
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf("Clear benchmark %s?", key), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	data, err := client.Call(g.Context, "workouts.delete-benchmark", map[string]string{"benchmarkID": strconv.FormatInt(b.ID, 10)}, nil)
	if err != nil {
		return fmt.Errorf("clear benchmark: %w", err)
	}
	return reportWrite(g, data, "Cleared benchmark %s", key)
}

// WorkoutScheduleAddCmd schedules a workout on a calendar date.
type WorkoutScheduleAddCmd struct {
	ID   ID   `arg:"" help:"Workout ID."`
	Date Date `arg:"" help:"Date to schedule (YYYY-MM-DD, today, yesterday, 3d)."`
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
	data, err := client.Call(g.Context, "workouts.schedule", map[string]string{"workoutID": string(c.ID)}, body)
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

// writeScheduledWorkouts prints the items as one list answer, so a day
// without a workout prints like every empty list.
func writeScheduledWorkouts(g *Globals, items []map[string]any) error {
	var data json.RawMessage
	if len(items) > 0 {
		var err error
		if data, err = json.Marshal(items); err != nil {
			return fmt.Errorf("marshal calendar items: %w", err)
		}
	}
	return list{noun: "calendar items", header: []string{"SCHEDULE ID", "WORKOUT ID", "TITLE", "SPORT", "DATE"}, rows: formatCalendarWorkoutRows}.print(g, data)
}

// filterCalendarWorkouts extracts workout items matching the given date from calendar data.
func filterCalendarWorkouts(data json.RawMessage, date string) ([]map[string]any, error) {
	var calendar map[string]any
	if err := unmarshalAnswer(data, &calendar); err != nil {
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
	if err := unmarshalAnswer(data, &calendar); err != nil {
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
