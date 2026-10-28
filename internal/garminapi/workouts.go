package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var workoutsEndpoints = []Endpoint{
	{Name: "workouts.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/workouts?start={start}&limit={limit}"},
	{Name: "workouts.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/workout/{workoutID}"},
	{Name: "workouts.upload", Method: http.MethodPost, Host: ConnectAPI, Path: "/workout-service/workout", Body: true},
	{Name: "workouts.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/workout-service/workout/{workoutID}", Body: true},
	{Name: "workouts.schedule", Method: http.MethodPost, Host: ConnectAPI, Path: "/workout-service/schedule/{workoutID}", Body: true},
	{Name: "workouts.unschedule", Method: http.MethodDelete, Host: ConnectAPI, Path: "/workout-service/schedule/{scheduleID}"},
	{Name: "workouts.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/workout-service/workout/{workoutID}"},
	{Name: "workouts.types", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/workout/types"},
	{Name: "workouts.sport-types", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/workout/types/sport"},
	{Name: "workouts.export-schedule", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/schedule/export/JSON?startDate={startDate}"},
	{Name: "workouts.benchmarks", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/benchmarks"},
	{Name: "workouts.create-benchmark", Method: http.MethodPost, Host: ConnectAPI, Path: "/workout-service/benchmarks", Body: true},
	{Name: "workouts.update-benchmark", Method: http.MethodPut, Host: ConnectAPI, Path: "/workout-service/benchmarks/{benchmarkID}", Body: true},
	{Name: "workouts.delete-benchmark", Method: http.MethodDelete, Host: ConnectAPI, Path: "/workout-service/benchmarks/{benchmarkID}"},
}

// Benchmark is one entry of the benchmark list. The list holds every
// benchmark Garmin knows, set or not. ID is 0 when the benchmark is not set.
type Benchmark struct {
	ID          int64  `json:"id"`
	Key         string `json:"benchmarkKey"`
	CategoryKey string `json:"benchmarkCategoryKey"`
}

// Benchmark returns the entry of the benchmark list with the given key. An
// unknown key is an error that lists the known keys.
func (c *Client) Benchmark(ctx context.Context, key string) (Benchmark, error) {
	data, err := c.Call(ctx, "workouts.benchmarks", nil, nil)
	if err != nil {
		return Benchmark{}, err
	}
	var list []Benchmark
	if err := json.Unmarshal(data, &list); err != nil {
		return Benchmark{}, fmt.Errorf("parse benchmarks: %w", err)
	}
	keys := make([]string, 0, len(list))
	for _, b := range list {
		if b.Key == key {
			return b, nil
		}
		keys = append(keys, b.Key)
	}
	return Benchmark{}, fmt.Errorf("unknown benchmark %q, known: %s", key, strings.Join(keys, ", "))
}

// BenchmarkValue is the heaviest weight lifted for a number of reps.
// UnitKey is the Garmin key of the weight unit: "kilogram" or "pound".
type BenchmarkValue struct {
	Weight  float64
	Reps    int
	UnitKey string
}

// OneRepMax estimates the one-rep max with the Brzycki formula, rounded to
// 0.1, as the Garmin Connect web app does. Garmin stores the value as sent.
func (v BenchmarkValue) OneRepMax() float64 {
	if v.Reps < 1 || v.Reps > 36 {
		return 0
	}
	return math.Floor(v.Weight*(36/float64(37-v.Reps))*10+0.5) / 10
}

type benchmarkBody struct {
	Key         string            `json:"benchmarkKey"`
	CategoryKey string            `json:"benchmarkCategoryKey"`
	MaxWeight   float64           `json:"maxWeight"`
	Reps        int               `json:"reps"`
	OneRepMax   float64           `json:"oneRepMax"`
	WeightUnit  benchmarkUnit     `json:"weightUnit"`
	Exercises   []benchmarkLinked `json:"linkedBenchmarkExercises"`
}

type benchmarkUnit struct {
	Key string `json:"unitKey"`
}

type benchmarkLinked struct {
	ExerciseKey string `json:"exerciseKey"`
	CategoryKey string `json:"categoryKey"`
}

// SetBenchmark saves v as the benchmark with the given key. A benchmark that
// is not set gets a POST, which answers the new ID. A set one gets a PUT of
// its ID, because a second POST of the same key fails.
func (c *Client) SetBenchmark(ctx context.Context, key string, v BenchmarkValue) (json.RawMessage, error) {
	b, err := c.Benchmark(ctx, key)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(benchmarkBody{
		Key: b.Key, CategoryKey: b.CategoryKey,
		MaxWeight: v.Weight, Reps: v.Reps, OneRepMax: v.OneRepMax(),
		WeightUnit: benchmarkUnit{Key: v.UnitKey},
		Exercises:  []benchmarkLinked{{ExerciseKey: b.Key, CategoryKey: b.CategoryKey}},
	})
	if err != nil {
		return nil, err
	}
	if b.ID == 0 {
		return c.Call(ctx, "workouts.create-benchmark", nil, body)
	}
	return c.Call(ctx, "workouts.update-benchmark", map[string]string{"benchmarkID": strconv.FormatInt(b.ID, 10)}, body)
}

// DownloadWorkout downloads a workout as a FIT file.
func (c *Client) DownloadWorkout(ctx context.Context, workoutID string) ([]byte, error) {
	return c.Download(ctx, "/workout-service/workout/FIT/"+workoutID)
}

// CopyWorkout copies a workout and returns the new one. Garmin takes the raw
// request body as the name of the copy, so the name goes as text/plain: a JSON
// string would keep its quotes.
func (c *Client) CopyWorkout(ctx context.Context, workoutID, name string) (json.RawMessage, error) {
	resp, err := c.do(ctx, request{
		kind: kindJSON, method: http.MethodPost, path: "/workout-service/workout/copy/" + url.PathEscape(workoutID),
		body: []byte(name), contentType: "text/plain; charset=utf-8",
	})
	if err != nil {
		return nil, err
	}
	return jsonResult(resp), nil
}

// Workout holds the fields of a workouts.get answer that a send to a device
// reads.
type Workout struct {
	Name      string    `json:"workoutName"`
	SportType SportType `json:"sportType"`
	Segments  []struct {
		SportType SportType     `json:"sportType"`
		Steps     []WorkoutStep `json:"workoutSteps"`
	} `json:"workoutSegments"`
}

// WorkoutStep is a step of a workout segment, or a repeat group of steps.
type WorkoutStep struct {
	Type       string `json:"type"`
	Iterations *int   `json:"numberOfIterations"`
	StepType   struct {
		Key string `json:"stepTypeKey"`
	} `json:"stepType"`
	Steps []WorkoutStep `json:"workoutSteps"`
}

// stepCount counts the steps of the workout as the Garmin Connect web app
// does for its step limit.
func (w Workout) stepCount() int {
	n := 0
	for _, s := range w.Segments {
		n += countSteps(s.Steps, 1)
	}
	return n
}

func countSteps(steps []WorkoutStep, times int) int {
	n := 0
	for _, s := range steps {
		switch {
		case s.Type == "RepeatGroupDTO":
			iterations := 1
			if s.Iterations != nil {
				iterations = *s.Iterations
			}
			n += countSteps(s.Steps, iterations*times)
		case s.StepType.Key != "rest":
			n += times
		}
	}
	return n
}

// SportType is the sport of a workout or of a workout segment.
type SportType struct {
	Key string `json:"sportTypeKey"`
}

// SendWorkoutToDevice queues the workout for the device, which downloads it
// at its next sync. The message matches the one of the Garmin Connect web
// app: a swim workout names the device in its URL when the device lacks
// swimWorkout2Capable.
func (c *Client) SendWorkoutToDevice(ctx context.Context, workoutID int64, w Workout, d Device) (json.RawMessage, error) {
	fileType := d.WorkoutFileType
	if fileType == "" {
		fileType = "FIT"
	}
	messageURL := fmt.Sprintf("workout-service/workout/%s/%d", fileType, workoutID)
	if w.SportType.Key == "swimming" && !d.flags["swimWorkout2Capable"] {
		messageURL += "?deviceId=" + d.ID.String()
	}
	body, err := json.Marshal([]map[string]any{{
		"deviceId":    d.ID,
		"messageUrl":  messageURL,
		"messageType": "workouts",
		"messageName": w.Name,
		"groupName":   nil,
		"priority":    1,
		"fileType":    fileType,
		"metaDataId":  workoutID,
	}})
	if err != nil {
		return nil, fmt.Errorf("marshal send payload: %w", err)
	}
	return c.Call(ctx, "devices.send-message", nil, body)
}

// GetCalendarWeek returns the calendar data for the week containing the given date.
// The date must be in YYYY-MM-DD format. The Garmin calendar API uses 0-indexed months.
func (c *Client) GetCalendarWeek(ctx context.Context, date string) (json.RawMessage, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("parse date %q: %w", date, err)
	}
	path := fmt.Sprintf("/calendar-service/year/%d/month/%d/day/%d/start/0", t.Year(), t.Month()-1, t.Day())
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}
