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

// TestUploadWorkout uploads a minimal workout, verifies it appears in the
// listing, downloads it as FIT, and deletes it via t.Cleanup.
func TestUploadWorkout(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Safety-net: clean up orphaned E2E workouts from prior runs.
	cleanupOrphanWorkouts(t, client)

	// Create a minimal workout JSON with E2E_TEST_ prefix.
	workoutName := fmt.Sprintf("%sWORKOUT_%d", e2eWorkoutPrefix, time.Now().UnixNano())
	workoutJSON := json.RawMessage(fmt.Sprintf(`{
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
				"endConditionValue": 1000
			}]
		}]
	}`, workoutName))

	data, err := client.UploadWorkout(ctx, workoutJSON)
	if err != nil {
		t.Fatalf("UploadWorkout failed: %v", err)
	}

	// Extract the created workout ID.
	var created map[string]any
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatalf("unmarshal created workout: %v", err)
	}

	workoutID := formatID(created["workoutId"])
	if workoutID == "" || workoutID == "0" {
		t.Fatalf("expected valid workoutId, got %q", workoutID)
	}
	t.Logf("Created workout %s with name %q", workoutID, workoutName)

	// Register cleanup to delete the workout — runs even on test failure.
	RegisterCleanup(t, func() {
		if delErr := client.DeleteWorkout(context.Background(), workoutID); delErr != nil {
			t.Logf("WARNING: failed to clean up workout %s: %v", workoutID, delErr)
		} else {
			t.Logf("Cleaned up workout %s", workoutID)
		}
	})

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
