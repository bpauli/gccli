package garminauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLoginBrowser_Success(t *testing.T) {
	mux := ssoMux(t)
	srv, ep := mockSSO(t, mux)

	origURL := oauthConsumerURL
	oauthConsumerURL = srv.URL + "/oauth_consumer.json"
	t.Cleanup(func() { oauthConsumerURL = origURL })

	origOpen := openBrowserFn
	openBrowserFn = func(ssoURL string) error {
		// Simulate SSO redirect by calling the callback URL with a ticket.
		parsed, err := url.Parse(ssoURL)
		if err != nil {
			return err
		}
		callbackURL := parsed.Query().Get("service")
		go func() {
			resp, err := http.Get(callbackURL + "?ticket=ST-mock-ticket-456")
			if err != nil {
				t.Errorf("callback request failed: %v", err)
				return
			}
			_ = resp.Body.Close()
		}()
		return nil
	}
	t.Cleanup(func() { openBrowserFn = origOpen })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokens, err := loginBrowser(ctx, "test@example.com", LoginOptions{}, ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokens.Email != "test@example.com" {
		t.Errorf("email = %q, want %q", tokens.Email, "test@example.com")
	}
	if tokens.Domain != DomainGlobal {
		t.Errorf("domain = %q, want %q", tokens.Domain, DomainGlobal)
	}
	if tokens.OAuth1Token != "mock-oauth1-token" {
		t.Errorf("oauth1_token = %q, want %q", tokens.OAuth1Token, "mock-oauth1-token")
	}
	if tokens.OAuth1Secret != "mock-oauth1-secret" {
		t.Errorf("oauth1_secret = %q, want %q", tokens.OAuth1Secret, "mock-oauth1-secret")
	}
	if tokens.OAuth2AccessToken != "mock-access-token" {
		t.Errorf("access_token = %q, want %q", tokens.OAuth2AccessToken, "mock-access-token")
	}
	if tokens.OAuth2RefreshToken != "mock-refresh-token" {
		t.Errorf("refresh_token = %q, want %q", tokens.OAuth2RefreshToken, "mock-refresh-token")
	}
	if tokens.OAuth2ExpiresAt.IsZero() {
		t.Error("expires_at should not be zero")
	}
	if tokens.IsExpired() {
		t.Error("token should not be expired")
	}
}

func TestLoginBrowser_Timeout(t *testing.T) {
	_, ep := mockSSO(t, http.NewServeMux())

	origOpen := openBrowserFn
	openBrowserFn = func(_ string) error {
		// Don't call callback — simulate user not completing login.
		return nil
	}
	t.Cleanup(func() { openBrowserFn = origOpen })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := loginBrowser(ctx, "test@example.com", LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error for timeout")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "timed out")
	}
}

func TestLoginBrowser_BrowserOpenError(t *testing.T) {
	_, ep := mockSSO(t, http.NewServeMux())

	origOpen := openBrowserFn
	openBrowserFn = func(_ string) error {
		return fmt.Errorf("no browser available")
	}
	t.Cleanup(func() { openBrowserFn = origOpen })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := loginBrowser(ctx, "test@example.com", LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error for browser open failure")
	}
	if !strings.Contains(err.Error(), "open browser") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "open browser")
	}
}

