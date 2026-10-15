package harness

import (
	"context"
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/garminauth"
)

func TestDecodeToken(t *testing.T) {
	want := &garminauth.Tokens{
		OAuth2AccessToken:  "AT",
		OAuth2RefreshToken: "RT",
		OAuth2ExpiresAt:    time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		DIClientID:         "DI",
		Domain:             garminauth.DomainGlobal,
		Email:              "e@example.com",
	}
	data, err := want.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeToken(base64.StdEncoding.EncodeToString(data))
	if err != nil {
		t.Fatalf("DecodeToken: %v", err)
	}
	if *got != *want {
		t.Errorf("DecodeToken = %+v, want %+v", got, want)
	}
}

func TestDecodeTokenRejects(t *testing.T) {
	for name, raw := range map[string]string{
		"not base64":      "notbase64",
		"not json":        base64.StdEncoding.EncodeToString([]byte("not json")),
		"no access token": base64.StdEncoding.EncodeToString([]byte(`{"oauth2_refresh_token":"RT"}`)),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeToken(raw)
			if err == nil || !strings.HasPrefix(err.Error(), "GARMIN_TOKEN ") {
				t.Errorf("err = %v, want an error that names GARMIN_TOKEN", err)
			}
		})
	}
}

func encodeToken(t *testing.T, tokens *garminauth.Tokens) string {
	t.Helper()
	data, err := tokens.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func expiredToken() *garminauth.Tokens {
	return &garminauth.Tokens{
		OAuth2AccessToken:  "AT-old",
		OAuth2RefreshToken: "RT-secret-old",
		OAuth2ExpiresAt:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		DIClientID:         "DI",
		Domain:             garminauth.DomainGlobal,
		Email:              "e@example.com",
	}
}

func TestLoadTokenRefreshesExpiredToken(t *testing.T) {
	var requests []string
	di := roundTripFunc(func(req *http.Request) *http.Response {
		if err := req.ParseForm(); err != nil {
			t.Error(err)
		}
		requests = append(requests, req.Method+" "+req.URL.Host+req.URL.Path+" "+req.PostForm.Get("refresh_token"))
		return page(req, http.StatusOK, `{"access_token":"AT-new","refresh_token":"RT-new","expires_in":3600}`)
	})
	got, err := LoadToken(context.Background(), encodeToken(t, expiredToken()), di)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	want := []string{"POST diauth.garmin.com/di-oauth2-service/oauth/token RT-secret-old"}
	if !slices.Equal(requests, want) {
		t.Errorf("requests = %q, want %q", requests, want)
	}
	if got.OAuth2AccessToken != "AT-new" || got.OAuth2RefreshToken != "RT-new" || got.Email != "e@example.com" || got.IsExpired() {
		t.Errorf("LoadToken = %+v, want the refreshed AT-new and RT-new for e@example.com", got)
	}
}

func TestLoadTokenKeepsFreshToken(t *testing.T) {
	fresh := expiredToken()
	fresh.OAuth2ExpiresAt = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	none := roundTripFunc(func(req *http.Request) *http.Response {
		t.Errorf("unexpected request %s %s", req.Method, req.URL)
		return page(req, http.StatusInternalServerError, "")
	})
	got, err := LoadToken(context.Background(), encodeToken(t, fresh), none)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if *got != *fresh {
		t.Errorf("LoadToken = %+v, want %+v", got, fresh)
	}
}

func TestLoadTokenNamesFailedRefresh(t *testing.T) {
	rejected := roundTripFunc(func(req *http.Request) *http.Response {
		return page(req, http.StatusBadRequest, `{"error":"invalid_grant"}`)
	})
	_, err := LoadToken(context.Background(), encodeToken(t, expiredToken()), rejected)
	if err == nil || !strings.HasPrefix(err.Error(), "GARMIN_TOKEN refresh: ") || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("err = %v, want a GARMIN_TOKEN refresh error with the invalid_grant answer", err)
	}
	if strings.Contains(err.Error(), "RT-secret") {
		t.Errorf("err = %v, want no refresh token in the message", err)
	}
}

