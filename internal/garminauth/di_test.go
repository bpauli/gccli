package garminauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
