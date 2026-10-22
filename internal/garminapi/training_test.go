package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestTrainingEndpoints(t *testing.T) {
	testEndpoints(t, trainingEndpoints, []endpointCase{
		{"training.plans", map[string]string{"locale": "de"}, "GET /trainingplan-service/trainingplan/plans?locale=de"},
		{"training.plans", map[string]string{"locale": ""}, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plans", nil, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plan", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/42"},
		{"training.active-plans", map[string]string{"startDate": "2024-01-15"}, "GET /trainingplan-service/trainingplan/active-plans?startDate=2024-01-15"},
		{"training.schedule", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/tasks/42?localeKey=en"},
		{"training.schedule", map[string]string{"planID": "42", "locale": "de"}, "GET /trainingplan-service/trainingplan/tasks/42?localeKey=de"},
		{"training.join", map[string]string{"planID": "42", "startDate": "2026-10-26"}, "GET /trainingplan-service/trainingplan/schedule/42?startDate=2026-10-26"},
		{"training.leave", map[string]string{"planID": "42"}, "DELETE /trainingplan-service/trainingplan/trainingplan/42"},
		{"training.delete-task", map[string]string{"planID": "42", "taskID": "7"}, "DELETE /trainingplan-service/trainingplan/workoutTask/42/7"},
		{"training.adaptive-coaching-workouts", nil, "GET /userprofile-service/userprofile/adaptive-coaching-workouts"},
	})
}

func TestTrainingFormRequests(t *testing.T) {
	tests := []struct {
		name string
		send func(*Client) (json.RawMessage, error)
		want sent
	}{
		{
			name: "search without a filter",
			send: func(c *Client) (json.RawMessage, error) {
				return c.SearchTrainingPlans(context.Background(), TrainingPlanFilter{})
			},
			want: sent{line: "POST /trainingplan-service/trainingplan/search?limit=1000&locale=en&start=1"},
		},
		{
			name: "search with every filter",
			send: func(c *Client) (json.RawMessage, error) {
				return c.SearchTrainingPlans(context.Background(), TrainingPlanFilter{Sport: "Running", Goal: "5k", Level: "Beginner", Target: "HeartRate", Locale: "de"})
			},
			want: sent{line: "POST /trainingplan-service/trainingplan/search?levels=Beginner&limit=1000&locale=de&start=1&subTypes=5k&types=Running&version=HeartRate"},
		},
		{
			name: "move task",
			send: func(c *Client) (json.RawMessage, error) {
				return c.MoveTrainingPlanTask(context.Background(), "4/2", "7", "2026-10-28")
			},
			want: sent{line: "POST /trainingplan-service/trainingplan/moveWorkoutTask/4%2F2", body: "scheduleDate=2026-10-28&workoutSchedulePk=7"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got sent
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				got = sent{line: r.Method + " " + r.URL.RequestURI(), contentType: r.Header.Get("Content-Type"), body: string(b)}
				_, _ = w.Write([]byte(`{"ok":1}`))
			}))
			data, err := tc.send(client)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != `{"ok":1}` {
				t.Errorf("data = %s, want the answer", data)
			}
			tc.want.contentType = formContentType
			if got != tc.want {
				t.Errorf("sent %+v, want %+v: Garmin answers 415 without the form type", got, tc.want)
			}
		})
	}
}

func TestTrainingFormRequestError(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	_, err := client.MoveTrainingPlanTask(context.Background(), "42", "7", "2026-10-28")
	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %v, want a GarminAPIError with status 404", err)
	}
}
