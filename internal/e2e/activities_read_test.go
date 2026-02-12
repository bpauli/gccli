//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// TestListActivities verifies that GetActivities returns a valid JSON array.
func TestListActivities(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	data, err := client.GetActivities(ctx, 0, 5, "")
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}

	// Verify the response is a valid JSON array.
	var activities []json.RawMessage
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Fatalf("expected JSON array, got unmarshal error: %v", err)
	}

	t.Logf("GetActivities returned %d activities", len(activities))
}

// TestCountActivities verifies that CountActivities returns a non-negative count.
func TestCountActivities(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	count, err := client.CountActivities(ctx)
	if err != nil {
		t.Fatalf("CountActivities failed: %v", err)
	}

	if count < 0 {
		t.Errorf("expected count >= 0, got %d", count)
	}

	t.Logf("CountActivities returned %d", count)
}

// TestActivityDetails fetches the first activity and verifies that GetActivity
// returns a response containing an activityId field.
func TestActivityDetails(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Fetch first activity to get an ID.
	data, err := client.GetActivities(ctx, 0, 1, "")
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}

	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Fatalf("unmarshal activities: %v", err)
	}
	if len(activities) == 0 {
		t.Skip("no activities in account, skipping detail tests")
	}

	activityIDRaw, ok := activities[0]["activityId"]
	if !ok {
		t.Fatal("first activity missing activityId field")
	}

	// activityId is a JSON number; format it as a string for the API.
	activityID := formatID(activityIDRaw)

	// Fetch the activity by ID.
	detail, err := client.GetActivity(ctx, activityID)
	if err != nil {
		t.Fatalf("GetActivity(%s) failed: %v", activityID, err)
	}

	var detailMap map[string]any
	if err := json.Unmarshal(detail, &detailMap); err != nil {
		t.Fatalf("unmarshal activity detail: %v", err)
	}

	if _, ok := detailMap["activityId"]; !ok {
		t.Error("activity detail response missing activityId field")
	}

	t.Logf("GetActivity(%s) returned activity: %v", activityID, detailMap["activityName"])
}

// TestActivitySubResources fetches splits, weather, and HR zones for the first
// activity. These are read-only operations that require no cleanup.
func TestActivitySubResources(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Fetch first activity to get an ID.
	data, err := client.GetActivities(ctx, 0, 1, "")
	if err != nil {
		t.Fatalf("GetActivities failed: %v", err)
	}

	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Fatalf("unmarshal activities: %v", err)
	}
	if len(activities) == 0 {
		t.Skip("no activities in account, skipping sub-resource tests")
	}

	activityID := formatID(activities[0]["activityId"])

	// Test splits — may return data or empty depending on activity type.
	t.Run("splits", func(t *testing.T) {
		resp, err := client.GetActivitySplits(ctx, activityID)
		if err != nil {
			t.Fatalf("GetActivitySplits(%s) failed: %v", activityID, err)
		}
		if len(resp) == 0 {
			t.Error("expected non-empty splits response")
		}
	})

	// Test weather — may return data or empty depending on activity.
	t.Run("weather", func(t *testing.T) {
		resp, err := client.GetActivityWeather(ctx, activityID)
		if err != nil {
			t.Fatalf("GetActivityWeather(%s) failed: %v", activityID, err)
		}
		if len(resp) == 0 {
			t.Error("expected non-empty weather response")
		}
	})

	// Test HR zones — may return data or empty depending on activity type.
	t.Run("hr-zones", func(t *testing.T) {
		resp, err := client.GetActivityHRZones(ctx, activityID)
		if err != nil {
			t.Fatalf("GetActivityHRZones(%s) failed: %v", activityID, err)
		}
		if len(resp) == 0 {
			t.Error("expected non-empty HR zones response")
		}
	})
}

// formatID converts a JSON number (float64) or string to a string ID.
func formatID(v any) string {
	switch id := v.(type) {
	case float64:
		return json.Number(fmt.Sprintf("%.0f", id)).String()
	case json.Number:
		return id.String()
	case string:
		return id
	default:
		return fmt.Sprintf("%v", v)
	}
}
