//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

const e2eWorkoutPrefix = "E2E_TEST_"

// workoutPayload builds a minimal running workout. workoutID is empty for an
// upload and set to the target ID for an update: Garmin resolves the update by
// the ID in the request path, and "gccli workouts update" writes the same ID
// into the body before sending it.
func workoutPayload(name string, distanceMeters int, workoutID string) json.RawMessage {
	idField := ""
	if workoutID != "" {
		idField = fmt.Sprintf("\n\t\t\"workoutId\": %s,", workoutID)
	}

	return json.RawMessage(fmt.Sprintf(`{%s
		"workoutName": %q,
		"sportType": {
			"sportTypeId": 1,
			"sportTypeKey": "running"
		},
		"workoutSegments": [{
			"segmentOrder": 1,
			"sportType": {
				"sportTypeId": 1,
				"sportTypeKey": "running"
			},
			"workoutSteps": [{
				"type": "ExecutableStepDTO",
				"stepOrder": 1,
				"stepType": {
					"stepTypeId": 3,
					"stepTypeKey": "interval"
				},
				"endCondition": {
					"conditionTypeId": 2,
					"conditionTypeKey": "distance"
				},
				"endConditionValue": %d
			}]
		}]
	}`, idField, name, distanceMeters))
}

// createE2EWorkout uploads a 1000 m workout and returns its ID. The workout is
// deleted when the test finishes, even on failure.
func createE2EWorkout(t *testing.T, client *garminapi.Client, name string) string {
	t.Helper()
	ctx := context.Background()

	data, err := client.UploadWorkout(ctx, workoutPayload(name, 1000, ""))
	if err != nil {
		t.Fatalf("UploadWorkout failed: %v", err)
	}

	var created map[string]any
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatalf("unmarshal created workout: %v", err)
	}

	workoutID := formatID(created["workoutId"])
	if workoutID == "" || workoutID == "0" {
		t.Fatalf("expected valid workoutId, got %q", workoutID)
	}
	t.Logf("Created workout %s with name %q", workoutID, name)

	RegisterCleanup(t, func() {
		if delErr := client.DeleteWorkout(context.Background(), workoutID); delErr != nil {
			t.Logf("WARNING: failed to clean up workout %s: %v", workoutID, delErr)
		} else {
			t.Logf("Cleaned up workout %s", workoutID)
		}
	})

	return workoutID
}

// TestUploadWorkout uploads a minimal workout, verifies it appears in the
// listing, downloads it as FIT, and deletes it via t.Cleanup.
func TestUploadWorkout(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Safety-net: clean up orphaned E2E workouts from prior runs.
	cleanupOrphanWorkouts(t, client)

	workoutName := fmt.Sprintf("%sWORKOUT_%d", e2eWorkoutPrefix, time.Now().UnixNano())
	workoutID := createE2EWorkout(t, client, workoutName)

	// Verify the workout appears in the listing.
	found := findWorkoutByID(t, client, workoutID)
	if !found {
		t.Errorf("created workout %s not found in GetWorkouts listing", workoutID)
	}

	// Test download as FIT.
	fitData, err := client.DownloadWorkout(ctx, workoutID)
	if err != nil {
		t.Fatalf("DownloadWorkout(%s) failed: %v", workoutID, err)
	}
	if len(fitData) == 0 {
		t.Error("expected non-empty FIT data from DownloadWorkout")
	}
	t.Logf("Downloaded workout %s as FIT: %d bytes", workoutID, len(fitData))

	// The t.Cleanup handler will delete the workout.
}

