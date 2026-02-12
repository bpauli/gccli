package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminauth"
)

// testTokens returns tokens suitable for testing.
func testTokens() *garminauth.Tokens {
	return &garminauth.Tokens{
		OAuth1Token:        "oauth1-token",
		OAuth1Secret:       "oauth1-secret",
		OAuth2AccessToken:  "access-token",
		OAuth2RefreshToken: "refresh-token",
		OAuth2ExpiresAt:    time.Now().Add(time.Hour),
		Domain:             garminauth.DomainGlobal,
		Email:              "test@example.com",
	}
}

// testServer creates a test HTTP server and returns it along with a configured client.
func testServer(t *testing.T, handler http.Handler) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	tokens := testTokens()
	client := NewClient(tokens,
		WithHTTPClient(srv.Client()),
		WithBaseURL(srv.URL),
	)
	return srv, client
}

// --- NewClient tests ---

func TestNewClient(t *testing.T) {
	tokens := testTokens()
	c := NewClient(tokens)

	if c.tokens != tokens {
		t.Error("tokens not set")
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if c.baseURL == "" {
		t.Error("baseURL is empty")
	}
	if c.baseURL != "https://connectapi.garmin.com" {
		t.Errorf("baseURL = %q, want https://connectapi.garmin.com", c.baseURL)
	}
}

func TestNewClient_ChinaDomain(t *testing.T) {
	tokens := testTokens()
	tokens.Domain = garminauth.DomainChina
	c := NewClient(tokens)

	if c.baseURL != "https://connectapi.garmin.cn" {
		t.Errorf("baseURL = %q, want https://connectapi.garmin.cn", c.baseURL)
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	tokens := testTokens()
	custom := &http.Client{}
	c := NewClient(tokens,
		WithHTTPClient(custom),
		WithBaseURL("http://localhost:8080"),
	)

	if c.httpClient != custom {
		t.Error("custom HTTP client not set")
	}
	if c.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL = %q, want http://localhost:8080", c.baseURL)
	}
}

func TestClient_Tokens(t *testing.T) {
	tokens := testTokens()
	c := NewClient(tokens)

	if c.Tokens() != tokens {
		t.Error("Tokens() did not return the expected tokens")
	}
}

// --- ConnectAPI tests ---

func TestConnectAPI_GET_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/test/path" {
			t.Errorf("path = %s, want /test/path", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Errorf("Authorization = %q, want Bearer access-token", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":123,"name":"test"}`))
	})

	_, client := testServer(t, handler)
	data, err := client.ConnectAPI(context.Background(), http.MethodGet, "/test/path", nil)
	if err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("name = %v, want test", result["name"])
	}
}

func TestConnectAPI_POST_WithBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"key":"value"}` {
			t.Errorf("body = %s, want {\"key\":\"value\"}", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, client := testServer(t, handler)
	body := strings.NewReader(`{"key":"value"}`)
	data, err := client.ConnectAPI(context.Background(), http.MethodPost, "/api/endpoint", body)
	if err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["ok"] != true {
		t.Errorf("ok = %v, want true", result["ok"])
	}
}

func TestConnectAPI_NoContent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := testServer(t, handler)
	data, err := client.ConnectAPI(context.Background(), http.MethodDelete, "/api/resource/1", nil)
	if err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}
	if data != nil {
		t.Errorf("data = %s, want nil", data)
	}
}

func TestConnectAPI_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	})

	_, client := testServer(t, handler)
	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if apiErr.Message != "internal error" {
		t.Errorf("Message = %q, want internal error", apiErr.Message)
	}
}

func TestConnectAPI_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	_, client := testServer(t, handler)
	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/missing", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestConnectAPI_RateLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, client := testServer(t, handler)
	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != "60" {
		t.Errorf("RetryAfter = %q, want 60", rlErr.RetryAfter)
	}
}

func TestConnectAPI_Unauthorized_NoOAuth1(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, client := testServer(t, handler)
	// Remove OAuth1 credentials so refresh cannot be attempted.
	client.tokens.OAuth1Token = ""
	client.tokens.OAuth1Secret = ""

	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var tokenErr *TokenExpiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("expected TokenExpiredError, got %T: %v", err, err)
	}
	if tokenErr.Email != "test@example.com" {
		t.Errorf("Email = %q, want test@example.com", tokenErr.Email)
	}
}

func TestConnectAPI_Unauthorized_RefreshSuccess(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			// First call: 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Second call after refresh: verify new token
		if got := r.Header.Get("Authorization"); got != "Bearer refreshed-token" {
			t.Errorf("Authorization after refresh = %q, want Bearer refreshed-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refreshed":true}`))
	})

	_, client := testServer(t, handler)

	// Override refresh function.
	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		newTokens := *tokens
		newTokens.OAuth2AccessToken = "refreshed-token"
		newTokens.OAuth2ExpiresAt = time.Now().Add(time.Hour)
		return &newTokens, nil
	}

	data, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["refreshed"] != true {
		t.Errorf("refreshed = %v, want true", result["refreshed"])
	}

	// Verify client tokens were updated.
	if client.Tokens().OAuth2AccessToken != "refreshed-token" {
		t.Errorf("token not updated after refresh")
	}
}

