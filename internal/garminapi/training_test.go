package garminapi

import (
	"context"
	"net/http"
	"testing"
)

func TestTrainingEndpoints(t *testing.T) {
	testEndpoints(t, trainingEndpoints, []endpointCase{
		{"training.plans", map[string]string{"locale": "de"}, "GET /trainingplan-service/trainingplan/plans?locale=de"},
		{"training.plans", map[string]string{"locale": ""}, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plans", nil, "GET /trainingplan-service/trainingplan/plans?locale=en"},
		{"training.plan", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/42"},
		{"training.coach-plan", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/fbt-adaptive/42"},
		{"training.phased-plan", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/phased/42"},
		{"training.strength-plan", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/strength/42"},
		{"training.active-plans", map[string]string{"startDate": "2024-01-15"}, "GET /trainingplan-service/trainingplan/active-plans?startDate=2024-01-15"},
		{"training.search", nil, "POST /trainingplan-service/trainingplan/search?start=1&limit=1000&locale=en"},
		{
			"training.search",
			map[string]string{"locale": "de", "sport": "Running", "goal": "5k", "level": "Beginner", "target": "HeartRate"},
			"POST /trainingplan-service/trainingplan/search?start=1&limit=1000&locale=de&types=Running&subTypes=5k&levels=Beginner&version=HeartRate",
		},
		{
			"training.search",
			map[string]string{"category": "FBT_ADAPTIVE,STRENGTH"},
			"POST /trainingplan-service/trainingplan/search?start=1&limit=1000&locale=en&categories=FBT_ADAPTIVE&categories=STRENGTH",
		},
		{"training.schedule", map[string]string{"planID": "42"}, "GET /trainingplan-service/trainingplan/tasks/42?localeKey=en"},
		{"training.schedule", map[string]string{"planID": "42", "locale": "de"}, "GET /trainingplan-service/trainingplan/tasks/42?localeKey=de"},
		{"training.join", map[string]string{"planID": "42", "startDate": "2026-10-26"}, "GET /trainingplan-service/trainingplan/schedule/42?startDate=2026-10-26"},
		{"training.coach-join", nil, "POST /trainingplan-service/trainingplan/fbt-adaptive"},
		{"training.leave", map[string]string{"planID": "42"}, "DELETE /trainingplan-service/trainingplan/trainingplan/42"},
		{"training.move-task", map[string]string{"planID": "42"}, "POST /trainingplan-service/trainingplan/moveWorkoutTask/42"},
		{"training.delete-task", map[string]string{"planID": "42", "taskID": "7"}, "DELETE /trainingplan-service/trainingplan/workoutTask/42/7"},
		{"training.adaptive-coaching-workouts", nil, "GET /userprofile-service/userprofile/adaptive-coaching-workouts"},
	})
}

// Garmin answers 415 to a search or a task move with a JSON body.
func TestTrainingFormRows(t *testing.T) {
	for name, args := range map[string]map[string]string{"training.search": nil, "training.move-task": {"planID": "42"}} {
		t.Run(name, func(t *testing.T) {
			var contentType string
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contentType = r.Header.Get("Content-Type")
				_, _ = w.Write([]byte(`{}`))
			}))
			if _, err := client.Call(context.Background(), name, args, []byte{}); err != nil {
				t.Fatal(err)
			}
			if contentType != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", contentType)
			}
		})
	}
}
