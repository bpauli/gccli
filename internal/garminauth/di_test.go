package garminauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRefreshOAuth2_DIRefresh(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /di-oauth2-service/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != basicAuth("test-di-client") {
			t.Errorf("Authorization = %q, want basic auth for DI client", r.Header.Get("Authorization"))
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", r.Form.Get("grant_type"))
		}
		if r.Form.Get("client_id") != "test-di-client" {
			t.Errorf("client_id = %q, want test-di-client", r.Form.Get("client_id"))
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Errorf("refresh_token = %q, want old-refresh", r.Form.Get("refresh_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":               "Bearer",
			"access_token":             "new-access",
			"refresh_token":            "new-refresh",
			"expires_in":               3600,
			"refresh_token_expires_in": 7776000,
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	origURL := diTokenURL
	diTokenURL = srv.URL + "/di-oauth2-service/oauth/token"
	t.Cleanup(func() { diTokenURL = origURL })

	tokens := &Tokens{
		OAuth2AccessToken:  "old-access",
		OAuth2RefreshToken: "old-refresh",
		DIClientID:         "test-di-client",
		Domain:             DomainGlobal,
		Email:              "test@example.com",
		DisplayName:        "Test User",
	}

	refreshed, err := RefreshOAuth2(context.Background(), tokens, LoginOptions{})
	if err != nil {
		t.Fatalf("RefreshOAuth2() error: %v", err)
	}

	if refreshed.OAuth2AccessToken != "new-access" {
		t.Errorf("OAuth2AccessToken = %q, want new-access", refreshed.OAuth2AccessToken)
	}
	if refreshed.OAuth2RefreshToken != "new-refresh" {
		t.Errorf("OAuth2RefreshToken = %q, want new-refresh", refreshed.OAuth2RefreshToken)
	}
	if refreshed.DIClientID != "test-di-client" {
		t.Errorf("DIClientID = %q, want test-di-client", refreshed.DIClientID)
	}
	if refreshed.Email != tokens.Email || refreshed.Domain != tokens.Domain || refreshed.DisplayName != tokens.DisplayName {
		t.Error("metadata was not preserved")
	}
	if !refreshed.OAuth2ExpiresAt.After(time.Now()) {
		t.Error("OAuth2ExpiresAt should be in the future")
	}
}

func TestRefreshOAuth2_DIRefreshPreservesRefreshTokenWhenOmitted(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /di-oauth2-service/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":   "Bearer",
			"access_token": "new-access",
			"expires_in":   3600,
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	origURL := diTokenURL
	diTokenURL = srv.URL + "/di-oauth2-service/oauth/token"
	t.Cleanup(func() { diTokenURL = origURL })

	tokens := &Tokens{
		OAuth2RefreshToken: "old-refresh",
		DIClientID:         "test-di-client",
	}

	refreshed, err := RefreshOAuth2(context.Background(), tokens, LoginOptions{})
	if err != nil {
		t.Fatalf("RefreshOAuth2() error: %v", err)
	}

	if refreshed.OAuth2RefreshToken != "old-refresh" {
		t.Errorf("OAuth2RefreshToken = %q, want old-refresh", refreshed.OAuth2RefreshToken)
	}
}

// TestRefreshDI_OnlyTriesStoredClientID verifies that when DIClientID is set,
// refresh attempts only that client_id and does not iterate fallbacks even
// when the server rejects it. Refresh tokens are bound to the issuing
// client_id, so iterating would burn requests for no possible benefit.
func TestRefreshDI_OnlyTriesStoredClientID(t *testing.T) {
	var calls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("POST /di-oauth2-service/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != basicAuth("test-di-client") {
			t.Errorf("Authorization = %q, want basic auth for test-di-client", r.Header.Get("Authorization"))
		}
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	origURL := diTokenURL
	diTokenURL = srv.URL + "/di-oauth2-service/oauth/token"
	t.Cleanup(func() { diTokenURL = origURL })

	tokens := &Tokens{
		OAuth2RefreshToken: "old-refresh",
		DIClientID:         "test-di-client",
	}

	_, err := RefreshOAuth2(context.Background(), tokens, LoginOptions{})
	if err == nil {
		t.Fatal("expected error from refresh")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server received %d calls, want exactly 1", got)
	}
}

// TestRefreshDI_IteratesWhenClientIDUnknown verifies that legacy imports
// without a stored DIClientID iterate the hardcoded list.
func TestRefreshDI_IteratesWhenClientIDUnknown(t *testing.T) {
	var calls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("POST /di-oauth2-service/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	origURL := diTokenURL
	diTokenURL = srv.URL + "/di-oauth2-service/oauth/token"
	t.Cleanup(func() { diTokenURL = origURL })

	tokens := &Tokens{OAuth2RefreshToken: "old-refresh"}

	_, err := RefreshOAuth2(context.Background(), tokens, LoginOptions{})
	if err == nil {
		t.Fatal("expected error from refresh")
	}
	if got, want := calls.Load(), int32(len(diClientIDs)); got != want {
		t.Errorf("server received %d calls, want %d (one per fallback client_id)", got, want)
	}
}

// TestExchangeServiceTicket_Success verifies the happy path uses the first
// (newest) client_id and populates token fields.
func TestExchangeServiceTicket_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /di-oauth2-service/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != basicAuth(diClientIDs[0]) {
			t.Errorf("Authorization for %q expected, got %q", diClientIDs[0], r.Header.Get("Authorization"))
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("client_id") != diClientIDs[0] {
			t.Errorf("client_id = %q, want %q", r.Form.Get("client_id"), diClientIDs[0])
		}
		if r.Form.Get("grant_type") != diGrantType {
			t.Errorf("grant_type = %q, want %q", r.Form.Get("grant_type"), diGrantType)
		}
		if r.Form.Get("service_ticket") != "ST-test-ticket" {
			t.Errorf("service_ticket = %q, want ST-test-ticket", r.Form.Get("service_ticket"))
		}
		if r.Form.Get("service_url") != "https://example.com/cb" {
			t.Errorf("service_url = %q, want https://example.com/cb", r.Form.Get("service_url"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":    "Bearer",
			"access_token":  "access-tok",
			"refresh_token": "refresh-tok",
			"expires_in":    3600,
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	origURL := diTokenURL
	diTokenURL = srv.URL + "/di-oauth2-service/oauth/token"
	t.Cleanup(func() { diTokenURL = origURL })

	tokens, err := exchangeServiceTicket(context.Background(), srv.Client(), "ST-test-ticket", "https://example.com/cb")
	if err != nil {
		t.Fatalf("exchangeServiceTicket() error: %v", err)
	}
	if tokens.OAuth2AccessToken != "access-tok" {
		t.Errorf("OAuth2AccessToken = %q, want access-tok", tokens.OAuth2AccessToken)
	}
	if tokens.OAuth2RefreshToken != "refresh-tok" {
		t.Errorf("OAuth2RefreshToken = %q, want refresh-tok", tokens.OAuth2RefreshToken)
	}
	if tokens.DIClientID != diClientIDs[0] {
		t.Errorf("DIClientID = %q, want %q", tokens.DIClientID, diClientIDs[0])
	}
}

// TestExchangeServiceTicket_DoesNotIterate verifies a single failed attempt
// does not retry against fallback client_ids — service tickets are single-use.
func TestExchangeServiceTicket_DoesNotIterate(t *testing.T) {
	var calls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("POST /di-oauth2-service/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "bad ticket", http.StatusBadRequest)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	origURL := diTokenURL
	diTokenURL = srv.URL + "/di-oauth2-service/oauth/token"
	t.Cleanup(func() { diTokenURL = origURL })

	_, err := exchangeServiceTicket(context.Background(), srv.Client(), "ST-bad", "https://example.com/cb")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server received %d calls, want exactly 1", got)
	}
}

// TestPostDIForm_RejectsInvalidExpiresIn verifies that a non-positive
// expires_in produces an error rather than a born-expired token.
func TestPostDIForm_RejectsInvalidExpiresIn(t *testing.T) {
	cases := []struct {
		name      string
		expiresIn int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("POST /di-oauth2-service/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"token_type":   "Bearer",
					"access_token": "a",
					"expires_in":   tc.expiresIn,
				})
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			origURL := diTokenURL
			diTokenURL = srv.URL + "/di-oauth2-service/oauth/token"
			t.Cleanup(func() { diTokenURL = origURL })

			_, err := exchangeServiceTicket(context.Background(), srv.Client(), "ST-ok", "https://example.com/cb")
			if err == nil {
				t.Fatal("expected error for non-positive expires_in")
			}
			if !strings.Contains(err.Error(), "invalid expires_in") {
				t.Errorf("error = %q, want to contain 'invalid expires_in'", err.Error())
			}
		})
	}
}

func TestExtractClientIDFromJWT(t *testing.T) {
	encode := func(payload string) string {
		// Header.payload.sig — we only decode the middle segment.
		body := base64.RawURLEncoding.EncodeToString([]byte(payload))
		return "header." + body + ".sig"
	}

	cases := []struct {
		name     string
		token    string
		fallback string
		want     string
	}{
		{
			name:     "valid client_id claim",
			token:    encode(`{"client_id":"FROM_JWT","sub":"u"}`),
			fallback: "FALLBACK",
			want:     "FROM_JWT",
		},
		{
			name:     "missing client_id claim",
			token:    encode(`{"sub":"u"}`),
			fallback: "FALLBACK",
			want:     "FALLBACK",
		},
		{
			name:     "empty client_id claim",
			token:    encode(`{"client_id":""}`),
			fallback: "FALLBACK",
			want:     "FALLBACK",
		},
		{
			name:     "malformed JSON payload",
			token:    encode(`not json`),
			fallback: "FALLBACK",
			want:     "FALLBACK",
		},
		{
			name:     "invalid base64 payload",
			token:    "header.!!!.sig",
			fallback: "FALLBACK",
			want:     "FALLBACK",
		},
		{
			name:     "fewer than 2 segments",
			token:    "single",
			fallback: "FALLBACK",
			want:     "FALLBACK",
		},
		{
			name:     "empty token",
			token:    "",
			fallback: "FALLBACK",
			want:     "FALLBACK",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractClientIDFromJWT(tc.token, tc.fallback)
			if got != tc.want {
				t.Errorf("extractClientIDFromJWT() = %q, want %q", got, tc.want)
			}
		})
	}
}