func TestConnectAPI_Unauthorized_RefreshFails(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, client := testServer(t, handler)

	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, errors.New("refresh failed")
	}

	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var tokenErr *TokenExpiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("expected TokenExpiredError, got %T: %v", err, err)
	}
}

func TestConnectAPI_POST_Retry_PreservesBody(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"data":"preserved"}` {
			t.Errorf("body on retry = %s, want {\"data\":\"preserved\"}", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	_, client := testServer(t, handler)

	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		newTokens := *tokens
		newTokens.OAuth2AccessToken = "new-token"
		return &newTokens, nil
	}

	body := strings.NewReader(`{"data":"preserved"}`)
	_, err := client.ConnectAPI(context.Background(), http.MethodPost, "/api/endpoint", body)
	if err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestConnectAPI_ContextCancelled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})

	_, client := testServer(t, handler)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.ConnectAPI(ctx, http.MethodGet, "/api/path", nil)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}

// --- Download tests ---

func TestDownload_Success(t *testing.T) {
	fileData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Errorf("Authorization = %q, want Bearer access-token", got)
		}
		// Download should NOT set Accept: application/json.
		if got := r.Header.Get("Accept"); got != "" {
			t.Errorf("Accept = %q, want empty for downloads", got)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(fileData)
	})

	_, client := testServer(t, handler)
	data, err := client.Download(context.Background(), "/download/file.fit")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}

	if len(data) != len(fileData) {
		t.Fatalf("len(data) = %d, want %d", len(data), len(fileData))
	}
	for i, b := range data {
		if b != fileData[i] {
			t.Errorf("data[%d] = %x, want %x", i, b, fileData[i])
		}
	}
}

func TestDownload_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})

	_, client := testServer(t, handler)
	_, err := client.Download(context.Background(), "/download/file.fit")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected GarminAPIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestDownload_RateLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, client := testServer(t, handler)
	_, err := client.Download(context.Background(), "/download/file.fit")
	if err == nil {
		t.Fatal("expected error")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rlErr.RetryAfter != "30" {
		t.Errorf("RetryAfter = %q, want 30", rlErr.RetryAfter)
	}
}

func TestDownload_Unauthorized_RefreshSuccess(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer refreshed-token" {
			t.Errorf("Authorization after refresh = %q, want Bearer refreshed-token", got)
		}
		_, _ = w.Write([]byte("file-data"))
	})

	_, client := testServer(t, handler)

	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		newTokens := *tokens
		newTokens.OAuth2AccessToken = "refreshed-token"
		return &newTokens, nil
	}

	data, err := client.Download(context.Background(), "/download/file.fit")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if string(data) != "file-data" {
		t.Errorf("data = %q, want file-data", data)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestDownload_Unauthorized_NoOAuth1(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, client := testServer(t, handler)
	client.tokens.OAuth1Token = ""
	client.tokens.OAuth1Secret = ""

	_, err := client.Download(context.Background(), "/download/file.fit")
	if err == nil {
		t.Fatal("expected error")
	}

	var tokenErr *TokenExpiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("expected TokenExpiredError, got %T: %v", err, err)
	}
}

// --- Error type tests ---

func TestAuthRequiredError(t *testing.T) {
	err := &AuthRequiredError{Email: "user@example.com"}
	if got := err.Error(); got != "no auth found for user@example.com" {
		t.Errorf("Error() = %q", got)
	}

	err2 := &AuthRequiredError{}
	if got := err2.Error(); got != "no auth found" {
		t.Errorf("Error() = %q", got)
	}
}

func TestTokenExpiredError(t *testing.T) {
	err := &TokenExpiredError{Email: "user@example.com"}
	if got := err.Error(); got != "token expired for user@example.com" {
		t.Errorf("Error() = %q", got)
	}

	err2 := &TokenExpiredError{}
	if got := err2.Error(); got != "token expired" {
		t.Errorf("Error() = %q", got)
	}
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{RetryAfter: "120"}
	if got := err.Error(); got != "rate limited; retry after 120" {
		t.Errorf("Error() = %q", got)
	}

	err2 := &RateLimitError{}
	if got := err2.Error(); got != "rate limited" {
		t.Errorf("Error() = %q", got)
	}
}

func TestGarminAPIError(t *testing.T) {
	tests := []struct {
		name   string
		err    *GarminAPIError
		expect string
	}{
		{
			name:   "known status with message",
			err:    &GarminAPIError{StatusCode: 403, Message: "forbidden resource"},
			expect: "garmin api: 403 Forbidden: forbidden resource",
		},
		{
			name:   "known status without message",
			err:    &GarminAPIError{StatusCode: 500},
			expect: "garmin api: 500 Internal Server Error",
		},
		{
			name:   "unknown status",
			err:    &GarminAPIError{StatusCode: 599, Message: "custom"},
			expect: "garmin api: 599: custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expect {
				t.Errorf("Error() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestInvalidFileFormatError(t *testing.T) {
	err := &InvalidFileFormatError{Format: "xyz"}
	if got := err.Error(); got != "invalid file format: xyz" {
		t.Errorf("Error() = %q", got)
	}
}

// --- RefreshToken tests ---

func TestRefreshToken_NoOAuth1(t *testing.T) {
	tokens := testTokens()
	tokens.OAuth1Token = ""
	tokens.OAuth1Secret = ""
	client := NewClient(tokens)

	err := client.refreshToken(context.Background())
	if err == nil {
		t.Fatal("expected error when no OAuth1 credentials")
	}
	if !strings.Contains(err.Error(), "no OAuth1 credentials") {
		t.Errorf("error = %q, want to contain 'no OAuth1 credentials'", err.Error())
	}
}

func TestRefreshToken_Success(t *testing.T) {
	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, tokens *garminauth.Tokens, opts garminauth.LoginOptions) (*garminauth.Tokens, error) {
		if opts.Domain != garminauth.DomainGlobal {
			t.Errorf("domain = %q, want %q", opts.Domain, garminauth.DomainGlobal)
		}
		newTokens := *tokens
		newTokens.OAuth2AccessToken = "new-access-token"
		newTokens.OAuth2ExpiresAt = time.Now().Add(time.Hour)
		return &newTokens, nil
	}

	tokens := testTokens()
	client := NewClient(tokens)

	err := client.refreshToken(context.Background())
	if err != nil {
		t.Fatalf("refreshToken: %v", err)
	}
	if client.tokens.OAuth2AccessToken != "new-access-token" {
		t.Errorf("access token = %q, want new-access-token", client.tokens.OAuth2AccessToken)
	}
}

func TestRefreshToken_Failure(t *testing.T) {
	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, errors.New("exchange failed")
	}

	tokens := testTokens()
	client := NewClient(tokens)

	err := client.refreshToken(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "exchange failed") {
		t.Errorf("error = %q, want to contain 'exchange failed'", err.Error())
	}
}
