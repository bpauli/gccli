package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/secrets"
)

// loadSecretsStore opens the OS keyring using the configured backend.
// Variable to allow overriding in tests.
var loadSecretsStore = defaultLoadSecretsStore

func defaultLoadSecretsStore() (*secrets.Store, error) {
	cfg, err := config.Read()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return secrets.Open(cfg.KeyringBackendValue())
}

// AuthCmd groups authentication subcommands.
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Log in to Garmin Connect."`
	Status AuthStatusCmd `cmd:"" help:"Show authentication status."`
	Remove AuthRemoveCmd `cmd:"" help:"Remove stored credentials."`
	Token  AuthTokenCmd  `cmd:"" help:"Print current access token."`
}

// AuthStatusCmd shows the current authentication state.
type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(g *Globals) error {
	email := g.Account
	if email == "" {
		return fmt.Errorf("no account specified; use --account, set GCCLI_ACCOUNT, or run gccli auth login")
	}

	store, err := loadSecretsStore()
	if err != nil {
		return err
	}

	data, err := store.Get(email)
	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			g.UI.Warnf("No credentials stored for %s", email)
			return nil
		}
		return fmt.Errorf("read credentials: %w", err)
	}

	tokens, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		return fmt.Errorf("parse tokens: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		status := map[string]any{
			"email":   tokens.Email,
			"domain":  tokens.Domain,
			"expired": tokens.IsExpired(),
		}
		if !tokens.OAuth2ExpiresAt.IsZero() {
			status["expires_at"] = tokens.OAuth2ExpiresAt.Format(time.RFC3339)
		}
		if tokens.DisplayName != "" {
			status["display_name"] = tokens.DisplayName
		}
		return outfmt.WriteJSON(os.Stdout, status)
	}

	g.UI.Successf("Authenticated as %s", email)
	if tokens.DisplayName != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Display name: %s\n", tokens.DisplayName)
	}
	_, _ = fmt.Fprintf(os.Stdout, "Domain:       %s\n", tokens.Domain)
	if tokens.IsExpired() {
		g.UI.Warnf("Token expired (at %s)", tokens.OAuth2ExpiresAt.Format(time.RFC3339))
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Expires at:   %s\n", tokens.OAuth2ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

// AuthRemoveCmd removes stored credentials.
type AuthRemoveCmd struct{}

func (c *AuthRemoveCmd) Run(g *Globals) error {
	email := g.Account
	if email == "" {
		return fmt.Errorf("no account specified; use --account, set GCCLI_ACCOUNT, or run gccli auth login")
	}

	store, err := loadSecretsStore()
	if err != nil {
		return err
	}

	if err := store.Delete(email); err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			g.UI.Warnf("No credentials stored for %s", email)
			return nil
		}
		return fmt.Errorf("remove credentials: %w", err)
	}

	g.UI.Successf("Removed credentials for %s", email)
	return nil
}

// AuthTokenCmd prints the current OAuth2 access token.
type AuthTokenCmd struct{}

func (c *AuthTokenCmd) Run(g *Globals) error {
	email := g.Account
	if email == "" {
		return fmt.Errorf("no account specified; use --account, set GCCLI_ACCOUNT, or run gccli auth login")
	}

	store, err := loadSecretsStore()
	if err != nil {
		return err
	}

	data, err := store.Get(email)
	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			return fmt.Errorf("no credentials stored for %s", email)
		}
		return fmt.Errorf("read credentials: %w", err)
	}

	tokens, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		return fmt.Errorf("parse tokens: %w", err)
	}

	if tokens.IsExpired() {
		g.UI.Warnf("Warning: token is expired")
	}

	_, _ = fmt.Fprintln(os.Stdout, tokens.OAuth2AccessToken)
	return nil
}
