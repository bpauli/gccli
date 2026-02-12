package garminauth

import (
	"encoding/json"
	"time"
)

// Tokens holds the OAuth1 and OAuth2 credentials obtained from Garmin SSO.
// Tokens are serialized to JSON for storage in the OS keyring.
type Tokens struct {
	// OAuth1 credentials (used to refresh OAuth2 tokens).
	OAuth1Token  string `json:"oauth1_token"`
	OAuth1Secret string `json:"oauth1_secret"`

	// OAuth2 credentials (used for API requests).
	OAuth2AccessToken  string    `json:"oauth2_access_token"`
	OAuth2RefreshToken string    `json:"oauth2_refresh_token"`
	OAuth2ExpiresAt    time.Time `json:"oauth2_expires_at"`

	// MFA token returned during MFA-enabled login.
	MFAToken string `json:"mfa_token,omitempty"`

	// Domain is the Garmin domain used for authentication (e.g. "garmin.com").
	Domain string `json:"domain"`

	// DisplayName is the user's display name from Garmin Connect.
	DisplayName string `json:"display_name,omitempty"`

	// Email is the account email address.
	Email string `json:"email"`
}

// IsExpired reports whether the OAuth2 access token has expired.
// A token is considered expired if it expires within the next 60 seconds.
func (t *Tokens) IsExpired() bool {
	if t.OAuth2ExpiresAt.IsZero() {
		return true
	}
	return time.Now().After(t.OAuth2ExpiresAt.Add(-60 * time.Second))
}

// HasOAuth1 reports whether the tokens contain OAuth1 credentials
// that can be used to refresh the OAuth2 token.
func (t *Tokens) HasOAuth1() bool {
	return t.OAuth1Token != "" && t.OAuth1Secret != ""
}

// Marshal serializes the tokens to JSON for keyring storage.
func (t *Tokens) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

// UnmarshalTokens deserializes tokens from JSON keyring data.
func UnmarshalTokens(data []byte) (*Tokens, error) {
	var t Tokens
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
