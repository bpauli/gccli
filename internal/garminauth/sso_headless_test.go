package garminauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// mockSSO creates a test server that simulates the Garmin SSO flow.
// It returns the server and Endpoints pointing to it.
func mockSSO(t *testing.T, handler http.Handler) (*httptest.Server, Endpoints) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, Endpoints{
		SSOBase:      srv.URL,
		SSOEmbed:     srv.URL + "/sso/embed",
		SSOSignin:    srv.URL + "/sso/signin",
		SSOVerifyMFA: srv.URL + "/sso/verifyMFA/loginEnterMfaCode",
		OAuthBase:    srv.URL,
		ConnectAPI:   srv.URL,
	}
}

// ssoMux returns an http.ServeMux that simulates the full Garmin SSO flow.
func ssoMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()

	// SSO embed (cookie setup).
	mux.HandleFunc("GET /sso/embed", func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "GARMIN-SSO-GUID", Value: "test-guid"})
		w.WriteHeader(http.StatusOK)
	})

	// SSO signin GET (return HTML with CSRF token).
	mux.HandleFunc("GET /sso/signin", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>GARMIN > Sign In</title></head>
<body><form><input type="hidden" name="_csrf" value="mock-csrf-token-123"></form></body></html>`))
	})

	// SSO signin POST (validate credentials, return ticket).
	mux.HandleFunc("POST /sso/signin", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("_csrf") != "mock-csrf-token-123" {
			http.Error(w, "bad csrf", http.StatusForbidden)
			return
		}
		if r.Form.Get("username") != "test@example.com" || r.Form.Get("password") != "correctpass" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><title>GARMIN > Error</title></head><body>Invalid credentials</body></html>`))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Success</title></head>
