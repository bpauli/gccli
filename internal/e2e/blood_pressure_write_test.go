//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

const e2eBPNotes = "E2E_TEST_blood_pressure"

// TestAddDeleteBloodPressure adds a blood pressure reading for now, finds
// it in the range view of today by its notes and deletes it by its date and
// version.
func TestAddDeleteBloodPressure(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	now := time.Now()
	date := now.Format("2006-01-02")
	cleanupOrphanBPReadings(t, client, date)

	if _, err := client.AddBloodPressure(ctx, 111, 77, 66, now.Format(garminTimestamp), now.UTC().Format(garminTimestamp), e2eBPNotes); err != nil {
		t.Fatalf("AddBloodPressure failed: %v", err)
	}
	t.Log("Added blood pressure reading")

	// A failure before the read leaves the reading to the orphan sweep of
	// the next run.
	added := findTestBPReadings(t, client, date)
	if len(added) != 1 {
		t.Fatalf("range view holds %d test readings, want 1", len(added))
	}
	if r := added[0]; r.Systolic != 111 || r.Diastolic != 77 || r.Pulse != 66 {
		t.Errorf("reading = %d/%d pulse %d, want 111/77 pulse 66", r.Systolic, r.Diastolic, r.Pulse)
	}

	params := map[string]string{"date": date, "version": added[0].Version.String()}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "body.delete-blood-pressure", params, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up blood pressure reading: %v", err)
		}
	})

	if _, err := client.Call(ctx, "body.delete-blood-pressure", params, nil); err != nil {
		t.Fatalf("body.delete-blood-pressure failed: %v", err)
	}
	t.Log("Deleted blood pressure reading")
	if left := findTestBPReadings(t, client, date); len(left) != 0 {
		t.Errorf("range view still holds %d test readings after the delete", len(left))
	}
}

type bpReading struct {
	Version   json.Number `json:"version"`
	Systolic  int         `json:"systolic"`
	Diastolic int         `json:"diastolic"`
	Pulse     int         `json:"pulse"`
	Notes     string      `json:"notes"`
}

// findTestBPReadings returns the readings of date whose notes have the
// E2E_TEST_ prefix.
func findTestBPReadings(t *testing.T, client *garminapi.Client, date string) []bpReading {
	t.Helper()
	data, err := client.Call(context.Background(), "body.blood-pressure", map[string]string{"startDate": date, "endDate": date}, nil)
	if err != nil {
		t.Fatalf("body.blood-pressure failed: %v", err)
	}
	var view struct {
		Summaries []struct {
			Measurements []bpReading `json:"measurements"`
		} `json:"measurementSummaries"`
	}
	if err := json.Unmarshal(data, &view); err != nil {
		t.Fatalf("unmarshal range view: %v", err)
	}
	var found []bpReading
	for _, s := range view.Summaries {
		for _, r := range s.Measurements {
			if strings.HasPrefix(r.Notes, "E2E_TEST_") {
				found = append(found, r)
			}
		}
	}
	return found
}

// cleanupOrphanBPReadings deletes the readings of date with the E2E_TEST_
// prefix, left over from prior failed runs.
func cleanupOrphanBPReadings(t *testing.T, client *garminapi.Client, date string) {
	t.Helper()
	for _, r := range findTestBPReadings(t, client, date) {
		params := map[string]string{"date": date, "version": r.Version.String()}
		if _, err := client.Call(context.Background(), "body.delete-blood-pressure", params, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to delete orphaned blood pressure reading: %v", err)
		} else {
			t.Log("Cleaned up orphaned blood pressure reading")
		}
	}
}
