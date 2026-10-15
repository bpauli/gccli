//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

func displayNameOf(t *testing.T, client *garminapi.Client) string {
	t.Helper()
	name, err := client.GetDisplayName(context.Background())
	if err != nil {
		t.Fatalf("GetDisplayName failed: %v", err)
	}
	return name
}

// TestHealthSummary calls GetDailySummary for today and verifies valid JSON.
func TestHealthSummary(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	displayName := displayNameOf(t, client)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetDailySummary(ctx, displayName, today)
	if err != nil {
		t.Fatalf("GetDailySummary failed: %v", err)
	}

	// Verify it's valid JSON.
	if !json.Valid(data) {
		t.Error("expected valid JSON from GetDailySummary")
	}
	t.Logf("GetDailySummary returned %d bytes", len(data))
}

// TestHealthSteps calls GetSteps for today and verifies valid JSON.
func TestHealthSteps(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	displayName := displayNameOf(t, client)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetSteps(ctx, displayName, today)
	if err != nil {
		t.Fatalf("GetSteps failed: %v", err)
	}

	if !json.Valid(data) {
		t.Error("expected valid JSON from GetSteps")
	}
	t.Logf("GetSteps returned %d bytes", len(data))
}

// TestHealthHeartRate calls GetHeartRate for today and verifies valid JSON.
func TestHealthHeartRate(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	displayName := displayNameOf(t, client)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetHeartRate(ctx, displayName, today)
	if err != nil {
		t.Fatalf("GetHeartRate failed: %v", err)
	}

	if !json.Valid(data) {
		t.Error("expected valid JSON from GetHeartRate")
	}
	t.Logf("GetHeartRate returned %d bytes", len(data))
}

// TestHealthSleep calls GetSleep for today and verifies valid JSON.
func TestHealthSleep(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	displayName := displayNameOf(t, client)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetSleep(ctx, displayName, today)
	if err != nil {
		t.Fatalf("GetSleep failed: %v", err)
	}

	if !json.Valid(data) {
		t.Error("expected valid JSON from GetSleep")
	}
	t.Logf("GetSleep returned %d bytes", len(data))
}

// TestHealthRestingHeartRate calls GetRestingHeartRate for today and verifies valid JSON.
func TestHealthRestingHeartRate(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	displayName := displayNameOf(t, client)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetRestingHeartRate(ctx, displayName, today)
	if err != nil {
		t.Fatalf("GetRestingHeartRate failed: %v", err)
	}

	if !json.Valid(data) {
		t.Error("expected valid JSON from GetRestingHeartRate")
	}
	t.Logf("GetRestingHeartRate returned %d bytes", len(data))
}

// TestPersonalRecords calls GetPersonalRecords and verifies valid JSON.
func TestPersonalRecords(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	displayName := displayNameOf(t, client)

	data, err := client.GetPersonalRecords(ctx, displayName)
	if err != nil {
		t.Fatalf("GetPersonalRecords failed: %v", err)
	}

	if !json.Valid(data) {
		t.Error("expected valid JSON from GetPersonalRecords")
	}
	t.Logf("GetPersonalRecords returned %d bytes", len(data))
}
