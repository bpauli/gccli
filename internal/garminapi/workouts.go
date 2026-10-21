package garminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	{Name: "workouts.export-schedule", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/schedule/export/JSON?startDate={startDate}"},
	{Name: "workouts.benchmarks", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/benchmarks"},
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
		SportType SportType `json:"sportType"`
	} `json:"workoutSegments"`
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