<body><script>window.location.replace("https://sso.garmin.com/sso/embed?ticket=ST-mock-ticket-456")</script></body></html>`))
	})

	// OAuth consumer credentials.
	mux.HandleFunc("GET /oauth_consumer.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oauthConsumer{
			ConsumerKey:    "test-consumer-key",
			ConsumerSecret: "test-consumer-secret",
		})
	})

	// OAuth1 preauthorized exchange.
	mux.HandleFunc("GET /oauth-service/oauth/preauthorized", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "OAuth ") {
			http.Error(w, "missing OAuth header", http.StatusUnauthorized)
			return
		}
		ticket := r.URL.Query().Get("ticket")
		if ticket == "" {
			http.Error(w, "missing ticket", http.StatusBadRequest)
			return
		}
		loginURL := r.URL.Query().Get("login-url")
		if loginURL == "" {
			http.Error(w, "missing login-url", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = w.Write([]byte("oauth_token=mock-oauth1-token&oauth_token_secret=mock-oauth1-secret"))
	})

	// OAuth2 token exchange.
	mux.HandleFunc("POST /oauth-service/oauth/exchange/user/2.0", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "OAuth ") {
			http.Error(w, "missing OAuth header", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":               "Bearer",
			"access_token":             "mock-access-token",
			"refresh_token":            "mock-refresh-token",
			"expires_in":               3600,
			"refresh_token_expires_in": 7776000,
		})
	})

	return mux
}

func TestLoginHeadless_Success(t *testing.T) {
	mux := ssoMux(t)
	srv, ep := mockSSO(t, mux)

	origURL := oauthConsumerURL
	oauthConsumerURL = srv.URL + "/oauth_consumer.json"
	t.Cleanup(func() { oauthConsumerURL = origURL })

	tokens, err := loginHeadless(context.Background(), "test@example.com", "correctpass", LoginOptions{}, ep)
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

func TestLoginHeadless_BadCredentials(t *testing.T) {
	mux := ssoMux(t)
	srv, ep := mockSSO(t, mux)

	origURL := oauthConsumerURL
	oauthConsumerURL = srv.URL + "/oauth_consumer.json"
	t.Cleanup(func() { oauthConsumerURL = origURL })

	_, err := loginHeadless(context.Background(), "test@example.com", "wrongpass", LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "authentication failed")
	}
}

func TestLoginHeadless_MFARequired_NoCodeOrPrompt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sso/embed", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /sso/signin", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><input name="_csrf" value="csrf123"></body></html>`))
	})
	mux.HandleFunc("POST /sso/signin", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>GARMIN > MFA Challenge</title></head>
<body><form><input type="hidden" name="_csrf" value="mfa-csrf-123"><input name="mfa-code"></form></body></html>`))
	})

	_, ep := mockSSO(t, mux)

	_, err := loginHeadless(context.Background(), "test@example.com", "pass", LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error for MFA required without code or prompt")
	}
	if !strings.Contains(err.Error(), "MFA required") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "MFA required")
	}
}

func TestLoginHeadless_AccountLocked(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sso/embed", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /sso/signin", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><input name="_csrf" value="csrf123"></body></html>`))
	})
	mux.HandleFunc("POST /sso/signin", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Account Locked</title></head><body></body></html>`))
	})

	_, ep := mockSSO(t, mux)

	_, err := loginHeadless(context.Background(), "test@example.com", "pass", LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error for locked account")
	}
	if !strings.Contains(err.Error(), "account locked") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "account locked")
	}
}

func TestLoginHeadless_CustomHTTPClient(t *testing.T) {
	mux := ssoMux(t)
	srv, ep := mockSSO(t, mux)

	origURL := oauthConsumerURL
	oauthConsumerURL = srv.URL + "/oauth_consumer.json"
	t.Cleanup(func() { oauthConsumerURL = origURL })

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	customClient := &http.Client{Jar: jar}

	tokens, err := loginHeadless(context.Background(), "test@example.com", "correctpass",
		LoginOptions{HTTPClient: customClient}, ep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.OAuth2AccessToken != "mock-access-token" {
		t.Errorf("access_token = %q, want %q", tokens.OAuth2AccessToken, "mock-access-token")
	}
}

func TestGetCSRFToken(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		want    string
		wantErr bool
	}{
		{
			name: "standard input",
			html: `<html><body><form><input type="hidden" name="_csrf" value="abc123"></form></body></html>`,
			want: "abc123",
		},
		{
			name: "input with other attributes",
			html: `<input id="x" name="_csrf" class="y" value="token-xyz">`,
			want: "token-xyz",
		},
		{
			name:    "no csrf input",
			html:    `<html><body><input name="username" value="foo"></body></html>`,
			wantErr: true,
		},
		{
			name:    "empty html",
			html:    "",
			wantErr: true,
		},
		{
			name: "nested form",
			html: `<div><form action="/login"><input name="_csrf" value="deep-token"><input name="user"></form></div>`,
			want: "deep-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCSRFToken(tt.html)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTitle(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "standard title",
			html: `<html><head><title>Success</title></head></html>`,
			want: "Success",
		},
		{
			name: "garmin title",
			html: `<html><head><title>GARMIN > Sign In</title></head></html>`,
			want: "GARMIN > Sign In",
		},
		{
			name: "mfa title",
			html: `<html><head><title>GARMIN > MFA Challenge</title></head></html>`,
			want: "GARMIN > MFA Challenge",
		},
		{
			name: "no title",
			html: `<html><body>no title here</body></html>`,
			want: "",
		},
		{
			name: "empty title",
			html: `<html><head><title></title></head></html>`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getTitle(tt.html)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTicket(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		want    string
		wantErr bool
	}{
		{
			name: "standard ticket",
			html: `<script>window.location.replace("https://sso.garmin.com/sso/embed?ticket=ST-12345-abc")</script>`,
			want: "ST-12345-abc",
		},
		{
			name: "ticket in full page",
			html: `<html><head><title>Success</title></head><body>
<script>window.location.replace("https://sso.garmin.com/sso/embed?ticket=ST-ticket-xyz")</script></body></html>`,
			want: "ST-ticket-xyz",
		},
		{
			name:    "no ticket",
			html:    `<html><head><title>Error</title></head><body>Login failed</body></html>`,
			wantErr: true,
		},
		{
			name:    "empty body",
			html:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getTicket(tt.html)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchOAuthConsumer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"consumer_key":"ck","consumer_secret":"cs"}`))
		}))
		t.Cleanup(srv.Close)

		origURL := oauthConsumerURL
		oauthConsumerURL = srv.URL
		t.Cleanup(func() { oauthConsumerURL = origURL })

		consumer, err := fetchOAuthConsumer(context.Background(), srv.Client())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if consumer.ConsumerKey != "ck" || consumer.ConsumerSecret != "cs" {
			t.Errorf("got key=%q secret=%q, want ck/cs", consumer.ConsumerKey, consumer.ConsumerSecret)
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)

		origURL := oauthConsumerURL
		oauthConsumerURL = srv.URL
		t.Cleanup(func() { oauthConsumerURL = origURL })

		_, err := fetchOAuthConsumer(context.Background(), srv.Client())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		t.Cleanup(srv.Close)

		origURL := oauthConsumerURL
		oauthConsumerURL = srv.URL
		t.Cleanup(func() { oauthConsumerURL = origURL })

		_, err := fetchOAuthConsumer(context.Background(), srv.Client())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestExchangePreauthorized(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth-service/oauth/preauthorized" {
				http.Error(w, "wrong path", http.StatusNotFound)
				return
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "OAuth ") {
				http.Error(w, "no auth", http.StatusUnauthorized)
				return
			}
			if r.Header.Get("User-Agent") != UserAgent {
				http.Error(w, "wrong user-agent", http.StatusBadRequest)
				return
			}
			loginURL := r.URL.Query().Get("login-url")
			if loginURL == "" {
				http.Error(w, "missing login-url", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte("oauth_token=t1&oauth_token_secret=s1&mfa_token=mfa1"))
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{
			SSOBase:   srv.URL,
			SSOEmbed:  srv.URL + "/sso/embed",
			OAuthBase: srv.URL,
		}
		consumer := &oauthConsumer{ConsumerKey: "ck", ConsumerSecret: "cs"}

		token, secret, mfa, err := exchangePreauthorized(
			context.Background(), srv.Client(), ep, consumer, "test-ticket", ep.SSOEmbed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "t1" {
			t.Errorf("token = %q, want %q", token, "t1")
		}
		if secret != "s1" {
			t.Errorf("secret = %q, want %q", secret, "s1")
		}
		if mfa != "mfa1" {
			t.Errorf("mfa = %q, want %q", mfa, "mfa1")
		}
	})

	t.Run("no mfa token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("oauth_token=t1&oauth_token_secret=s1"))
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{SSOEmbed: srv.URL + "/sso/embed", OAuthBase: srv.URL}
		consumer := &oauthConsumer{ConsumerKey: "ck", ConsumerSecret: "cs"}

		token, secret, mfa, err := exchangePreauthorized(
			context.Background(), srv.Client(), ep, consumer, "ticket", ep.SSOEmbed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "t1" || secret != "s1" {
			t.Errorf("got token=%q secret=%q", token, secret)
		}
		if mfa != "" {
			t.Errorf("mfa = %q, want empty", mfa)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("oauth_token=&oauth_token_secret="))
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{SSOEmbed: srv.URL + "/sso/embed", OAuthBase: srv.URL}
		consumer := &oauthConsumer{ConsumerKey: "ck", ConsumerSecret: "cs"}

		_, _, _, err := exchangePreauthorized(
			context.Background(), srv.Client(), ep, consumer, "ticket", ep.SSOEmbed)
		if err == nil {
			t.Fatal("expected error for missing token")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{SSOEmbed: srv.URL + "/sso/embed", OAuthBase: srv.URL}
		consumer := &oauthConsumer{ConsumerKey: "ck", ConsumerSecret: "cs"}

		_, _, _, err := exchangePreauthorized(
			context.Background(), srv.Client(), ep, consumer, "ticket", ep.SSOEmbed)
		if err == nil {
			t.Fatal("expected error for server error")
		}
	})
}

