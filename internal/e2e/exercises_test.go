//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// TestStrengthActivityWithExercises creates a strength training activity,
// fetches the exercise catalog, picks 5 random exercises, sets them on the
// activity, verifies the exercise sets, and deletes the activity.
func TestStrengthActivityWithExercises(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Safety-net: clean up orphaned E2E activities from prior runs.
	cleanupOrphanActivities(t, client)

	// --- Step 1: Fetch exercise catalog ---
	t.Log("Fetching exercise catalog...")
	catalogData, err := garminapi.FetchExerciseCatalog(ctx)
	if err != nil {
		t.Fatalf("FetchExerciseCatalog failed: %v", err)
	}

	var catalog struct {
		Categories map[string]struct {
			Exercises map[string]json.RawMessage `json:"exercises"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		t.Fatalf("unmarshal exercise catalog: %v", err)
	}

	if len(catalog.Categories) == 0 {
		t.Fatal("exercise catalog has no categories")
	}
	t.Logf("Exercise catalog: %d categories", len(catalog.Categories))

	// --- Step 2: Pick 5 random exercises ---
	type exercise struct {
		Category string
		Name     string
	}

	var allExercises []exercise
	for cat, catData := range catalog.Categories {
		for name := range catData.Exercises {
			allExercises = append(allExercises, exercise{Category: cat, Name: name})
		}
	}

	if len(allExercises) < 5 {
		t.Fatalf("expected at least 5 exercises, got %d", len(allExercises))
	}

	// Shuffle and pick 5.
	rand.Shuffle(len(allExercises), func(i, j int) {
		allExercises[i], allExercises[j] = allExercises[j], allExercises[i]
	})
	picked := allExercises[:5]

	for i, ex := range picked {
		t.Logf("Exercise %d: %s/%s", i+1, ex.Category, ex.Name)
	}

	// --- Step 3: Create a strength training activity ---
	activityName := fmt.Sprintf("%s%d", e2eActivityPrefix, time.Now().UnixNano())
	now := time.Now()
	startTime := now.Format("2006-01-02T15:04:05.000")
	timezone := now.Location().String()

	data, err := client.CreateManualActivity(ctx, activityName, "strength_training", timezone, 0, 1800, startTime)
	if err != nil {
		t.Fatalf("CreateManualActivity failed: %v", err)
	}

	var created map[string]any
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatalf("unmarshal created activity: %v", err)
	}

	activityID := formatID(created["activityId"])
	if activityID == "" || activityID == "0" {
		t.Fatalf("expected valid activityId, got %q", activityID)
	}
	t.Logf("Created strength activity %s with name %q", activityID, activityName)

	// Register cleanup to delete the activity.
	RegisterCleanup(t, func() {
		if delErr := client.DeleteActivity(context.Background(), activityID); delErr != nil {
			t.Logf("WARNING: failed to clean up activity %s: %v", activityID, delErr)
		} else {
			t.Logf("Cleaned up activity %s", activityID)
		}
	})

	// --- Step 4: Set exercise sets on the activity ---
	type exerciseSetPayload struct {
		Exercises []map[string]any `json:"exercises"`
		RepCount  *int             `json:"repetitionCount"`
		Duration  *float64         `json:"duration"`
		Weight    float64          `json:"weight"`
		SetType   string           `json:"setType"`
		StartTime *string          `json:"startTime"`
	}

	var sets []exerciseSetPayload
	reps := []int{12, 10, 8, 15, 20}
	weights := []float64{20000, 15000, 30000, 10000, 25000} // milligrams

	for i, ex := range picked {
		r := reps[i]
		sets = append(sets, exerciseSetPayload{
			Exercises: []map[string]any{
				{"probability": 100, "category": ex.Category, "name": ex.Name},
			},
			RepCount: &r,
			Weight:   weights[i],
			SetType:  "ACTIVE",
		})

		// Add a rest set between exercises (not after the last one).
		if i < len(picked)-1 {
			dur := float64(30)
			sets = append(sets, exerciseSetPayload{
				Exercises: []map[string]any{},
				Duration:  &dur,
				Weight:    -1,
				SetType:   "REST",
			})
		}
	}

	payload := map[string]any{
		"activityId":   json.Number(activityID),
		"exerciseSets": sets,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal exercise sets: %v", err)
	}

	t.Logf("Setting %d exercise sets (5 active + 4 rest)...", len(sets))
	if err := client.SetExerciseSets(ctx, activityID, body); err != nil {
		t.Fatalf("SetExerciseSets failed: %v", err)
	}
	t.Log("Exercise sets saved successfully")

	// --- Step 5: Verify exercise sets on the activity ---
	esData, err := client.GetActivityExerciseSets(ctx, activityID)
	if err != nil {
		t.Fatalf("GetActivityExerciseSets failed: %v", err)
	}

	var esResult struct {
		ActivityID   json.Number `json:"activityId"`
		ExerciseSets []struct {
			SetType   string `json:"setType"`
			RepCount  *int   `json:"repetitionCount"`
			Weight    *float64
			Exercises []map[string]any `json:"exercises"`
		} `json:"exerciseSets"`
	}
	if err := json.Unmarshal(esData, &esResult); err != nil {
		t.Fatalf("unmarshal exercise sets: %v", err)
	}

	if esResult.ExerciseSets == nil {
		t.Fatal("expected non-nil exerciseSets after setting them")
	}

	activeCount := 0
	restCount := 0
	for _, s := range esResult.ExerciseSets {
		switch s.SetType {
		case "ACTIVE":
			activeCount++
		case "REST":
			restCount++
		}
	}

	t.Logf("Retrieved exercise sets: %d active, %d rest", activeCount, restCount)
	if activeCount != 5 {
		t.Errorf("expected 5 active sets, got %d", activeCount)
	}
	if restCount != 4 {
		t.Errorf("expected 4 rest sets, got %d", restCount)
	}

	// Verify that the exercise categories match what we sent.
	activeIdx := 0
	for _, s := range esResult.ExerciseSets {
		if s.SetType != "ACTIVE" {
			continue
		}
		if len(s.Exercises) == 0 {
			t.Errorf("active set %d has no exercises", activeIdx)
			activeIdx++
			continue
		}
		gotCategory, _ := s.Exercises[0]["category"].(string)
		wantCategory := picked[activeIdx].Category
		if !strings.EqualFold(gotCategory, wantCategory) {
			t.Errorf("set %d: category = %q, want %q", activeIdx, gotCategory, wantCategory)
		}
		activeIdx++
	}

	// --- Step 6: Verify activity appears in listing ---
	found := findActivityByID(t, client, activityID)
	if !found {
		t.Errorf("created activity %s not found in listing", activityID)
	}

	t.Log("E2E strength training test passed — cleanup will delete the activity")
}
