package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bpauli/gccli/internal/config"
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

// refreshFn is a variable to allow overriding in tests.
var refreshFn = garminauth.RefreshOAuth2

// resolveClient loads stored tokens for the account and returns an API client.
// If the stored token is already expired and refresh credentials are available,
// it proactively refreshes the token before creating the client to avoid
// a wasted 401 round-trip.
func resolveClient(g *Globals) (*garminapi.Client, error) {
	email := g.Account
	if email == "" {
		return nil, errNoAccount()
	}

	store, err := loadSecretsStore()
	if err != nil {
		return nil, err
	}

	tokens, err := loadTokens(store, email)
	if err != nil {
		return nil, err
	}

	r := &tokenRefresher{store: store, email: email}
	if tokens.IsExpired() && tokens.CanRefresh() {
		fresh, err := r.refresh(g.Context, tokens)
		var expired *garminapi.TokenExpiredError
		switch {
		case err == nil:
			tokens = fresh
		case errors.As(err, &expired):
			// IsExpired counts a token as expired 60 s early, so Garmin may still accept it.
		default:
			return nil, err
		}
	}

	client := newClientFn(tokens)
	client.Refresher = r.refresh
	return client, nil
}

func loadTokens(store *secrets.Store, email string) (*garminauth.Tokens, error) {
	data, err := store.Get(email)
	if err != nil {
		if err == secrets.ErrNotFound {
			return nil, errNoCredentials(email)
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	tokens, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		return nil, fmt.Errorf("parse tokens: %w", err)
	}
	if tokens.Domain == "" {
		cfg, err := config.Read()
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
		tokens.Domain = cfg.Domain()
	}
	if tokens.Domain, err = garminauth.ParseDomain(tokens.Domain); err != nil {
		return nil, err
	}
	return tokens, nil
}

type tokenRefresher struct {
	store *secrets.Store
	email string
}

func (r *tokenRefresher) refresh(ctx context.Context, sent *garminauth.Tokens) (*garminauth.Tokens, error) {
	unlock, err := lockTokens()
	if err != nil {
		return nil, err
	}
	defer unlock()

	stored, err := loadTokens(r.store, r.email)
	if err != nil {
		return nil, err
	}
	if stored.OAuth2AccessToken != sent.OAuth2AccessToken && !stored.IsExpired() {
		return stored, nil
	}
	if !stored.CanRefresh() {
		return nil, &garminapi.TokenExpiredError{Email: r.email}
	}
	fresh, err := refreshFn(ctx, stored, garminauth.LoginOptions{Domain: stored.Domain})
	if err != nil {
		return nil, &garminapi.TokenExpiredError{Email: r.email}
	}
	if err := persistTokens(r.store, r.email, fresh); err != nil {
		return nil, fmt.Errorf("save refreshed token: %w", err)
	}
	return fresh, nil
}

func lockTokens() (unlock func(), err error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("lock token refresh: %w", err)
	}
	unlock, err = config.LockFile(filepath.Join(dir, "token.lock"))
	if err != nil {
		return nil, fmt.Errorf("lock token refresh: %w", err)
	}
	return unlock, nil
}

func persistTokens(store *secrets.Store, email string, tokens *garminauth.Tokens) error {
	data, err := tokens.Marshal()
	if err != nil {
		return err
	}
	return store.Set(email, data)
}

func resolveDisplayName(g *Globals, client *garminapi.Client) (string, error) {
	displayName, err := client.GetDisplayName(g.Context)
	if err != nil {
		return "", fmt.Errorf("resolve display name: %w", err)
	}
	return displayName, nil
}