func TestLoginBrowser_ContextCancelled(t *testing.T) {
	_, ep := mockSSO(t, http.NewServeMux())

	origOpen := openBrowserFn
	openBrowserFn = func(_ string) error {
		return nil
	}
	t.Cleanup(func() { openBrowserFn = origOpen })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := loginBrowser(ctx, "test@example.com", LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestStartCallbackServer(t *testing.T) {
	callbackURL, _, cleanup, err := startCallbackServer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if !strings.HasPrefix(callbackURL, "http://127.0.0.1:") {
		t.Errorf("callbackURL = %q, want prefix %q", callbackURL, "http://127.0.0.1:")
	}

	// Verify server is listening.
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("failed to reach callback server: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestCallbackServer_WithTicket(t *testing.T) {
	callbackURL, ticketCh, cleanup, err := startCallbackServer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	resp, err := http.Get(callbackURL + "?ticket=ST-test-ticket")
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case result := <-ticketCh:
		if result.err != nil {
			t.Fatalf("unexpected error: %v", result.err)
		}
		if result.ticket != "ST-test-ticket" {
			t.Errorf("ticket = %q, want %q", result.ticket, "ST-test-ticket")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for ticket")
	}
}

func TestCallbackServer_NoTicket(t *testing.T) {
	callbackURL, ticketCh, cleanup, err := startCallbackServer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	// Request without ticket should not send a result.
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case result := <-ticketCh:
		t.Fatalf("unexpected result: %+v", result)
	case <-time.After(100 * time.Millisecond):
		// Expected: no result for request without ticket.
	}
}

func TestBuildSSOURL(t *testing.T) {
	ep := Endpoints{
		SSOBase:   "https://sso.garmin.com",
		SSOSignin: "https://sso.garmin.com/sso/signin",
	}
	callbackURL := "http://127.0.0.1:12345"

	ssoURL := buildSSOURL(ep, callbackURL, "user@example.com")

	parsed, err := url.Parse(ssoURL)
	if err != nil {
		t.Fatalf("failed to parse SSO URL: %v", err)
	}

	if !strings.HasPrefix(ssoURL, ep.SSOSignin) {
		t.Errorf("SSO URL should start with %q, got %q", ep.SSOSignin, ssoURL)
	}

	params := parsed.Query()
	if params.Get("service") != callbackURL {
		t.Errorf("service = %q, want %q", params.Get("service"), callbackURL)
	}
	if params.Get("source") != callbackURL {
		t.Errorf("source = %q, want %q", params.Get("source"), callbackURL)
	}
	if params.Get("gauthHost") != ep.SSOBase+"/sso" {
		t.Errorf("gauthHost = %q, want %q", params.Get("gauthHost"), ep.SSOBase+"/sso")
	}
	if params.Get("redirectAfterAccountLoginUrl") != callbackURL {
		t.Errorf("redirectAfterAccountLoginUrl = %q, want %q",
			params.Get("redirectAfterAccountLoginUrl"), callbackURL)
	}
	if params.Get("redirectAfterAccountCreationUrl") != callbackURL {
		t.Errorf("redirectAfterAccountCreationUrl = %q, want %q",
			params.Get("redirectAfterAccountCreationUrl"), callbackURL)
	}
	if params.Get("prepopUsername") != "user@example.com" {
		t.Errorf("prepopUsername = %q, want %q",
			params.Get("prepopUsername"), "user@example.com")
	}
}

func TestBuildSSOURL_NoEmail(t *testing.T) {
	ep := Endpoints{
		SSOBase:   "https://sso.garmin.com",
		SSOSignin: "https://sso.garmin.com/sso/signin",
	}
	callbackURL := "http://127.0.0.1:12345"

	ssoURL := buildSSOURL(ep, callbackURL, "")

	parsed, err := url.Parse(ssoURL)
	if err != nil {
		t.Fatalf("failed to parse SSO URL: %v", err)
	}
	if parsed.Query().Get("prepopUsername") != "" {
		t.Errorf("prepopUsername should be absent, got %q", parsed.Query().Get("prepopUsername"))
	}
}

func TestBuildSSOURL_ChinaDomain(t *testing.T) {
	ep := Endpoints{
		SSOBase:   "https://sso.garmin.cn",
		SSOSignin: "https://sso.garmin.cn/sso/signin",
	}
	callbackURL := "http://127.0.0.1:54321"

	ssoURL := buildSSOURL(ep, callbackURL, "")

	if !strings.HasPrefix(ssoURL, ep.SSOSignin) {
		t.Errorf("SSO URL should start with %q, got %q", ep.SSOSignin, ssoURL)
	}

	parsed, _ := url.Parse(ssoURL)
	if parsed.Query().Get("gauthHost") != "https://sso.garmin.cn/sso" {
		t.Errorf("gauthHost = %q, want %q",
			parsed.Query().Get("gauthHost"), "https://sso.garmin.cn/sso")
	}
}
