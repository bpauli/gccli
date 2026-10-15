package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
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

func TestConnectAPI_Unauthorized_RefresherReturnsStoredToken(t *testing.T) {
	var auth []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.Header.Get("Authorization"))
		if len(auth) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	_, client := testServer(t, handler)

	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		t.Error("refreshTokensFn called, want the stored token without a refresh")
		return nil, errors.New("unexpected")
	}

	var sent string
	client.Refresher = func(_ context.Context, tokens *garminauth.Tokens) (*garminauth.Tokens, error) {
		sent = tokens.OAuth2AccessToken
		stored := *tokens
		stored.OAuth2AccessToken = "stored-newer"
		return &stored, nil
	}

	if _, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil); err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}
	if sent != "access-token" {
		t.Errorf("Refresher got %q, want the rejected token access-token", sent)
	}
	want := []string{"Bearer access-token", "Bearer stored-newer"}
	if !reflect.DeepEqual(auth, want) {
		t.Errorf("Authorization = %q, want %q", auth, want)
	}
	if client.Tokens().OAuth2AccessToken != "stored-newer" {
		t.Errorf("client token = %q, want stored-newer", client.Tokens().OAuth2AccessToken)
	}
}

func TestConnectAPI_Unauthorized_RefresherErrorPassesThrough(t *testing.T) {
	calls := 0
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	lockErr := errors.New("lock token refresh: permission denied")
	client.Refresher = func(context.Context, *garminauth.Tokens) (*garminauth.Tokens, error) {
		return nil, lockErr
	}

	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if !errors.Is(err, lockErr) {
		t.Fatalf("err = %v, want %v", err, lockErr)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestConnectAPI_Unauthorized_RefreshFails(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, client := testServer(t, handler)

	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	cause := errors.New("refresh failed")
	refreshTokensFn = func(_ context.Context, _ *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, cause
	}

	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var tokenErr *TokenExpiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("expected TokenExpiredError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), cause.Error()) {
		t.Errorf("err = %q, want it to keep the cause %q", err, cause)
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

func TestConnectAPI_Unauthorized_NoRefreshCredentials(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	client.tokens.OAuth1Token = ""
	client.tokens.OAuth1Secret = ""
	client.tokens.OAuth2RefreshToken = ""
	client.Refresher = func(context.Context, *garminauth.Tokens) (*garminauth.Tokens, error) {
		t.Error("Refresher called without refresh credentials")
		return nil, errors.New("unexpected")
	}

	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	var tokenErr *TokenExpiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("err = %T %v, want TokenExpiredError", err, err)
	}
}

func TestRefreshInMemory(t *testing.T) {
	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(_ context.Context, tokens *garminauth.Tokens, opts garminauth.LoginOptions) (*garminauth.Tokens, error) {
		if opts.Domain != garminauth.DomainGlobal {
			t.Errorf("domain = %q, want %q", opts.Domain, garminauth.DomainGlobal)
		}
		newTokens := *tokens
		newTokens.OAuth2AccessToken = "new-access-token"
		return &newTokens, nil
	}

	got, err := refreshInMemory(context.Background(), testTokens())
	if err != nil {
		t.Fatalf("refreshInMemory: %v", err)
	}
	if got.OAuth2AccessToken != "new-access-token" {
		t.Errorf("access token = %q, want new-access-token", got.OAuth2AccessToken)
	}
}

func TestRefreshInMemory_Failure(t *testing.T) {
	origRefresh := refreshTokensFn
	t.Cleanup(func() { refreshTokensFn = origRefresh })
	refreshTokensFn = func(context.Context, *garminauth.Tokens, garminauth.LoginOptions) (*garminauth.Tokens, error) {
		return nil, errors.New("exchange failed")
	}

	_, err := refreshInMemory(context.Background(), testTokens())
	var tokenErr *TokenExpiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("err = %T %v, want TokenExpiredError", err, err)
	}
}

type requestKind struct {
	name        string
	method      string
	path        string
	body        string
	accept      string
	contentType string
	call        func(c *Client) ([]byte, error)
	emptyIsNil  bool
}

func requestKinds(t *testing.T) []requestKind {
	t.Helper()
	fitPath := t.TempDir() + "/run.fit"
	if err := os.WriteFile(fitPath, []byte("FIT-DATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	return []requestKind{
		{
			name: "json get", method: http.MethodGet, path: "/api/get", accept: "application/json", emptyIsNil: true,
			call: func(c *Client) ([]byte, error) { return c.ConnectAPI(ctx, http.MethodGet, "/api/get", nil) },
		},
		{
			name: "json put", method: http.MethodPut, path: "/api/put", body: `{"a":1}`, accept: "application/json", contentType: "application/json", emptyIsNil: true,
			call: func(c *Client) ([]byte, error) {
				return c.ConnectAPI(ctx, http.MethodPut, "/api/put", strings.NewReader(`{"a":1}`))
			},
		},
		{
			name: "download", method: http.MethodGet, path: "/download/1",
			call: func(c *Client) ([]byte, error) { return c.Download(ctx, "/download/1") },
		},
		{
			name: "upload", method: http.MethodPost, path: "/upload-service/upload", body: "FIT-DATA", contentType: "multipart/form-data; boundary=", emptyIsNil: true,
			call: func(c *Client) ([]byte, error) { return c.UploadActivity(ctx, fitPath) },
		},
	}
}

type answer struct {
	status     int
	body       string
	retryAfter string
}

func TestRequestKinds(t *testing.T) {
	ok := answer{status: http.StatusOK, body: `{"ok":true}`}
	unauthorized := answer{status: http.StatusUnauthorized, body: "expired"}
	scenarios := []struct {
		name      string
		answers   []answer
		refreshes int
		wantBody  string
		wantErr   error
	}{
		{name: "200", answers: []answer{ok}, wantBody: ok.body},
		{name: "204", answers: []answer{{status: http.StatusNoContent}}},
		{name: "empty 200", answers: []answer{{status: http.StatusOK}}},
		{name: "401 then 200", answers: []answer{unauthorized, ok}, refreshes: 1, wantBody: ok.body},
		{name: "401 then 401", answers: []answer{unauthorized, unauthorized}, refreshes: 1, wantErr: &TokenExpiredError{Email: "test@example.com"}},
		{name: "429", answers: []answer{{status: http.StatusTooManyRequests, body: "slow down", retryAfter: "120"}}, wantErr: &RateLimitError{RetryAfter: "120"}},
		{name: "500", answers: []answer{{status: http.StatusInternalServerError, body: "boom"}}, wantErr: &GarminAPIError{StatusCode: 500, Message: "boom"}},
	}

	for _, kind := range requestKinds(t) {
		for _, sc := range scenarios {
			t.Run(kind.name+"/"+sc.name, func(t *testing.T) {
				var got []*http.Request
				var bodies []string
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					b, _ := io.ReadAll(r.Body)
					got = append(got, r)
					bodies = append(bodies, string(b))
					if len(got) > len(sc.answers) {
						w.WriteHeader(http.StatusTeapot)
						return
					}
					a := sc.answers[len(got)-1]
					if a.retryAfter != "" {
						w.Header().Set("Retry-After", a.retryAfter)
					}
					w.WriteHeader(a.status)
					_, _ = io.WriteString(w, a.body)
				})
				_, client := testServer(t, handler)

				refreshes := 0
				origRefresh := refreshTokensFn
				t.Cleanup(func() { refreshTokensFn = origRefresh })
				refreshTokensFn = func(_ context.Context, tokens *garminauth.Tokens, _ garminauth.LoginOptions) (*garminauth.Tokens, error) {
					refreshes++
					next := *tokens
					next.OAuth2AccessToken = "refreshed-token"
					return &next, nil
				}

				data, err := kind.call(client)

				if sc.wantErr != nil {
					if err == nil || err.Error() != sc.wantErr.Error() || reflect.TypeOf(err) != reflect.TypeOf(sc.wantErr) {
						t.Fatalf("err = %T %v, want %T %v", err, err, sc.wantErr, sc.wantErr)
					}
					if data != nil {
						t.Errorf("data = %q, want nil", data)
					}
				} else {
					if err != nil {
						t.Fatalf("err = %v", err)
					}
					if string(data) != sc.wantBody {
						t.Errorf("data = %q, want %q", data, sc.wantBody)
					}
					if sc.wantBody == "" && kind.emptyIsNil != (data == nil) {
						t.Errorf("data nil = %v, want %v", data == nil, kind.emptyIsNil)
					}
				}
				if refreshes != sc.refreshes {
					t.Errorf("refreshes = %d, want %d", refreshes, sc.refreshes)
				}
				if len(got) != len(sc.answers) {
					t.Fatalf("requests = %d, want %d", len(got), len(sc.answers))
				}
				for i, r := range got {
					token := "access-token"
					if i > 0 {
						token = "refreshed-token"
					}
					checkRequest(t, i, r, bodies[i], kind, token)
				}
			})
		}
	}
}

func checkRequest(t *testing.T, i int, r *http.Request, body string, kind requestKind, token string) {
	t.Helper()
	if r.Method != kind.method || r.URL.Path != kind.path {
		t.Errorf("request %d = %s %s, want %s %s", i, r.Method, r.URL.Path, kind.method, kind.path)
	}
	want := map[string]string{
		"Authorization": "Bearer " + token,
		"User-Agent":    "com.garmin.android.apps.connectmobile",
		"Accept":        kind.accept,
	}
	for name, v := range want {
		if got := r.Header.Get(name); got != v {
			t.Errorf("request %d %s = %q, want %q", i, name, got, v)
		}
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, kind.contentType) || (kind.contentType == "") != (ct == "") {
		t.Errorf("request %d Content-Type = %q, want prefix %q", i, ct, kind.contentType)
	}
	if !strings.Contains(body, kind.body) || (kind.body == "") != (body == "") {
		t.Errorf("request %d body = %q, want it to hold %q", i, body, kind.body)
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("cut") }
func (failingBody) Close() error             { return nil }

func TestRequestKinds_ErrorText(t *testing.T) {
	sendFails := roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("down") })
	readFails := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: failingBody{}}, nil
	})
	want := map[string][2]string{
		"json get": {`request GET /api/get: Get "https://x.test/api/get": down`, "read response: cut"},
		"json put": {`request PUT /api/put: Put "https://x.test/api/put": down`, "read response: cut"},
		"download": {`download /download/1: Get "https://x.test/download/1": down`, "read download response: cut"},
		"upload":   {`upload /upload-service/upload: Post "https://x.test/upload-service/upload": down`, "read upload response: cut"},
	}
	for _, kind := range requestKinds(t) {
		for i, rt := range []http.RoundTripper{sendFails, readFails} {
			client := NewClient(testTokens(), WithHTTPClient(&http.Client{Transport: rt}), WithBaseURL("https://x.test"))
			_, err := kind.call(client)
			if err == nil || err.Error() != want[kind.name][i] {
				t.Errorf("%s: err = %v, want %s", kind.name, err, want[kind.name][i])
			}
		}
	}
}
