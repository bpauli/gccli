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

// TestCreateRenameDeleteGearCollection creates a gear collection, adds a new
// gear item to it, renames it and deletes it, with a GET after each step. The
// gear item must stay after the collection is deleted.
func TestCreateRenameDeleteGearCollection(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	cleanupOrphanGearCollections(t, client)
	cleanupOrphanGear(t, client)

	today := time.Now().Format("2006-01-02")
	name := fmt.Sprintf("%sCOLLECTION_%d", e2eGearPrefix, time.Now().UnixNano())
	body, err := json.Marshal(map[string]any{"name": name, "firstUseDate": today, "associatedActivityTypes": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := client.Call(ctx, "gear.collection.create", nil, body)
	if err != nil {
		t.Fatalf("gear.collection.create failed: %v", err)
	}
	collectionUUID := jsonUUID(t, data, "collection")
	t.Log("Created gear collection")

	collection := map[string]string{"collectionUUID": collectionUUID}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "gear.collection.delete", collection, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up gear collection: %v", err)
		}
	})

	data, err = client.CreateGear(ctx, garminapi.NewGear{
		Name:         name + "_GEAR",
		Type:         "SHOES",
		Brand:        e2eGearPrefix + "BRAND",
		Model:        e2eGearPrefix + "MODEL",
		FirstUseDate: today,
	})
	if err != nil {
		t.Fatalf("CreateGear failed: %v", err)
	}
	gearUUID := jsonUUID(t, data, "gear")
	gear := map[string]string{"gearUUID": gearUUID}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "gear.delete", gear, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up gear: %v", err)
		}
	})

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "gear.collection.add-gear", map[string]string{"collectionUUID": collectionUUID, "gearUUID": gearUUID}, nil); err != nil {
		t.Fatalf("gear.collection.add-gear failed: %v", err)
	}
	if got := readCollection(t, client, collectionUUID); !got.has(gearUUID) {
		t.Error("collection does not list the added gear")
	}

	time.Sleep(2 * time.Second)
	renamed := name + "_2"
	if _, err := client.RenameGearCollection(ctx, collectionUUID, renamed); err != nil {
		t.Fatalf("RenameGearCollection failed: %v", err)
	}
	got := readCollection(t, client, collectionUUID)
	if got.Name != renamed {
		t.Errorf("name after rename = %q, want %q", got.Name, renamed)
	}
	if !got.has(gearUUID) {
		t.Error("collection lost its gear after rename")
	}

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "gear.collection.delete", collection, nil); err != nil {
		t.Fatalf("gear.collection.delete failed: %v", err)
	}
	t.Log("Deleted gear collection")
	if listsCollection(t, client, collectionUUID) {
		t.Error("collection list still has the deleted collection")
	}
	if _, err := client.Call(ctx, "gear.get", gear, nil); err != nil {
		t.Errorf("gear item after collection delete: %v, want it to stay", err)
	}
}

type gearCollection struct {
	Name             string `json:"name"`
	GearInCollection []struct {
		UUID string `json:"uuid"`
	} `json:"gearInCollection"`
}

func (c gearCollection) has(gearUUID string) bool {
	for _, g := range c.GearInCollection {
		if g.UUID == gearUUID {
			return true
		}
	}
	return false
}

func readCollection(t *testing.T, client *garminapi.Client, uuid string) gearCollection {
	t.Helper()
	data, err := client.Call(context.Background(), "gear.collection.get", map[string]string{"collectionUUID": uuid}, nil)
	if err != nil {
		t.Fatalf("gear.collection.get failed: %v", err)
	}
	var c gearCollection
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal gear collection: %v", err)
	}
	return c
}

func listsCollection(t *testing.T, client *garminapi.Client, uuid string) bool {
	t.Helper()
	data, err := client.Call(context.Background(), "gear.collections", nil, nil)
	if err != nil {
		t.Fatalf("gear.collections failed: %v", err)
	}
	var list []struct {
		UUID string `json:"uuid"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("unmarshal gear collections: %v", err)
	}
	for _, c := range list {
		if c.UUID == uuid {
			return true
		}
	}
	return false
}

func jsonUUID(t *testing.T, data json.RawMessage, what string) string {
	t.Helper()
	var obj struct {
		UUID string `json:"uuid"`
	}
	if err := json.Unmarshal(data, &obj); err != nil || obj.UUID == "" {
		t.Fatalf("created %s has no uuid (err %v)", what, err)
	}
	return obj.UUID
}

// cleanupOrphanGearCollections deletes gear collections with the E2E_TEST_
// prefix left over from prior failed runs. The list gives the dashed UUID.
func cleanupOrphanGearCollections(t *testing.T, client *garminapi.Client) {
	t.Helper()
	ctx := context.Background()
	data, err := client.Call(ctx, "gear.collections", nil, nil)
	if err != nil {
		t.Logf("WARNING: could not list gear collections for orphan cleanup: %v", err)
		return
	}
	var list []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		t.Logf("WARNING: could not parse gear collections for orphan cleanup: %v", err)
		return
	}
	for _, c := range list {
		if !strings.HasPrefix(c.Name, e2eGearPrefix) || c.UUID == "" {
			continue
		}
		if _, err := client.Call(ctx, "gear.collection.delete", map[string]string{"collectionUUID": c.UUID}, nil); err != nil {
			t.Logf("WARNING: failed to delete orphaned gear collection: %v", err)
		} else {
			t.Log("Cleaned up orphaned gear collection")
		}
	}
}