func TestExchangeOAuth2(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "want POST", http.StatusMethodNotAllowed)
				return
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "OAuth ") {
				http.Error(w, "no auth", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(oauth2Response{
				TokenType:             "Bearer",
				AccessToken:           "at-123",
				RefreshToken:          "rt-456",
				ExpiresIn:             3600,
				RefreshTokenExpiresIn: 7776000,
			})
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{OAuthBase: srv.URL}
		consumer := &oauthConsumer{ConsumerKey: "ck", ConsumerSecret: "cs"}

		tokens, err := exchangeOAuth2(
			context.Background(), srv.Client(), ep, consumer,
			"oauth1-token", "oauth1-secret", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokens.OAuth2AccessToken != "at-123" {
			t.Errorf("access_token = %q, want %q", tokens.OAuth2AccessToken, "at-123")
		}
		if tokens.OAuth2RefreshToken != "rt-456" {
			t.Errorf("refresh_token = %q, want %q", tokens.OAuth2RefreshToken, "rt-456")
		}
		if tokens.OAuth2ExpiresAt.IsZero() {
			t.Error("expires_at should not be zero")
		}
	})

	t.Run("with mfa token", func(t *testing.T) {
		var gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err == nil {
				gotBody = r.Form.Encode()
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(oauth2Response{
				AccessToken:  "at",
				RefreshToken: "rt",
				ExpiresIn:    3600,
			})
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{OAuthBase: srv.URL}
		consumer := &oauthConsumer{ConsumerKey: "ck", ConsumerSecret: "cs"}

		_, err := exchangeOAuth2(
			context.Background(), srv.Client(), ep, consumer,
			"t", "s", "mfa-token-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parsed, _ := url.ParseQuery(gotBody)
		if parsed.Get("mfa_token") != "mfa-token-123" {
			t.Errorf("mfa_token = %q, want %q", parsed.Get("mfa_token"), "mfa-token-123")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		}))
		t.Cleanup(srv.Close)

		ep := Endpoints{OAuthBase: srv.URL}
		consumer := &oauthConsumer{ConsumerKey: "ck", ConsumerSecret: "cs"}

		_, err := exchangeOAuth2(
			context.Background(), srv.Client(), ep, consumer,
			"t", "s", "")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSignOAuth1_Format(t *testing.T) {
	header := signOAuth1(
		http.MethodGet,
		"https://example.com/oauth?ticket=abc&login-url=https://sso.example.com",
		"consumer-key", "consumer-secret",
		"", "",
		nil,
	)

	if !strings.HasPrefix(header, "OAuth ") {
		t.Errorf("header should start with 'OAuth ', got: %s", header)
	}

	// Verify required OAuth parameters are present.
	required := []string{
		"oauth_consumer_key=",
		"oauth_nonce=",
		"oauth_signature_method=",
		"oauth_timestamp=",
		"oauth_version=",
		"oauth_signature=",
	}
	for _, param := range required {
		if !strings.Contains(header, param) {
			t.Errorf("header missing %q: %s", param, header)
		}
	}

	// oauth_token should not be present when token is empty.
	if strings.Contains(header, "oauth_token=") {
		t.Error("oauth_token should not be present when token is empty")
	}
}

func TestSignOAuth1_WithToken(t *testing.T) {
	header := signOAuth1(
		http.MethodPost,
		"https://example.com/exchange",
		"ck", "cs",
		"resource-token", "resource-secret",
		url.Values{"mfa_token": {"mfa123"}},
	)

	if !strings.Contains(header, "oauth_token=") {
		t.Error("header should contain oauth_token when token is provided")
	}
	if !strings.Contains(header, `oauth_consumer_key="ck"`) {
		t.Error("header should contain consumer key")
	}
}

func TestPercentEncode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abc", "abc"},
		{"hello world", "hello%20world"},
		{"a+b", "a%2Bb"},
		{"~tilde", "~tilde"},
		{"100%", "100%25"},
		{"https://example.com/path", "https%3A%2F%2Fexample.com%2Fpath"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := percentEncode(tt.input)
			if got != tt.want {
				t.Errorf("percentEncode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoginHeadless_ContextCancelled(t *testing.T) {
	mux := ssoMux(t)
	_, ep := mockSSO(t, mux)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := loginHeadless(ctx, "test@example.com", "correctpass", LoginOptions{}, ep)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestEndpointsDomain(t *testing.T) {
	tests := []struct {
		ssoBase string
		want    string
	}{
		{"https://sso.garmin.com", DomainGlobal},
		{"https://sso.garmin.cn", DomainChina},
		{"https://localhost:8080", DomainGlobal},
	}
	for _, tt := range tests {
		ep := Endpoints{SSOBase: tt.ssoBase}
		if got := ep.domain(); got != tt.want {
			t.Errorf("Endpoints{SSOBase: %q}.domain() = %q, want %q", tt.ssoBase, got, tt.want)
		}
	}
}