func TestLoadTokenStopsHangingRefresh(t *testing.T) {
	old := refreshTimeout
	refreshTimeout = 50 * time.Millisecond
	t.Cleanup(func() { refreshTimeout = old })
	var hanging hangingTransport
	start := time.Now()
	_, err := LoadToken(context.Background(), encodeToken(t, expiredToken()), hanging)
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("err = %v, want a client timeout", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("LoadToken took %v, want it to stop at the refresh timeout", elapsed)
	}
}

type hangingTransport struct{}

func (hangingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-time.After(2 * time.Second):
		return page(req, http.StatusOK, `{"access_token":"AT-new","refresh_token":"RT-new","expires_in":3600}`), nil
	}
}

func rejectedToken() *garminauth.Tokens {
	tokens := expiredToken()
	tokens.OAuth2ExpiresAt = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	return tokens
}

func sharedGarmin(t *testing.T, diStatus int, diBody string) (transport http.RoundTripper, refreshes *[]string) {
	t.Helper()
	refreshes = new([]string)
	transport = roundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Host == "diauth.garmin.com" {
			if err := req.ParseForm(); err != nil {
				t.Error(err)
			}
			*refreshes = append(*refreshes, req.PostForm.Get("refresh_token"))
			return page(req, diStatus, diBody)
		}
		if req.Header.Get("Authorization") != "Bearer AT-new" {
			return page(req, http.StatusUnauthorized, "")
		}
		return page(req, http.StatusOK, `{"ok":true}`)
	})
	return transport, refreshes
}

func sharedClient(shared *SharedTokens, transport http.RoundTripper) *garminapi.Client {
	client := garminapi.NewClient(shared.Tokens(),
		garminapi.WithHTTPClient(&http.Client{Transport: transport}),
		garminapi.WithBaseURL("https://connectapi.garmin.com"))
	client.Refresher = shared.Refresh
	return client
}

func TestSharedTokensRefreshOnceForAllClients(t *testing.T) {
	transport, refreshes := sharedGarmin(t, http.StatusOK, `{"access_token":"AT-new","refresh_token":"RT-new","expires_in":3600}`)
	shared := NewSharedTokens(rejectedToken(), transport)
	get := func(name string, client *garminapi.Client) {
		if _, err := client.ConnectAPI(context.Background(), http.MethodGet, "/x", nil); err != nil {
			t.Fatalf("%s client: %v", name, err)
		}
	}
	first, second := sharedClient(shared, transport), sharedClient(shared, transport)
	get("first", first)
	get("second", second)
	get("later", sharedClient(shared, transport))
	if want := []string{"RT-secret-old"}; !slices.Equal(*refreshes, want) {
		t.Errorf("refresh tokens sent = %q, want %q", *refreshes, want)
	}
	if got := shared.Tokens(); got.OAuth2AccessToken != "AT-new" || got.OAuth2RefreshToken != "RT-new" {
		t.Errorf("shared tokens = %+v, want AT-new and RT-new", got)
	}
}

func TestSharedTokensKeepFailedRefresh(t *testing.T) {
	transport, refreshes := sharedGarmin(t, http.StatusBadRequest, `{"error":"invalid_grant"}`)
	shared := NewSharedTokens(rejectedToken(), transport)
	for i := range 2 {
		_, err := sharedClient(shared, transport).ConnectAPI(context.Background(), http.MethodGet, "/x", nil)
		if err == nil || !strings.HasPrefix(err.Error(), "token refresh after 401: ") || !strings.Contains(err.Error(), "invalid_grant") {
			t.Fatalf("client %d: err = %v, want the failed refresh with the invalid_grant answer", i, err)
		}
		if strings.Contains(err.Error(), "RT-secret") {
			t.Errorf("client %d: err = %v, want no refresh token in the message", i, err)
		}
	}
	if len(*refreshes) != 1 {
		t.Errorf("refreshes = %d, want 1", len(*refreshes))
	}
	if shared.Err() == nil {
		t.Error("Err() = nil, want the failed refresh")
	}
}
