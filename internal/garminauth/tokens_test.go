package garminauth

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTokens_IsExpired_ZeroTime(t *testing.T) {
	t.Parallel()

	tok := &Tokens{}
	if !tok.IsExpired() {
		t.Error("IsExpired() = false for zero time, want true")
	}
}

func TestTokens_IsExpired_PastExpiry(t *testing.T) {
	t.Parallel()

	tok := &Tokens{
		OAuth2ExpiresAt: time.Now().Add(-10 * time.Minute),
	}
	if !tok.IsExpired() {
		t.Error("IsExpired() = false for past expiry, want true")
	}
}

func TestTokens_IsExpired_WithinGracePeriod(t *testing.T) {
	t.Parallel()

	// Expires in 30 seconds -- within the 60-second grace period.
	tok := &Tokens{
		OAuth2ExpiresAt: time.Now().Add(30 * time.Second),
	}
	if !tok.IsExpired() {
		t.Error("IsExpired() = false for token expiring within grace period, want true")
	}
}

func TestTokens_IsExpired_FutureExpiry(t *testing.T) {
	t.Parallel()

	tok := &Tokens{
		OAuth2ExpiresAt: time.Now().Add(2 * time.Hour),
	}
	if tok.IsExpired() {
		t.Error("IsExpired() = true for future expiry, want false")
	}
}

func TestTokens_HasOAuth1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		token  string
		secret string
		want   bool
	}{
		{"both set", "tok", "sec", true},
		{"token empty", "", "sec", false},
		{"secret empty", "tok", "", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tok := &Tokens{OAuth1Token: tt.token, OAuth1Secret: tt.secret}
			if got := tok.HasOAuth1(); got != tt.want {
				t.Errorf("HasOAuth1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokens_MarshalRoundtrip(t *testing.T) {
	t.Parallel()

	expiry := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	original := &Tokens{
		OAuth1Token:        "oauth1-tok",
		OAuth1Secret:       "oauth1-sec",
		OAuth2AccessToken:  "access-tok",
		OAuth2RefreshToken: "refresh-tok",
		OAuth2ExpiresAt:    expiry,
		MFAToken:           "mfa-tok",
		Domain:             DomainGlobal,
		DisplayName:        "Test User",
		Email:              "test@example.com",
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	restored, err := UnmarshalTokens(data)
	if err != nil {
		t.Fatalf("UnmarshalTokens() error: %v", err)
	}

	if restored.OAuth1Token != original.OAuth1Token {
		t.Errorf("OAuth1Token = %q, want %q", restored.OAuth1Token, original.OAuth1Token)
	}
	if restored.OAuth1Secret != original.OAuth1Secret {
		t.Errorf("OAuth1Secret = %q, want %q", restored.OAuth1Secret, original.OAuth1Secret)
	}
	if restored.OAuth2AccessToken != original.OAuth2AccessToken {
		t.Errorf("OAuth2AccessToken = %q, want %q", restored.OAuth2AccessToken, original.OAuth2AccessToken)
	}
	if restored.OAuth2RefreshToken != original.OAuth2RefreshToken {
		t.Errorf("OAuth2RefreshToken = %q, want %q", restored.OAuth2RefreshToken, original.OAuth2RefreshToken)
	}
	if !restored.OAuth2ExpiresAt.Equal(original.OAuth2ExpiresAt) {
		t.Errorf("OAuth2ExpiresAt = %v, want %v", restored.OAuth2ExpiresAt, original.OAuth2ExpiresAt)
	}
	if restored.MFAToken != original.MFAToken {
		t.Errorf("MFAToken = %q, want %q", restored.MFAToken, original.MFAToken)
	}
	if restored.Domain != original.Domain {
		t.Errorf("Domain = %q, want %q", restored.Domain, original.Domain)
	}
	if restored.DisplayName != original.DisplayName {
		t.Errorf("DisplayName = %q, want %q", restored.DisplayName, original.DisplayName)
	}
	if restored.Email != original.Email {
		t.Errorf("Email = %q, want %q", restored.Email, original.Email)
	}
}

func TestTokens_MarshalJSON_OmitsEmptyMFA(t *testing.T) {
	t.Parallel()

	tok := &Tokens{
		OAuth1Token:        "tok",
		OAuth1Secret:       "sec",
		OAuth2AccessToken:  "access",
		OAuth2RefreshToken: "refresh",
		Domain:             DomainGlobal,
		Email:              "test@example.com",
	}

	data, err := tok.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal raw: %v", err)
	}

	if _, ok := raw["mfa_token"]; ok {
		t.Error("mfa_token should be omitted when empty")
	}
	if _, ok := raw["display_name"]; ok {
		t.Error("display_name should be omitted when empty")
	}
}

func TestUnmarshalTokens_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := UnmarshalTokens([]byte("not json"))
	if err == nil {
		t.Error("UnmarshalTokens(invalid) should return error")
	}
}
