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

// resolveClient loads stored tokens for the account and returns an API client.
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

	return newClientFn(tokens), nil
}
