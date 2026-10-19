//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
)

// TestAddDeletePrivacyZone adds a privacy zone in the open sea and deletes
// it, with a list GET after each step. A zone has no name, so it cannot
// carry the E2E_TEST_ prefix, and the test has no orphan sweep: it cannot
// tell a zone it left behind from a zone of the account owner. The cleanup
// deletes the zone by the ID that the add returned.
func TestAddDeletePrivacyZone(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	body, err := json.Marshal(map[string]any{"centerLatitude": 0.0, "centerLongitude": -30.0, "radius": 500})
	if err != nil {
		t.Fatal(err)
	}
	data, err := client.Call(ctx, "profile.privacy-zone.create", nil, body)
	if err != nil {
		t.Fatalf("profile.privacy-zone.create failed: %v", err)
	}
	var created struct {
		ID json.Number `json:"privacyZoneId"`
	}
	if err := json.Unmarshal(data, &created); err != nil || created.ID == "" {
		t.Fatalf("created privacy zone has no privacyZoneId (err %v)", err)
	}
	zone := map[string]string{"privacyZoneID": created.ID.String()}
	t.Log("Added privacy zone")

	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "profile.privacy-zone.delete", zone, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up privacy zone: %v", err)
		}
	})

	if !slices.Contains(privacyZoneIDs(t, client), created.ID.String()) {
		t.Error("privacy zone list does not have the added zone")
	}

	if _, err := client.Call(ctx, "profile.privacy-zone.delete", zone, nil); err != nil {
		t.Fatalf("profile.privacy-zone.delete failed: %v", err)
	}
	t.Log("Deleted privacy zone")
	if slices.Contains(privacyZoneIDs(t, client), created.ID.String()) {
		t.Error("privacy zone list still has the deleted zone")
	}
}

func privacyZoneIDs(t *testing.T, client *garminapi.Client) []string {
	t.Helper()
	data, err := client.Call(context.Background(), "profile.privacy-zones", nil, nil)
	if err != nil {
		t.Fatalf("profile.privacy-zones failed: %v", err)
	}
	var zones []struct {
		ID json.Number `json:"privacyZoneId"`
	}
	if err := json.Unmarshal(data, &zones); err != nil {
		t.Fatalf("unmarshal privacy zones: %v", err)
	}
	ids := make([]string, len(zones))
	for i, z := range zones {
		ids[i] = z.ID.String()
	}
	return ids
}
