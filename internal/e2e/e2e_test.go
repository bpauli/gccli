//go:build e2e

package e2e

import "testing"

// TestLoadEnv verifies that the .env loading and skip logic works.
// When credentials are missing, this test is skipped (not failed).
func TestLoadEnv(t *testing.T) {
	email, password := LoadEnv(t)

	if email == "" {
		t.Fatal("expected non-empty GARMIN_EMAIL after LoadEnv")
	}
	if password == "" {
		t.Fatal("expected non-empty GARMIN_PASSWORD after LoadEnv")
	}
}
