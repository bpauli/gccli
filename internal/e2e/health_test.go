//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestHealthSummary calls GetDailySummary for today and verifies valid JSON.
func TestHealthSummary(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	tokens := getOrLogin(t)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetDailySummary(ctx, tokens.DisplayName, today)
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
	tokens := getOrLogin(t)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetSteps(ctx, tokens.DisplayName, today)
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
	tokens := getOrLogin(t)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetHeartRate(ctx, tokens.DisplayName, today)
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
	tokens := getOrLogin(t)
	today := time.Now().Format("2006-01-02")

	data, err := client.GetSleep(ctx, tokens.DisplayName, today)
	if err != nil {
		t.Fatalf("GetSleep failed: %v", err)
	}

	if !json.Valid(data) {
		t.Error("expected valid JSON from GetSleep")
	}
	t.Logf("GetSleep returned %d bytes", len(data))
}
