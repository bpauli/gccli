package harness

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
)

func DecodeToken(raw string) (*garminauth.Tokens, error) {
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("GARMIN_TOKEN is not base64: %w", err)
	}
	tokens, err := garminauth.UnmarshalTokens(data)
	if err != nil {
		return nil, fmt.Errorf("GARMIN_TOKEN is not a gccli token: %w", err)
	}
	if tokens.OAuth2AccessToken == "" {
		return nil, errors.New("GARMIN_TOKEN has no OAuth2 access token")
	}
	return tokens, nil
}

func LoadToken(ctx context.Context, raw string, base http.RoundTripper) (*garminauth.Tokens, error) {
	tokens, err := DecodeToken(raw)
	if err != nil {
		return nil, err
	}
	if !tokens.IsExpired() {
		return tokens, nil
	}
	refreshed, err := refresh(ctx, tokens, base)
	if err != nil {
		return nil, fmt.Errorf("GARMIN_TOKEN refresh: %w", err)
	}
	return refreshed, nil
}

var refreshTimeout = 60 * time.Second

func refresh(ctx context.Context, tokens *garminauth.Tokens, base http.RoundTripper) (*garminauth.Tokens, error) {
	return garminauth.RefreshOAuth2(ctx, tokens, garminauth.LoginOptions{
		Domain:     tokens.Domain,
		HTTPClient: &http.Client{Transport: base, Timeout: refreshTimeout},
	})
}
