package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/bpauli/gccli/internal/garminauth"
	"github.com/bpauli/gccli/internal/secrets"
)

// AuthExportCmd exports stored credentials as a portable base64 string.
type AuthExportCmd struct{}

func (c *AuthExportCmd) Run(g *Globals) error {
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

	// Validate that the data is valid tokens before exporting.
	if _, err := garminauth.UnmarshalTokens(data); err != nil {
		return fmt.Errorf("invalid token data: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	_, _ = fmt.Fprintln(os.Stdout, encoded)
	return nil
}