// TestUpdateWorkout replaces an uploaded workout and verifies that the changed
// payload lands on the server while the workout keeps its ID.
func TestUpdateWorkout(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	workoutName := fmt.Sprintf("%sUPDATE_%d", e2eWorkoutPrefix, time.Now().UnixNano())
	workoutID := createE2EWorkout(t, client, workoutName)

	updatedName := workoutName + "_V2"
	data, err := client.UpdateWorkout(ctx, workoutID, workoutPayload(updatedName, 2000, workoutID))
	if err != nil {
		t.Fatalf("UpdateWorkout(%s) failed: %v", workoutID, err)
	}

	// Garmin answers the PUT with an empty body. Older behaviour echoed the
	// workout back, so check the ID whenever there is something to check.
	if len(data) == 0 {
		t.Logf("UpdateWorkout(%s) returned an empty body", workoutID)
	} else {
		var updated map[string]any
		if err := json.Unmarshal(data, &updated); err != nil {
			t.Fatalf("unmarshal update response: %v", err)
		}
		if got := formatID(updated["workoutId"]); got != workoutID {
			t.Errorf("update response workoutId = %q, want %q", got, workoutID)
		}
	}

	// Read the workout back: an update is a full replace, so both the name and
	// the step must reflect the new payload under the original ID.
	stored := fetchWorkout(t, client, workoutID)
	if got := formatID(stored["workoutId"]); got != workoutID {
		t.Errorf("stored workoutId = %q, want %q", got, workoutID)
	}
	if got, _ := stored["workoutName"].(string); got != updatedName {
		t.Errorf("stored workoutName = %q, want %q", got, updatedName)
	}
	if got, ok := firstStepDistance(stored); !ok {
		t.Error("stored workout has no first step endConditionValue")
	} else if got != 2000 {
		t.Errorf("stored endConditionValue = %v, want 2000", got)
	}

	// The update must not have created a second workout.
	if !findWorkoutByID(t, client, workoutID) {
		t.Errorf("updated workout %s not found in GetWorkouts listing", workoutID)
	}
	t.Logf("Updated workout %s in place", workoutID)
}

// TestUpdateWorkoutKeepsSchedule verifies the reason update exists: the
// calendar entry survives. Delete plus upload would drop the schedule and hand
// out a new workout ID.
func TestUpdateWorkoutKeepsSchedule(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	workoutName := fmt.Sprintf("%sSCHEDULED_%d", e2eWorkoutPrefix, time.Now().UnixNano())
	workoutID := createE2EWorkout(t, client, workoutName)

	date := time.Now().AddDate(0, 0, 14).Format("2006-01-02")
	if _, err := client.ScheduleWorkout(ctx, workoutID, date); err != nil {
		t.Fatalf("ScheduleWorkout(%s, %s) failed: %v", workoutID, date, err)
	}

	scheduleID := findScheduleID(t, client, workoutID, date)
	if scheduleID == "" {
		t.Fatalf("workout %s not on the calendar for %s after scheduling", workoutID, date)
	}
	t.Logf("Scheduled workout %s on %s (schedule %s)", workoutID, date, scheduleID)

	// Runs before the workout delete registered above (t.Cleanup is LIFO).
	RegisterCleanup(t, func() {
		if err := client.UnscheduleWorkout(context.Background(), scheduleID); err != nil {
			t.Logf("WARNING: failed to remove schedule %s: %v", scheduleID, err)
		} else {
			t.Logf("Removed schedule %s", scheduleID)
		}
	})

	updatedName := workoutName + "_V2"
	if _, err := client.UpdateWorkout(ctx, workoutID, workoutPayload(updatedName, 2000, workoutID)); err != nil {
		t.Fatalf("UpdateWorkout(%s) failed: %v", workoutID, err)
	}

	if got := findScheduleID(t, client, workoutID, date); got != scheduleID {
		t.Errorf("schedule ID for workout %s on %s = %q after update, want %q", workoutID, date, got, scheduleID)
	}
}

// fetchWorkout reads a single workout from the server.
func fetchWorkout(t *testing.T, client *garminapi.Client, workoutID string) map[string]any {
	t.Helper()

	data, err := client.GetWorkout(context.Background(), workoutID)
	if err != nil {
		t.Fatalf("GetWorkout(%s) failed: %v", workoutID, err)
	}

	var workout map[string]any
	if err := json.Unmarshal(data, &workout); err != nil {
		t.Fatalf("unmarshal workout %s: %v", workoutID, err)
	}
	return workout
}

