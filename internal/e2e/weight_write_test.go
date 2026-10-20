//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// e2eWeightKG is the weight of the test weigh-in. A weigh-in has no name, so
// the orphan sweep knows a test weigh-in by this value.
const e2eWeightKG = 42.42

const garminTimestamp = "2006-01-02T15:04:05.000"

// TestAddDeleteWeight adds a weigh-in for now, finds it in the day view by
// its weight and deletes it by its date and samplePk.
func TestAddDeleteWeight(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	now := time.Now()
	date := now.Format("2006-01-02")
	cleanupOrphanWeighIns(t, client, date)

	if _, err := client.AddWeight(ctx, e2eWeightKG, "kg", now.Format(garminTimestamp), now.UTC().Format(garminTimestamp)); err != nil {
		t.Fatalf("AddWeight failed: %v", err)
	}
	t.Log("Added weigh-in")

	// A failure before the read leaves the weigh-in to the orphan sweep of
	// the next run.
	added := findTestWeighIns(t, client, date)
	if len(added) != 1 {
		t.Fatalf("day view holds %d test weigh-ins, want 1", len(added))
	}

	params := map[string]string{"date": date, "version": added[0].SamplePK.String()}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "body.delete-weight", params, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up weigh-in: %v", err)
		}
	})

	if _, err := client.Call(ctx, "body.delete-weight", params, nil); err != nil {
		t.Fatalf("body.delete-weight failed: %v", err)
	}
	t.Log("Deleted weigh-in")
	if left := findTestWeighIns(t, client, date); len(left) != 0 {
		t.Errorf("day view still holds %d test weigh-ins after the delete", len(left))
	}
}

type weighIn struct {
	SamplePK json.Number `json:"samplePk"`
	Grams    float64     `json:"weight"`
}

// findTestWeighIns returns the weigh-ins of date that have the test weight.
// The day view gives the weight in grams.
func findTestWeighIns(t *testing.T, client *garminapi.Client, date string) []weighIn {
	t.Helper()
	data, err := client.Call(context.Background(), "body.daily-weigh-ins", map[string]string{"date": date}, nil)
	if err != nil {
		t.Fatalf("body.daily-weigh-ins failed: %v", err)
	}
	var view struct {
		List []weighIn `json:"dateWeightList"`
	}
	if err := json.Unmarshal(data, &view); err != nil {
		t.Fatalf("unmarshal day view: %v", err)
	}
	var found []weighIn
	for _, w := range view.List {
		if math.Abs(w.Grams-e2eWeightKG*1000) < 1 {
			found = append(found, w)
		}
	}
	return found
}

// cleanupOrphanWeighIns deletes the weigh-ins of date that have the test
// weight, left over from prior failed runs.
func cleanupOrphanWeighIns(t *testing.T, client *garminapi.Client, date string) {
	t.Helper()
	for _, w := range findTestWeighIns(t, client, date) {
		params := map[string]string{"date": date, "version": w.SamplePK.String()}
		if _, err := client.Call(context.Background(), "body.delete-weight", params, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to delete orphaned weigh-in: %v", err)
		} else {
			t.Log("Cleaned up orphaned weigh-in")
		}
	}
}
