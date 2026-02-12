package testutil

import (
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
)

// TestTokens returns a Tokens fixture suitable for testing.
// All fields are populated with realistic but obviously fake values.
func TestTokens() *garminauth.Tokens {
	return &garminauth.Tokens{
		OAuth1Token:        "test-oauth1-token",
		OAuth1Secret:       "test-oauth1-secret",
		OAuth2AccessToken:  "test-access-token",
		OAuth2RefreshToken: "test-refresh-token",
		OAuth2ExpiresAt:    time.Now().Add(time.Hour),
		Domain:             garminauth.DomainGlobal,
		Email:              "test@example.com",
		DisplayName:        "Test User",
	}
}

// ExpiredTokens returns a Tokens fixture with an expired OAuth2 token.
func ExpiredTokens() *garminauth.Tokens {
	t := TestTokens()
	t.OAuth2ExpiresAt = time.Now().Add(-time.Hour)
	return t
}

// TokensWithoutOAuth1 returns a Tokens fixture without OAuth1 credentials.
func TokensWithoutOAuth1() *garminauth.Tokens {
	t := TestTokens()
	t.OAuth1Token = ""
	t.OAuth1Secret = ""
	return t
}
