package garminauth

import (
	"context"
	"fmt"
	"net/http"
)

// RefreshOAuth2 exchanges existing OAuth1 credentials for new OAuth2 tokens.
// The provided tokens must have valid OAuth1 credentials (HasOAuth1() == true).
func RefreshOAuth2(ctx context.Context, tokens *Tokens, opts LoginOptions) (*Tokens, error) {
	if !tokens.HasOAuth1() {
		return nil, fmt.Errorf("no OAuth1 credentials available for refresh")
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	ep := NewEndpoints(opts.domain())

	consumer, err := fetchOAuthConsumer(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("fetch oauth consumer: %w", err)
	}

	newTokens, err := exchangeOAuth2(ctx, client, ep, consumer,
		tokens.OAuth1Token, tokens.OAuth1Secret, tokens.MFAToken)
	if err != nil {
		return nil, fmt.Errorf("oauth2 exchange: %w", err)
	}

	// Preserve existing metadata.
	newTokens.Domain = tokens.Domain
	newTokens.Email = tokens.Email
	newTokens.OAuth1Token = tokens.OAuth1Token
	newTokens.OAuth1Secret = tokens.OAuth1Secret
	newTokens.MFAToken = tokens.MFAToken
	newTokens.DisplayName = tokens.DisplayName

	return newTokens, nil
}
