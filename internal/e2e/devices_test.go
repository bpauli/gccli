//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
)

// TestGetDevices calls GetDevices and verifies the response is a valid JSON array.
func TestGetDevices(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	data, err := client.GetDevices(ctx)
	if err != nil {
		t.Fatalf("GetDevices failed: %v", err)
	}

	// Verify the response is a valid JSON array.
	var devices []json.RawMessage
	if err := json.Unmarshal(data, &devices); err != nil {
		t.Fatalf("expected JSON array from GetDevices, got: %v", err)
	}

	t.Logf("GetDevices returned %d devices", len(devices))
}
