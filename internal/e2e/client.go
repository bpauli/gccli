//go:build e2e

package e2e

import (
	"context"
	"sync"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
)

// cachedTokens stores tokens across tests in the same test binary run so
// we only authenticate once against the real Garmin SSO.
var (
	cachedTokens *garminauth.Tokens
	tokensMu     sync.Mutex
)

// AuthenticatedClient returns a *garminapi.Client that is authenticated
// against the real Garmin Connect API using credentials from .env.
//
// Tokens are cached for the lifetime of the test binary so that multiple
// tests share a single SSO login. If authentication fails, the test is
// marked as fatal.
func AuthenticatedClient(t *testing.T) *garminapi.Client {
	t.Helper()

	tokens := getOrLogin(t)
	return garminapi.NewClient(tokens)
}

// getOrLogin returns cached tokens or performs a headless SSO login.
func getOrLogin(t *testing.T) *garminauth.Tokens {
	t.Helper()

	tokensMu.Lock()
	defer tokensMu.Unlock()

	if cachedTokens != nil && !cachedTokens.IsExpired() {
		return cachedTokens
	}

	email, password := LoadEnv(t)

	tokens, err := garminauth.LoginHeadless(
		context.Background(),
		email,
		password,
		garminauth.LoginOptions{},
	)
	if err != nil {
		t.Fatalf("E2E login failed: %v", err)
	}

	cachedTokens = tokens
	return cachedTokens
}
