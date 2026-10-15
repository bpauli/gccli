package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/secrets"
)

// AuthExportCmd exports stored credentials as a portable base64 string.
type AuthExportCmd struct{}

func (c *AuthExportCmd) Run(g *Globals) error {
	email := g.Account
	if email == "" {
		return errNoAccount()
	}

	store, err := loadSecretsStore()
	if err != nil {
		return err
	}

	data, err := store.Get(email)
	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			return errNoCredentials(email)
		}
		return fmt.Errorf("read credentials: %w", err)
	}

	// Validate that the data is valid tokens before exporting.
	if _, err := garminauth.UnmarshalTokens(data); err != nil {
		return fmt.Errorf("invalid token data: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	_, _ = fmt.Fprintln(g.Stdout, encoded)
	return nil
}
