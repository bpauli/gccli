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
	{Name: "workouts.sport-types", Method: http.MethodGet, Host: ConnectAPI, Path: "/workout-service/workout/types/sport"},
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
