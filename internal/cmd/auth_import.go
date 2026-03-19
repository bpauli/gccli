package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/garminauth"
)

// readStdinFn reads all data from stdin. Variable for testing.
var readStdinFn = func() ([]byte, error) { return io.ReadAll(os.Stdin) }

// AuthImportCmd imports credentials from a base64 string produced by auth export.
type AuthImportCmd struct {
	Token string `arg:"" optional:"" help:"Base64-encoded token string from 'auth export'. Reads from stdin if omitted."`
}

func (c *AuthImportCmd) Run(g *Globals) error {
	input := c.Token
	if input == "" {
		data, err := readStdinFn()
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		input = strings.TrimSpace(string(data))
	}

	if input == "" {
		return fmt.Errorf("no token data provided; pass as argument or pipe from stdin")
	}

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return fmt.Errorf("invalid base64 data: %w", err)
	}

	tokens, err := garminauth.UnmarshalTokens(decoded)
	if err != nil {
		return fmt.Errorf("invalid token data: %w", err)
	}

	if tokens.Email == "" {
		return fmt.Errorf("token data missing email address")
	}

	store, err := loadSecretsStore()
	if err != nil {
		return err
	}

	if err := store.Set(tokens.Email, decoded); err != nil {
		return fmt.Errorf("store tokens: %w", err)
	}

	// Save as default account.
	cfg, err := config.Read()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	cfg.DefaultAccount = tokens.Email
	if err := config.Write(cfg); err != nil {
		return fmt.Errorf("save default account: %w", err)
	}

	g.UI.Successf("Imported credentials for %s (domain: %s)", tokens.Email, tokens.Domain)
	if tokens.IsExpired() {
		g.UI.Warnf("Warning: imported token is expired; you may need to log in again")
	}
	return nil
}
