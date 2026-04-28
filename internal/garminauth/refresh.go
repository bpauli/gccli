package garminauth

import (
	"context"
	"fmt"
	"net/http"
)

// RefreshOAuth2 refreshes stored Garmin API credentials.
// DI OAuth refresh tokens are preferred; legacy OAuth1 refresh remains for
// older exported credentials.
//
// When DI credentials are present the refresh is terminal — failures do not
// fall through to the OAuth1 path, since the DI refresh_token has been
// consumed or revoked and OAuth1 fallback would yield stale state.
func RefreshOAuth2(ctx context.Context, tokens *Tokens, opts LoginOptions) (*Tokens, error) {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	if tokens.DIClientID != "" && tokens.OAuth2RefreshToken != "" {
		return refreshDI(ctx, client, tokens)
	}

	if tokens.HasOAuth1() {
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

	if tokens.OAuth2RefreshToken != "" {
		return refreshDI(ctx, client, tokens)
	}

	return nil, fmt.Errorf("no refresh credentials available")
}
