//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
)

// TestGetProfile calls GetProfile and verifies the response contains displayName.
func TestGetProfile(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	data, err := client.GetProfile(ctx)
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}

	var profile map[string]any
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatalf("unmarshal profile: %v", err)
	}

	if _, ok := profile["displayName"]; !ok {
		t.Error("profile response missing displayName field")
	}

	t.Logf("GetProfile returned profile for: %v", profile["displayName"])
}
