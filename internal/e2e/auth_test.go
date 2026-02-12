//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
)

// TestHeadlessLogin verifies that headless SSO login returns valid tokens
// with all expected fields populated.
func TestHeadlessLogin(t *testing.T) {
	email, password := LoadEnv(t)

	tokens, err := garminauth.LoginHeadless(
		context.Background(),
		email,
		password,
		garminauth.LoginOptions{},
	)
	if err != nil {
		t.Fatalf("LoginHeadless failed: %v", err)
	}

	// Verify OAuth2 access token is present.
	if tokens.OAuth2AccessToken == "" {
		t.Error("expected non-empty OAuth2AccessToken")
	}

	// Verify OAuth2 refresh token is present.
	if tokens.OAuth2RefreshToken == "" {
		t.Error("expected non-empty OAuth2RefreshToken")
	}

	// Verify OAuth1 credentials are present (needed for token refresh).
	if tokens.OAuth1Token == "" {
		t.Error("expected non-empty OAuth1Token")
	}
	if tokens.OAuth1Secret == "" {
		t.Error("expected non-empty OAuth1Secret")
	}
}

// TestTokenFields verifies that login populates metadata fields correctly:
// Email, Domain, and OAuth2ExpiresAt.
func TestTokenFields(t *testing.T) {
	email, _ := LoadEnv(t)

	// Use the cached authenticated client path to get tokens.
	tokens := getOrLogin(t)

	// Verify email matches the login email.
	if tokens.Email != email {
		t.Errorf("expected Email=%q, got %q", email, tokens.Email)
	}

	// Verify domain is set (should be garmin.com for global accounts).
	if tokens.Domain == "" {
		t.Error("expected non-empty Domain")
	}
	if tokens.Domain != garminauth.DomainGlobal && tokens.Domain != garminauth.DomainChina {
		t.Errorf("expected Domain to be %q or %q, got %q",
			garminauth.DomainGlobal, garminauth.DomainChina, tokens.Domain)
	}

	// Verify OAuth2ExpiresAt is in the future (token should be fresh).
	if tokens.OAuth2ExpiresAt.IsZero() {
		t.Error("expected non-zero OAuth2ExpiresAt")
	}
	if tokens.OAuth2ExpiresAt.Before(time.Now()) {
		t.Error("expected OAuth2ExpiresAt to be in the future")
	}

	// Token should not be expired (just authenticated).
	if tokens.IsExpired() {
		t.Error("expected token to not be expired immediately after login")
	}
}

// TestAuthStatus verifies that after login, we can create an authenticated
// client and make a basic API call to confirm the tokens are valid.
func TestAuthStatus(t *testing.T) {
	client := AuthenticatedClient(t)

	// Make a lightweight API call to verify tokens work.
	// GetProfile is a simple GET that requires authentication.
	data, err := client.ConnectAPI(context.Background(), "GET", "/userprofile-service/usersocialprofile", nil)
	if err != nil {
		t.Fatalf("API call with authenticated client failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty response from profile API")
	}
}
