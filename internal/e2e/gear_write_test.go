//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

const e2eGearPrefix = "E2E_TEST_"

// TestCreateUpdateRetireDeleteGear adds a gear item, renames it, retires it,
// makes it active again and deletes it, with a GET after each step.
func TestCreateUpdateRetireDeleteGear(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	cleanupOrphanGear(t, client)

	name := fmt.Sprintf("%sGEAR_%d", e2eGearPrefix, time.Now().UnixNano())
	data, err := client.CreateGear(ctx, garminapi.NewGear{
		Name:         name,
		Type:         "SHOES",
		Brand:        e2eGearPrefix + "BRAND",
		Model:        e2eGearPrefix + "MODEL",
		FirstUseDate: time.Now().Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("CreateGear failed: %v", err)
	}
	var created map[string]any
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatalf("unmarshal created gear: %v", err)
	}
	uuid, _ := created["uuid"].(string)
	if uuid == "" {
		t.Fatal("created gear has no uuid")
	}
	t.Log("Created gear")

	params := map[string]string{"gearUUID": uuid}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "gear.delete", params, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up gear: %v", err)
		}
	})

	if got := gearField(t, client, uuid, "notes"); got != name {
		t.Errorf("notes = %q, want the name %q", got, name)
	}

	time.Sleep(2 * time.Second)
	renamed := name + "_2"
	if _, err := client.UpdateGear(ctx, uuid, garminapi.GearEdit{Name: renamed}); err != nil {
		t.Fatalf("UpdateGear failed: %v", err)
	}
	if got := gearField(t, client, uuid, "name"); got != renamed {
		t.Errorf("name after update = %q, want %q", got, renamed)
	}
	if got := gearField(t, client, uuid, "brand"); got != e2eGearPrefix+"BRAND" {
		t.Errorf("brand after update = %q, want it unchanged", got)
	}

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "gear.retire", params, nil); err != nil {
		t.Fatalf("gear.retire failed: %v", err)
	}
	if got := gearField(t, client, uuid, "status"); !strings.EqualFold(got, "RETIRED") {
		t.Errorf("status after retire = %q, want RETIRED", got)
	}

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "gear.unretire", params, nil); err != nil {
		t.Fatalf("gear.unretire failed: %v", err)
	}
	if got := gearField(t, client, uuid, "status"); !strings.EqualFold(got, "ACTIVE") {
		t.Errorf("status after unretire = %q, want ACTIVE", got)
	}

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "gear.delete", params, nil); err != nil {
		t.Fatalf("gear.delete failed: %v", err)
	}
	t.Log("Deleted gear")
	if _, err := client.Call(ctx, "gear.get", params, nil); !isNotFound(err) {
		t.Errorf("GET after delete: err = %v, want 404", err)
	}
}

// gearField reads one string field of a gear item.
func gearField(t *testing.T, client *garminapi.Client, uuid, key string) string {
	t.Helper()
	data, err := client.Call(context.Background(), "gear.get", map[string]string{"gearUUID": uuid}, nil)
	if err != nil {
		t.Fatalf("gear.get failed: %v", err)
	}
	var gear map[string]any
	if err := json.Unmarshal(data, &gear); err != nil {
		t.Fatalf("unmarshal gear: %v", err)
	}
	value, _ := gear[key].(string)
	return value
}

func isNotFound(err error) bool {
	var apiErr *garminapi.GarminAPIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// cleanupOrphanGear deletes gear with the E2E_TEST_ prefix left over from
// prior failed runs. gear list gives the UUID without dashes, and the v2
// endpoints need the dashed form.
func cleanupOrphanGear(t *testing.T, client *garminapi.Client) {
	t.Helper()
	ctx := context.Background()

	pk, err := client.GetProfilePK(ctx)
	if err != nil {
		t.Logf("WARNING: could not read the profile for orphan cleanup: %v", err)
		return
	}
	data, err := client.Call(ctx, "gear.list", map[string]string{"userProfilePK": pk}, nil)
	if err != nil {
		t.Logf("WARNING: could not list gear for orphan cleanup: %v", err)
		return
	}
	var gear []map[string]any
	if err := json.Unmarshal(data, &gear); err != nil {
		t.Logf("WARNING: could not parse gear for orphan cleanup: %v", err)
		return
	}
	for _, g := range gear {
		name, _ := g["displayName"].(string)
		hex, _ := g["uuid"].(string)
		if !strings.HasPrefix(name, e2eGearPrefix) || len(hex) != 32 {
			continue
		}
		uuid := strings.Join([]string{hex[:8], hex[8:12], hex[12:16], hex[16:20], hex[20:]}, "-")
		if _, err := client.Call(ctx, "gear.delete", map[string]string{"gearUUID": uuid}, nil); err != nil {
			t.Logf("WARNING: failed to delete orphaned gear: %v", err)
		} else {
			t.Log("Cleaned up orphaned gear")
		}
	}
}
