//go:build e2e

package e2e

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/bpauli/gccli/internal/e2e/harness"
	"github.com/bpauli/gccli/internal/garminapi"
)

var (
	shared   *harness.SharedTokens
	loginErr error
	tokensMu sync.Mutex
)

func AuthenticatedClient(t *testing.T) *garminapi.Client {
	t.Helper()

	s := getOrLogin(t)
	client := garminapi.NewClient(s.Tokens())
	client.Refresher = s.Refresh
	return client
}

func getOrLogin(t *testing.T) *harness.SharedTokens {
	t.Helper()

	tokensMu.Lock()
	defer tokensMu.Unlock()

	if loginErr == nil && shared != nil {
		loginErr = shared.Err()
	}

	var rateLimited *harness.SSORateLimitError
	if errors.As(loginErr, &rateLimited) {
		t.Skip(loginErr)
	}
	if loginErr != nil {
		t.Skipf("skipping: prior auth failed: %v", loginErr)
	}

	if raw := os.Getenv("GARMIN_TOKEN"); raw != "" {
		if shared == nil {
			tokens, err := harness.LoadToken(context.Background(), raw, http.DefaultTransport)
			if err != nil {
				loginErr = err
				t.Fatalf("E2E token: %v", err)
			}
			shared = harness.NewSharedTokens(tokens, http.DefaultTransport)
		}
		return shared
	}

	if shared != nil && !shared.Tokens().IsExpired() {
		return shared
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

	shared = harness.NewSharedTokens(tokens, http.DefaultTransport)
	return shared
}
