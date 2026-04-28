package cmd

import (
	"fmt"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/secrets"
)

// newClientFn creates a garminapi.Client from tokens.
// Variable to allow overriding in tests.
var newClientFn = defaultNewClient

func defaultNewClient(tokens *garminauth.Tokens) *garminapi.Client {
	return garminapi.NewClient(tokens)
}

// proactiveRefreshFn is the function used for proactive token refresh.
// Variable to allow overriding in tests.
var proactiveRefreshFn = garminauth.RefreshOAuth2

// resolveClient loads stored tokens for the account and returns an API client.
// If the stored token is already expired and refresh credentials are available,
// it proactively refreshes the token before creating the client to avoid
// a wasted 401 round-trip. Refreshed tokens are persisted back to the keyring.
func resolveClient(g *Globals) (*garminapi.Client, error) {
	email := g.Account
	if email == "" {
		return nil, fmt.Errorf("no account specified; use --account, set GCCLI_ACCOUNT, or run gccli auth login")
	}

	store, err := loadSecretsStore()
	if err != nil {
		return nil, err
	}

	data, err := store.Get(email)
	if err != nil {
		if err == secrets.ErrNotFound {
			return nil, fmt.Errorf("no credentials stored for %s; run: gccli auth login %s", email, email)
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	tokens, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		return nil, fmt.Errorf("parse tokens: %w", err)
	}

	// Proactive refresh: if the token is already expired, refresh before
	// creating the client to avoid a wasted 401 round-trip.
	if tokens.IsExpired() && tokens.CanRefresh() {
		newTokens, refreshErr := proactiveRefreshFn(g.Context, tokens, garminauth.LoginOptions{
			Domain: tokens.Domain,
		})
		if refreshErr == nil {
			tokens = newTokens
			persistTokens(store, email, tokens)
		}
		// On refresh failure, proceed with expired tokens — the client's
		// 401 retry path may still succeed.
	}

	client := newClientFn(tokens)

	// Wire up OnTokenRefresh to persist tokens after 401-triggered refresh.
	client.OnTokenRefresh = func(refreshed *garminauth.Tokens) {
		persistTokens(store, email, refreshed)
	}

	return client, nil
}

// persistTokens writes tokens to the keyring. Errors are silently ignored
// because this is a best-effort operation — the client still works with
// in-memory tokens.
func persistTokens(store *secrets.Store, email string, tokens *garminauth.Tokens) {
	data, err := tokens.Marshal()
	if err != nil {
		return
	}
	_ = store.Set(email, data)
}
