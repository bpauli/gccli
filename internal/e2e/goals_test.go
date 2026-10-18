//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
)

func TestListGoals(t *testing.T) {
	client := AuthenticatedClient(t)

	data, err := client.Call(context.Background(), "goals.list", map[string]string{"status": "active"}, nil)
	if err != nil {
		t.Fatalf("goals.list failed: %v", err)
	}
	if !json.Valid(data) {
		t.Fatal("goals.list returned invalid JSON")
	}
}
