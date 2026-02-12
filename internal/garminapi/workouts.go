package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GetWorkouts returns a page of workouts.
func (c *Client) GetWorkouts(ctx context.Context, start, limit int) (json.RawMessage, error) {
	path := fmt.Sprintf("/workout-service/workouts?start=%d&limit=%d", start, limit)
	return c.ConnectAPI(ctx, http.MethodGet, path, nil)
}

// GetWorkout returns a single workout by ID.
func (c *Client) GetWorkout(ctx context.Context, workoutID string) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodGet, "/workout-service/workout/"+workoutID, nil)
}

// DownloadWorkout downloads a workout as a FIT file.
func (c *Client) DownloadWorkout(ctx context.Context, workoutID string) ([]byte, error) {
	return c.Download(ctx, "/workout-service/workout/FIT/"+workoutID)
}

// UploadWorkout uploads a workout from JSON data.
func (c *Client) UploadWorkout(ctx context.Context, workoutJSON json.RawMessage) (json.RawMessage, error) {
	return c.ConnectAPI(ctx, http.MethodPost, "/workout-service/workout", bytes.NewReader(workoutJSON))
}

// ScheduleWorkout schedules a workout on a calendar date.
func (c *Client) ScheduleWorkout(ctx context.Context, workoutID, date string) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]string{"date": date})
	if err != nil {
		return nil, fmt.Errorf("marshal schedule: %w", err)
	}
	return c.ConnectAPI(ctx, http.MethodPost, "/workout-service/schedule/"+workoutID, bytes.NewReader(body))
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

// UnscheduleWorkout removes a scheduled workout from the calendar.
func (c *Client) UnscheduleWorkout(ctx context.Context, scheduleID string) error {
	_, err := c.ConnectAPI(ctx, http.MethodDelete, "/workout-service/schedule/"+scheduleID, nil)
	return err
}

// DeleteWorkout deletes a workout by ID.
func (c *Client) DeleteWorkout(ctx context.Context, workoutID string) error {
	_, err := c.ConnectAPI(ctx, http.MethodDelete, "/workout-service/workout/"+workoutID, nil)
	return err
}
