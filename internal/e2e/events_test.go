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

const e2eEventPrefix = "E2E_TEST_"

// TestCreateListDeleteEvent creates a calendar event, verifies it appears in
// the listing, and deletes it.
func TestCreateListDeleteEvent(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Safety-net: clean up orphaned E2E events from prior runs.
	cleanupOrphanEvents(t, client)

	eventName := fmt.Sprintf("%sEVENT_%d", e2eEventPrefix, time.Now().UnixNano())
	eventDate := time.Now().AddDate(0, 3, 0).Format("2006-01-02")

	// Step 1: Create event with goal and training priority.
	payload := json.RawMessage(fmt.Sprintf(`{
		"eventName": %q,
		"date": %q,
		"eventType": "running",
		"race": true,
		"eventPrivacy": {"label": "PRIVATE"},
		"completionTarget": {"value": 10, "unit": "kilometer", "unitType": "distance"},
		"eventCustomization": {
			"customGoal": {"value": 3000, "unit": "second", "unitType": "time"},
			"isPrimaryEvent": false,
			"isTrainingEvent": false
		}
	}`, eventName, eventDate))

	data, err := client.AddEvent(ctx, payload)
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	var created map[string]any
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatalf("unmarshal created event: %v", err)
	}

	eventID := formatID(created["id"])
	if eventID == "" || eventID == "0" {
		t.Fatalf("expected valid event id, got %q", eventID)
	}
	t.Logf("Created event %s with name %q on %s", eventID, eventName, eventDate)

	// Verify eventCustomization was accepted.
	if cust, ok := created["eventCustomization"].(map[string]any); ok {
		if goal, ok := cust["customGoal"].(map[string]any); ok {
			if goal["value"] != float64(3000) {
				t.Errorf("customGoal value = %v, want 3000", goal["value"])
			}
			t.Logf("Event has custom goal: %v seconds", goal["value"])
		} else {
			t.Error("eventCustomization.customGoal not present in response")
		}
	} else {
		t.Error("eventCustomization not present in response")
	}

	// Register cleanup to delete the event — runs even on test failure.
	RegisterCleanup(t, func() {
		if delErr := client.DeleteEvent(context.Background(), eventID); delErr != nil {
			t.Logf("WARNING: failed to clean up event %s: %v", eventID, delErr)
		} else {
			t.Logf("Cleaned up event %s", eventID)
		}
	})

	// Brief pause to avoid rate limiting.
	time.Sleep(2 * time.Second)

	// Step 2: Verify the event appears in the listing.
	found := findEventByID(t, client, eventID, eventDate)
	if !found {
		t.Errorf("created event %s not found in GetEvents listing", eventID)
	}

	// Brief pause to avoid rate limiting.
	time.Sleep(2 * time.Second)

	// Step 3: Delete the event explicitly (cleanup is the safety net).
	if err := client.DeleteEvent(ctx, eventID); err != nil {
		t.Fatalf("DeleteEvent(%s) failed: %v", eventID, err)
	}
	t.Logf("Deleted event %s", eventID)

	// Brief pause to allow deletion to propagate.
	time.Sleep(2 * time.Second)

	// Verify it no longer appears in the listing.
	if findEventByID(t, client, eventID, eventDate) {
		t.Errorf("event %s still appears in listing after delete", eventID)
	}
}

// findEventByID searches events from a start date for one with the given ID.
func findEventByID(t *testing.T, client *garminapi.Client, eventID, startDate string) bool {
	t.Helper()

	data, err := client.GetEvents(context.Background(), startDate, 1, 100, "eventDate_asc")
	if err != nil {
		t.Fatalf("GetEvents failed while searching for event %s: %v", eventID, err)
	}

	var events []map[string]any
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("unmarshal events: %v", err)
	}

	for _, e := range events {
		if formatID(e["id"]) == eventID {
			return true
		}
	}
	return false
}

// cleanupOrphanEvents deletes any events with the E2E_TEST_ prefix
// left over from prior failed runs.
func cleanupOrphanEvents(t *testing.T, client *garminapi.Client) {
	t.Helper()
	ctx := context.Background()

	startDate := time.Now().Format("2006-01-02")
	data, err := client.GetEvents(ctx, startDate, 1, 100, "eventDate_asc")
	if err != nil {
		t.Logf("WARNING: could not list events for orphan cleanup: %v", err)
		return
	}

	var events []map[string]any
	if err := json.Unmarshal(data, &events); err != nil {
		t.Logf("WARNING: could not parse events for orphan cleanup: %v", err)
		return
	}

	for _, e := range events {
		name, _ := e["eventName"].(string)
		if strings.HasPrefix(name, e2eEventPrefix) {
			id := formatID(e["id"])
			if delErr := client.DeleteEvent(ctx, id); delErr != nil {
				t.Logf("WARNING: failed to delete orphaned event %s (%s): %v", id, name, delErr)
			} else {
				t.Logf("Cleaned up orphaned event %s (%s)", id, name)
			}
		}
	}
}
