package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/secrets"
	"github.com/bpauli/gccli/internal/ui"
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

	r := &tokenRefresher{store: store, email: email, ui: g.UI}
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
	if tokens.Domain, err = tokenDomain(email, tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func tokenDomain(email string, tokens *garminauth.Tokens) (string, error) {
	if tokens.Domain == "" {
		return loginDomain()
	}
	if _, err := garminauth.ParseDomain(tokens.Domain); err != nil {
		return "", fmt.Errorf("stored token for %s has unknown Garmin domain %q; run: gccli auth login %s", email, tokens.Domain, email)
	}
	return tokens.Domain, nil
}

type tokenRefresher struct {
	store *secrets.Store
	email string
	ui    *ui.UI
}

func (r *tokenRefresher) refresh(ctx context.Context, sent *garminauth.Tokens) (*garminauth.Tokens, error) {
	unlock, err := lockTokens(r.ui)
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
		return nil, &garminapi.TokenExpiredError{Email: r.email, Err: err}
	}
	if err := persistTokens(r.store, r.email, fresh); err != nil {
		r.ui.Warnf("Could not save the refreshed token: %v", err)
	}
	return fresh, nil
}

func lockTokens(u *ui.UI) (unlock func(), err error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(dir, 0o700); err == nil {
		unlock, err = config.LockFile(filepath.Join(dir, "token.lock"), func() {
			u.Infof("Waiting for another gccli process to refresh the token")
		})
	}
	switch {
	case err == nil:
		return unlock, nil
	case errors.Is(err, fs.ErrPermission), errors.Is(err, syscall.EROFS):
		return refreshWithoutLock, nil
	}
	return nil, fmt.Errorf("lock token refresh: %w", err)
}

func refreshWithoutLock() {}

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
