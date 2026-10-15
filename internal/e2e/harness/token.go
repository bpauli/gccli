package harness

import (
	"encoding/base64"
	"errors"
	"fmt"

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
