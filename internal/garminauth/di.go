package garminauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultDITokenURL = "https://diauth.garmin.com/di-oauth2-service/oauth/token"
	diGrantType       = "https://connectapi.garmin.com/di-oauth2-service/oauth/grant/service_ticket"

	nativeAPIUserAgent = "GCM-Android-5.23"
	nativeGarminUA     = "com.garmin.android.apps.connectmobile/5.23; ; Google/sdk_gphone64_arm64/google; Android/33; Dalvik/2.1.0"
)

var diClientIDs = []string{
	"GARMIN_CONNECT_MOBILE_ANDROID_DI_2025Q2",
	"GARMIN_CONNECT_MOBILE_ANDROID_DI_2024Q4",
	"GARMIN_CONNECT_MOBILE_ANDROID_DI",
	"GARMIN_CONNECT_MOBILE_IOS_DI",
}

var diTokenURL = defaultDITokenURL

type diTokenResponse struct {
	TokenType             string `json:"token_type"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
}

// exchangeServiceTicket exchanges a Garmin CAS service ticket for DI OAuth
// Bearer tokens. serviceURL must match the service URL used to issue ticket.
//
// Service tickets are single-use, so we don't iterate over fallback client_ids
// — a failed attempt would burn the ticket. The newest client_id is used; if
// Garmin rotates the constant, the diClientIDs list must be updated.
func exchangeServiceTicket(ctx context.Context, client *http.Client, ticket, serviceURL string) (*Tokens, error) {
	clientID := diClientIDs[0]
	values := url.Values{
		"client_id":      {clientID},
		"service_ticket": {ticket},
		"grant_type":     {diGrantType},
		"service_url":    {serviceURL},
	}

	resp, err := postDIForm(ctx, client, clientID, values)
	if err != nil {
		return nil, fmt.Errorf("DI token exchange (%s): %w", clientID, err)
	}

	tokens := tokensFromDIResponse(resp, clientID)
	tokens.DIClientID = extractClientIDFromJWT(tokens.OAuth2AccessToken, clientID)
	return tokens, nil
}

// refreshDI refreshes DI OAuth Bearer tokens using a stored refresh token.
// Refresh tokens are bound to the client_id that issued them, so when we know
// the issuing client_id we only attempt that one; iterating fallbacks is
// reserved for legacy imports where DIClientID is unset.
func refreshDI(ctx context.Context, client *http.Client, tokens *Tokens) (*Tokens, error) {
	if tokens.OAuth2RefreshToken == "" {
		return nil, fmt.Errorf("no DI refresh token available")
	}

	var clientIDs []string
	if tokens.DIClientID != "" {
		clientIDs = []string{tokens.DIClientID}
	} else {
		clientIDs = diClientIDs
	}

	var failures []string
	for _, clientID := range clientIDs {
		values := url.Values{
			"grant_type":    {"refresh_token"},
			"client_id":     {clientID},
			"refresh_token": {tokens.OAuth2RefreshToken},
		}

		resp, err := postDIForm(ctx, client, clientID, values)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", clientID, err))
			continue
		}

		refreshed := tokensFromDIResponse(resp, clientID)
		refreshed.Domain = tokens.Domain
		refreshed.Email = tokens.Email
		refreshed.DisplayName = tokens.DisplayName
		refreshed.MFAToken = tokens.MFAToken
		refreshed.OAuth1Token = tokens.OAuth1Token
		refreshed.OAuth1Secret = tokens.OAuth1Secret
		refreshed.DIClientID = extractClientIDFromJWT(refreshed.OAuth2AccessToken, clientID)
		if refreshed.OAuth2RefreshToken == "" {
			refreshed.OAuth2RefreshToken = tokens.OAuth2RefreshToken
		}
		return refreshed, nil
	}

	return nil, fmt.Errorf("DI token refresh failed: %s", strings.Join(failures, "; "))
}

func postDIForm(ctx context.Context, client *http.Client, clientID string, values url.Values) (*diTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, diTokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	for key, value := range nativeHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Authorization", basicAuth(clientID))
	req.Header.Set("Accept", "application/json,text/html;q=0.9,*/*;q=0.8")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var parsed diTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("response missing access_token")
	}
	if parsed.ExpiresIn <= 0 {
		return nil, fmt.Errorf("response has invalid expires_in: %d", parsed.ExpiresIn)
	}
	return &parsed, nil
}

func tokensFromDIResponse(resp *diTokenResponse, clientID string) *Tokens {
	return &Tokens{
		OAuth2AccessToken:  resp.AccessToken,
		OAuth2RefreshToken: resp.RefreshToken,
		OAuth2ExpiresAt:    time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
		DIClientID:         clientID,
	}
}

func nativeHeaders() map[string]string {
	return map[string]string{
		"User-Agent":                  nativeAPIUserAgent,
		"X-Garmin-User-Agent":         nativeGarminUA,
		"X-Garmin-Paired-App-Version": "10861",
		"X-Garmin-Client-Platform":    "Android",
		"X-App-Ver":                   "10861",
		"X-Lang":                      "en",
		"X-GCExperience":              "GC5",
		"Accept-Language":             "en-US,en;q=0.9",
	}
}

func basicAuth(clientID string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"))
}

// extractClientIDFromJWT decodes the JWT payload to read the client_id claim.
// The signature is NOT verified — the value is informational only and is used
// solely as a hint for which client_id to send on subsequent refresh.
func extractClientIDFromJWT(token, fallback string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return fallback
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fallback
	}

	var claims struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.ClientID == "" {
		return fallback
	}
	return claims.ClientID
}
