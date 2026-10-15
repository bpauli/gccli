//go:build e2e

package e2e

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/bpauli/gccli/internal/e2e/harness"
	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
)

// cachedTokens stores tokens across tests in the same test binary run so
// we only authenticate once against the real Garmin SSO.
// loginErr is stored on first auth failure so subsequent tests skip immediately.
var (
	cachedTokens *garminauth.Tokens
	loginErr     error
	tokensMu     sync.Mutex
)

func AuthenticatedClient(t *testing.T) *garminapi.Client {
	t.Helper()

	tokens := getOrLogin(t)
	return garminapi.NewClient(tokens)
}

func getOrLogin(t *testing.T) *garminauth.Tokens {
	t.Helper()

	tokensMu.Lock()
	defer tokensMu.Unlock()

	var rateLimited *harness.SSORateLimitError
	if errors.As(loginErr, &rateLimited) {
		t.Skip(loginErr)
	}
	if loginErr != nil {
		t.Skipf("skipping: prior auth failed: %v", loginErr)
	}

	if cachedTokens != nil && !cachedTokens.IsExpired() {
		return cachedTokens
	}

	email, password := LoadEnv(t)

	tokens, err := harness.Login(context.Background(), email, password, http.DefaultTransport)
	if err != nil {
		loginErr = err
		if errors.As(err, &rateLimited) {
			t.Skip(err)
		}
		t.Fatalf("E2E login failed: %v", err)
	}

	cachedTokens = tokens
	return cachedTokens
}
