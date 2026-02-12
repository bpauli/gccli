package garminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// DeleteWorkout deletes a workout by ID.
func (c *Client) DeleteWorkout(ctx context.Context, workoutID string) error {
	_, err := c.ConnectAPI(ctx, http.MethodDelete, "/workout-service/workout/"+workoutID, nil)
	return err
}
