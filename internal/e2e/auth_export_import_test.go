//go:build e2e

package e2e

import (
	"encoding/base64"
	"testing"

	"github.com/bpauli/gccli/internal/garminauth"
)

// TestAuthExportImport verifies that tokens can be exported as base64 and
// imported back, producing an identical token set. Token values are never
// logged to avoid leaking credentials in CI output.
func TestAuthExportImport(t *testing.T) {
	original := getOrLogin(t)

	// --- Export: marshal and base64-encode ---
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("marshal tokens: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	if encoded == "" {
		t.Fatal("expected non-empty base64 export string")
	}

	// Verify the base64 output does NOT contain raw token values.
	// This ensures that even if test output is captured (-v), secrets
	// are not readable.
	for _, secret := range []string{
		original.OAuth1Token,
		original.OAuth1Secret,
		original.OAuth2AccessToken,
		original.OAuth2RefreshToken,
	} {
		if secret == "" {
			continue
		}
		if containsSubstring(encoded, secret) {
			t.Error("exported base64 string contains a raw token value — secrets must be opaque")
		}
	}

	// --- Import: base64-decode and unmarshal ---
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	imported, err := garminauth.UnmarshalTokens(decoded)
	if err != nil {
		t.Fatalf("unmarshal imported tokens: %v", err)
	}

	// Verify all fields round-trip correctly without logging actual values.
	assertFieldEqual(t, "Email", imported.Email, original.Email)
	assertFieldEqual(t, "Domain", imported.Domain, original.Domain)
	assertFieldNonEmpty(t, "OAuth1Token", imported.OAuth1Token)
	assertFieldNonEmpty(t, "OAuth1Secret", imported.OAuth1Secret)
	assertFieldNonEmpty(t, "OAuth2AccessToken", imported.OAuth2AccessToken)
	assertFieldNonEmpty(t, "OAuth2RefreshToken", imported.OAuth2RefreshToken)

	if !imported.OAuth2ExpiresAt.Equal(original.OAuth2ExpiresAt) {
		t.Error("OAuth2ExpiresAt mismatch after round-trip")
	}

	// Verify the imported token set matches the original byte-for-byte.
	reimported, err := imported.Marshal()
	if err != nil {
		t.Fatalf("re-marshal imported tokens: %v", err)
	}
	if string(reimported) != string(data) {
		t.Error("re-marshaled tokens differ from original — round-trip not lossless")
	}
}

// containsSubstring checks if s contains sub. Defined here to avoid
// importing strings (keeps the test file minimal).
func containsSubstring(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// assertFieldEqual fails if got != want, but only logs field name, not values.
func assertFieldEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: imported value does not match original", field)
	}
}

// assertFieldNonEmpty fails if value is empty, without logging the value.
func assertFieldNonEmpty(t *testing.T, field, value string) {
	t.Helper()
	if value == "" {
		t.Errorf("%s: expected non-empty value after import", field)
	}
}
