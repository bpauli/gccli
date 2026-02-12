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

const e2eActivityPrefix = "E2E_TEST_"

// TestCreateManualActivity creates a manual activity, verifies it appears in
// the listing, renames it, and deletes it via t.Cleanup.
func TestCreateManualActivity(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Safety-net: clean up orphaned E2E activities from prior runs.
	cleanupOrphanActivities(t, client)

	// Create a manual activity with E2E_TEST_ prefix.
	activityName := fmt.Sprintf("%s%d", e2eActivityPrefix, time.Now().UnixNano())
	startTime := time.Now().UTC().Format("2006-01-02T15:04:05.000")

	data, err := client.CreateManualActivity(ctx, activityName, "running", 1000, 600, startTime)
	if err != nil {
		t.Fatalf("CreateManualActivity failed: %v", err)
	}

	// Extract the created activity ID.
	var created map[string]any
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatalf("unmarshal created activity: %v", err)
	}

	activityID := formatID(created["activityId"])
	if activityID == "" || activityID == "0" {
		t.Fatalf("expected valid activityId, got %q", activityID)
	}
	t.Logf("Created activity %s with name %q", activityID, activityName)

	// Register cleanup to delete the activity — runs even on test failure.
	RegisterCleanup(t, func() {
		if delErr := client.DeleteActivity(context.Background(), activityID); delErr != nil {
			t.Logf("WARNING: failed to clean up activity %s: %v", activityID, delErr)
		} else {
			t.Logf("Cleaned up activity %s", activityID)
		}
	})

	// Verify the created activity appears in the listing.
	found := findActivityByID(t, client, activityID)
	if !found {
		t.Errorf("created activity %s not found in GetActivities listing", activityID)
	}

	// Test rename: rename to E2E_TEST_RENAMED.
	renamedName := e2eActivityPrefix + "RENAMED"
	_, err = client.RenameActivity(ctx, activityID, renamedName)
	if err != nil {
		t.Fatalf("RenameActivity(%s) failed: %v", activityID, err)
	}

	// Verify the name changed.
	detail, err := client.GetActivity(ctx, activityID)
	if err != nil {
		t.Fatalf("GetActivity(%s) after rename failed: %v", activityID, err)
	}

	var detailMap map[string]any
	if err := json.Unmarshal(detail, &detailMap); err != nil {
		t.Fatalf("unmarshal activity detail: %v", err)
	}

	if name, ok := detailMap["activityName"].(string); !ok || name != renamedName {
		t.Errorf("expected activityName=%q after rename, got %q", renamedName, name)
	}
	t.Logf("Successfully renamed activity %s to %q", activityID, renamedName)

	// The t.Cleanup handler will delete the activity.
	// After cleanup runs, the activity should be gone from the listing.
}

// findActivityByID searches recent activities for one with the given ID.
func findActivityByID(t *testing.T, client *garminapi.Client, activityID string) bool {
	t.Helper()
	ctx := context.Background()

	data, err := client.GetActivities(ctx, 0, 50, "")
	if err != nil {
		t.Fatalf("GetActivities failed while searching for activity %s: %v", activityID, err)
	}

	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Fatalf("unmarshal activities: %v", err)
	}

	for _, a := range activities {
		if formatID(a["activityId"]) == activityID {
			return true
		}
	}
	return false
}

// cleanupOrphanActivities deletes any activities with the E2E_TEST_ prefix
// left over from prior failed runs.
func cleanupOrphanActivities(t *testing.T, client *garminapi.Client) {
	t.Helper()
	ctx := context.Background()

	data, err := client.GetActivities(ctx, 0, 50, "")
	if err != nil {
		t.Logf("WARNING: could not list activities for orphan cleanup: %v", err)
		return
	}

	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Logf("WARNING: could not parse activities for orphan cleanup: %v", err)
		return
	}

	for _, a := range activities {
		name, _ := a["activityName"].(string)
		if strings.HasPrefix(name, e2eActivityPrefix) {
			id := formatID(a["activityId"])
			if delErr := client.DeleteActivity(ctx, id); delErr != nil {
				t.Logf("WARNING: failed to delete orphaned activity %s (%s): %v", id, name, delErr)
			} else {
				t.Logf("Cleaned up orphaned activity %s (%s)", id, name)
			}
		}
	}
}