// firstStepDistance returns the endConditionValue of the first step of the
// first segment.
func firstStepDistance(workout map[string]any) (float64, bool) {
	segments, ok := workout["workoutSegments"].([]any)
	if !ok || len(segments) == 0 {
		return 0, false
	}
	segment, ok := segments[0].(map[string]any)
	if !ok {
		return 0, false
	}
	steps, ok := segment["workoutSteps"].([]any)
	if !ok || len(steps) == 0 {
		return 0, false
	}
	step, ok := steps[0].(map[string]any)
	if !ok {
		return 0, false
	}
	value, ok := step["endConditionValue"].(float64)
	return value, ok
}

// findScheduleID returns the calendar schedule ID for a workout on a date, or
// an empty string when the workout is not scheduled that day.
func findScheduleID(t *testing.T, client *garminapi.Client, workoutID, date string) string {
	t.Helper()

	data, err := client.GetCalendarWeek(context.Background(), date)
	if err != nil {
		t.Fatalf("GetCalendarWeek(%s) failed: %v", date, err)
	}

	var calendar struct {
		CalendarItems []map[string]any `json:"calendarItems"`
	}
	if err := json.Unmarshal(data, &calendar); err != nil {
		t.Fatalf("parse calendar for %s: %v", date, err)
	}

	for _, item := range calendar.CalendarItems {
		if itemType, _ := item["itemType"].(string); itemType != "workout" {
			continue
		}
		if itemDate, _ := item["date"].(string); itemDate != date {
			continue
		}
		if formatID(item["workoutId"]) != workoutID {
			continue
		}
		return formatID(item["id"])
	}
	return ""
}

// findWorkoutByID searches recent workouts for one with the given ID.
func findWorkoutByID(t *testing.T, client *garminapi.Client, workoutID string) bool {
	t.Helper()
	ctx := context.Background()

	data, err := client.GetWorkouts(ctx, 0, 50)
	if err != nil {
		t.Fatalf("GetWorkouts failed while searching for workout %s: %v", workoutID, err)
	}

	// GetWorkouts may return an array or a wrapper object.
	var workouts []map[string]any
	if err := json.Unmarshal(data, &workouts); err != nil {
		// Try wrapper format.
		var wrapper map[string]json.RawMessage
		if wErr := json.Unmarshal(data, &wrapper); wErr != nil {
			t.Fatalf("unmarshal workouts: %v (and wrapper: %v)", err, wErr)
		}
		// Look for common wrapper keys.
		for _, key := range []string{"workouts", "items"} {
			if raw, ok := wrapper[key]; ok {
				if jErr := json.Unmarshal(raw, &workouts); jErr == nil {
					break
				}
			}
		}
	}

	for _, w := range workouts {
		if formatID(w["workoutId"]) == workoutID {
			return true
		}
	}
	return false
}

// cleanupOrphanWorkouts deletes any workouts with the E2E_TEST_ prefix
// left over from prior failed runs.
func cleanupOrphanWorkouts(t *testing.T, client *garminapi.Client) {
	t.Helper()
	ctx := context.Background()

	data, err := client.GetWorkouts(ctx, 0, 50)
	if err != nil {
		t.Logf("WARNING: could not list workouts for orphan cleanup: %v", err)
		return
	}

	var workouts []map[string]any
	if err := json.Unmarshal(data, &workouts); err != nil {
		// Try wrapper format.
		var wrapper map[string]json.RawMessage
		if wErr := json.Unmarshal(data, &wrapper); wErr != nil {
			t.Logf("WARNING: could not parse workouts for orphan cleanup: %v", err)
			return
		}
		for _, key := range []string{"workouts", "items"} {
			if raw, ok := wrapper[key]; ok {
				if jErr := json.Unmarshal(raw, &workouts); jErr == nil {
					break
				}
			}
		}
	}

	for _, w := range workouts {
		name, _ := w["workoutName"].(string)
		if strings.HasPrefix(name, e2eWorkoutPrefix) {
			id := formatID(w["workoutId"])
			if delErr := client.DeleteWorkout(ctx, id); delErr != nil {
				t.Logf("WARNING: failed to delete orphaned workout %s (%s): %v", id, name, delErr)
			} else {
				t.Logf("Cleaned up orphaned workout %s (%s)", id, name)
			}
		}
	}
}
